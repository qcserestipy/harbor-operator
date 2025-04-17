package harborserverconfiguration

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	goharborv1 "github.com/goharbor/harbor-operator/apis/goharbor.io/v1beta1"
	"github.com/umisama/go-regexpcache"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-hsc,mutating=false,failurePolicy=fail,groups="goharbor.io",resources=harborserverconfigurations,verbs=create;update,sideEffects=None,admissionReviewVersions=v1beta1,versions=v1beta1,name=hsc.goharbor.io

// Important: Change *admission.Decoder to admission.Decoder (interface not pointer)
type Validator struct {
	Client  client.Client
	Log     logr.Logger
	decoder admission.Decoder
}

// Implement CustomValidator interface for newer controller-runtime
var _ admission.Handler = (*Validator)(nil)
var _ admission.CustomValidator = (*Validator)(nil)

func NewValidator(client client.Client, log logr.Logger, decoder admission.Decoder) *Validator {
	return &Validator{
		Client:  client,
		Log:     log,
		decoder: decoder,
	}
}

// Required ValidationCreate method for CustomValidator interface
func (h *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	hsc, ok := obj.(*goharborv1.HarborServerConfiguration)
	if !ok {
		return nil, fmt.Errorf("expected a HarborServerConfiguration but got a %T", obj)
	}

	for _, rule := range hsc.Spec.Rules {
		registryRegex := rule[:strings.LastIndex(rule, ",")+1]

		if _, err := regexpcache.Compile(registryRegex); err != nil {
			return nil, fmt.Errorf("%s can not be validated, %q is not a valid regular expression: %s", hsc.Name, registryRegex, err.Error())
		}
	}

	// Check for duplicate default configurations
	if hsc.Spec.Default {
		hscList := &goharborv1.HarborServerConfigurationList{}
		if err := h.Client.List(ctx, hscList); err != nil {
			return nil, fmt.Errorf("failed to list harbor server configurations: %w", err)
		}

		for _, harborConf := range hscList.Items {
			if harborConf.Name != hsc.Name && harborConf.Spec.Default {
				return nil, fmt.Errorf("%q can not be set as default, %q is the default harbor server configuration", hsc.Name, harborConf.Name)
			}
		}
	}

	return nil, nil
}

// Required ValidationUpdate method for CustomValidator interface
func (h *Validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	// Similar validation as ValidateCreate
	return h.ValidateCreate(ctx, newObj)
}

// Required ValidationDelete method for CustomValidator interface
func (h *Validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// No validation on delete
	return nil, nil
}

// Legacy Handle method - keep for compatibility
func (h *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	hsc := &goharborv1.HarborServerConfiguration{}

	// Use Decode instead of DecodeRaw
	err := h.decoder.Decode(req, hsc)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	for _, rule := range hsc.Spec.Rules {
		registryRegex := rule[:strings.LastIndex(rule, ",")+1]

		if _, err := regexpcache.Compile(registryRegex); err != nil {
			return admission.ValidationResponse(false, fmt.Sprintf("%s can not be validated, %q is not a valid regular expression: %s", hsc.Name, registryRegex, err.Error()))
		}
	}

	// Check for duplicate default configurations
	if hsc.Spec.Default {
		hscList := &goharborv1.HarborServerConfigurationList{}
		if err := h.Client.List(ctx, hscList); err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("failed to list harbor server configurations: %w", err))
		}

		for _, harborConf := range hscList.Items {
			if harborConf.Name != hsc.Name && harborConf.Spec.Default {
				return admission.ValidationResponse(false, fmt.Sprintf("%q can not be set as default, %q is the default harbor server configuration", hsc.Name, harborConf.Name))
			}
		}
	}

	return admission.Allowed("")
}

// Update for correct type
func (h *Validator) InjectDecoder(decoder admission.Decoder) error {
	h.decoder = decoder
	return nil
}

// Updated function with correct types
func (h *Validator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	decoder := admission.NewDecoder(mgr.GetScheme())
	h.decoder = decoder

	return ctrl.NewWebhookManagedBy(mgr).
		For(&goharborv1.HarborServerConfiguration{}).
		WithValidator(h).
		Complete()
}

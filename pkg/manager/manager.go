package manager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goharbor/harbor-operator/pkg/config"
	"github.com/goharbor/harbor-operator/pkg/factories/logger"
	nettracing "github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/ovh/configstore"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/transport"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	WebHookPort = 9443
	MetricsPort = 8080
	ProbePort   = 5000

	ManagerConfigKey = "operator"
)

func New(ctx context.Context, scheme *runtime.Scheme) (manager.Manager, error) {
	mgrConfig := ctrl.Options{
		Metrics: server.Options{
			BindAddress: fmt.Sprintf(":%d", MetricsPort),
		},
		HealthProbeBindAddress: fmt.Sprintf(":%d", ProbePort),
		LeaderElection:         false,
		Scheme:                 scheme,
	}

	webhookServer := ctrlwebhook.NewServer(ctrlwebhook.Options{
		Port: WebHookPort,
	})
	mgrConfig.WebhookServer = webhookServer

	item, err := configstore.Filter().
		Slice(ManagerConfigKey).
		Unmarshal(func() interface{} {
			// Duplicate mgrConfig
			c := mgrConfig

			return &c
		}).
		GetFirstItem()
	if err != nil {
		if !config.IsNotFound(err, ManagerConfigKey) {
			return nil, errors.Wrap(err, "cannot get configuration")
		}
	} else {
		c, err := item.Unmarshaled()
		if err != nil {
			return nil, errors.Wrap(err, "unable to get configuration")
		}

		mgrConfig = *c.(*ctrl.Options)
	}

	logger.Get(ctx).Info(
		"Manager initialized",
		"Webhook.Port", WebHookPort,
		"Metrics.Address", mgrConfig.Metrics.BindAddress,
		"Probe.Address", mgrConfig.HealthProbeBindAddress,
		"LeaderElection.Enabled", mgrConfig.LeaderElection,
		"LeaderElection.Namespace", mgrConfig.LeaderElectionNamespace,
		"LeaderElection.ID", mgrConfig.LeaderElectionID,
	)

	c, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get rest configuration")
	}

	c.WrapTransport = transport.Wrappers(func(rt http.RoundTripper) http.RoundTripper {
		return &nettracing.Transport{RoundTripper: rt}
	})

	mgr, err := ctrl.NewManager(c, mgrConfig)

	return mgr, errors.Wrap(err, "unable to get the manager")
}

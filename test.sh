#!/bin/bash

set -ex
make kustomize

kustomize build config/helm/crds | kubectl create -f -
operatorNamespace="harbor-operator-ns"
dockerImage="harbor-operator:dev_test"


make helm-install NAMESPACE="${operatorNamespace}" IMG=${dockerImage}
kubectl -n "${operatorNamespace}" wait --for=condition=Available deployment --all --timeout 300s

if ! time kubectl -n ${operatorNamespace} wait --for=condition=Available deployment --all --timeout 300s; then
  kubectl get all -n ${operatorNamespace}
  exit 1
fi

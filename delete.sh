#!/bin/bash
set -ex

helm uninstall minio-operator   
helm uninstall postgres-operator
helm uninstall redis-operator 
helm uninstall harbor-operator
crds=($(kubectl get customresourcedefinitions.apiextensions.k8s.io | grep -i goharbor.io | awk '{print $1}'))
for crd in $crds ; do kubectl delete customresourcedefinitions.apiextensions.k8s.io $crd ; done  

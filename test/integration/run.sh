#!usr/bin/env bash

# Setup Service Account
kubectl create -f deploy/service_account.yaml

# Setup RBAC
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml

# Setup the CRD
kubectl create -f deploy/crds/mumoshu_v1alpha1_awssecret_crd.yaml

# Deploy the app-operator
cat deploy/operator.yaml | sed -e 's/REPLACE_THIS_WITH_YOUR_REGION/ap-northeast-1/' | kubectl create -f -

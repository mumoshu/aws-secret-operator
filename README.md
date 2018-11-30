# aws-secret-operator

A Kubernetes operator that automatically creates and updates Kubernetes secrets according to what are stored in AWS Secrets Manager.

# Benefits

- Security from "decryption at rest". No need to create Kubernetes secrets by hand, helm, kustomize, or anything that requires you to decrypt the original secret on CI or your laptop
- Relies on Secrets Manager instead of SSM Parameter Store, so that the operator doesn't suffer from tight AWS API rate limit

# Usage

Let's say you've stored a secrets manager secret named `prod/mysecret` whose value is:

```json
{
  "foo": "bar"
}
```

Create a custom resource that points the secret:

`deploy/crds/mumoshu_v1alpha1_awssecret_cr.yaml`:

```yaml
apiVersion: mumoshu.github.io/v1alpha1
kind: AWSSecret
metadata:
  name: example
spec:
  secretsManagerSecretId: prod/mysecret
```

The operator then creates a Kubernetes secret named `example-awssecret` that looks like:

```json
{
  "kind": "Secret",
  "apiVersion": "v1",
  "metadata": {
    "name": "example-awssecret",
    "namespace": "default",
    "selfLink": "/api/v1/namespaces/default/secrets/test",
    "uid": "82ef45ee-4fdd-11e8-87bf-00e092001ba4",
    "resourceVersion": "25758",
    "creationTimestamp": "2018-05-04T20:55:43Z"
  },
  "data": {
    "foo": "YmFyCg=="
  },
  "type": "Opaque"
}
```

## Installation

```
# Setup Service Account
$ kubectl create -f deploy/service_account.yaml
# Setup RBAC
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
# Setup the CRD
$ kubectl create -f deploy/crds/mumoshu_v1alpha1_awssecret_crd.yaml
# Deploy the app-operator
$ kubectl create -f deploy/operator.yaml

# Create an AppService CR
# The default controller will watch for AppService objects and create a pod for each CR
$ kubectl create -f deploy/crds/mumoshu_v1alpha1_awssecret_cr.yaml

# Verify that a pod is created
$ kubectl get pod -l app=example-appservice
NAME                     READY     STATUS    RESTARTS   AGE
example-appservice-pod   1/1       Running   0          1m

# Cleanup
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_crd.yaml
```

# aws-secret-operator

A Kubernetes operator that automatically creates and updates Kubernetes secrets according to what are stored in AWS Secrets Manager.

`aws-secret-operator` custom resources maps AWS secrets to K8S secrets. Consider K8S secrets as just cached, latest AWS secrets.

# Benefits

**Security:**

By "decryption at rest". No need to create Kubernetes secrets by hand, helm, kustomize, or anything that requires you to decrypt the original secret on CI or your laptop

**Scalability:**

Relies on Secrets Manager instead of SSM Parameter Store for less chances being throttled by SSM's API rate limit.

Kubernetes secrets act as cache of Secrets Manager secrets, even number of API calls to Secrets Manager is minimum.

# Usage

Let's say you've stored a secrets manager secret named `prod/mysecret` whose value is:

```json
{
  "foo": "bar"
}
```

```console
$ aws secretsmanager get-secret-value \
    --secret-id prod/mysecret

$ aws secretsmanager create-secret \
    --name prod/mysecret

{
    "ARN": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:prod/mysecret-Ld0PUs",
    "Name": "prod/mysecret"
}

$ aws secretsmanager put-secret-value\
    --secret-id prod/mysecret \
    --secret-string '{"foo":"bar"}'
```

Let's see the `SecretId` and `VersionId` which uniquely identifies the secret:

```console
$ aws secretsmanager describe-secret --secret-id prod/mysecret
{
    "ARN": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:prod/mysecret-Ld0PUs",
    "Name": "prod/mysecret",
    "LastChangedDate": 1543636981.306,
    "LastAccessedDate": 1543622400.0,
    "VersionIdsToStages": {
        "c43e66cb-d0fe-44c5-9b7e-d450441a04be": [
            "AWSCURRENT"
        ]
    }
}
```

> Note that `aws-secret-operator` intentionally disallow omitting `VersionId` or specifying `VersionStage` as it makes you
difficult to trigger updates to Pods in response to AWS secrets changes.

Create a custom resource that points the secret:

`deploy/crds/mumoshu_v1alpha1_awssecret_cr.yaml`:

```yaml
apiVersion: mumoshu.github.io/v1alpha1
kind: AWSSecret
metadata:
  name: example
spec:
  stringDataFrom:
    secretsManagerSecretRef:
      secretId: prod/mysecret
      versionId: c43e66cb-d0fe-44c5-9b7e-d450441a04be
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

Now, your pod should either mount the generated secret as a volume, or set an environment variable from the secret.

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

## Why not...

1. Why not use AWS SSM Parameter Store as a primary source of secrets?

   **Pros:**

   Parameter Store has an efficient API to batch get multiple secrets sharing a same prefix.

   **Cons:**

   Its **API rate limit** is way too low. This has been discussed in several places in the Internet:

   - https://github.com/segmentio/chamber/issues/84#issuecomment-437728047
   - https://www.stackery.io/blog/serverless-secrets/
   - https://news.ycombinator.com/item?id=16758382

2. Why not use S3 as a primary source of secrets?

   **Pros:**

   Scalability. This project could have used S3 instead, because S3 supports efficient batch gets with filters by prefixes.
   An example of such project is [chamber](https://github.com/segmentio/chamber). chamber is a CLI wraps SSM Param Store and S3,
   [moving from Parameter Store to S3](https://github.com/segmentio/chamber/issues/84#issuecomment-438451470) due to the issue 1 explained above.

   **Cons:**

   Tooling. One of benefit of Secrets Manager over S3 is that in theory Secrets Manager has possibilities to deserve attentions of developers who
   who, for a better U/X, wraps Secrets Manager into a dedicated service/application to manager secrets.

   As using S3 for a primary storage for secrets is not a common practice, S3 can be said to have less possibilities to deserve.

## Use in combination with...

1. [sops](https://github.com/mozilla/sops) so that you can version-control the "latest master data" of secrets on Git repos.
   Each pull request that changes the master data results in CI workflows that deploys the master data to Secrets Manager.

## Alternatives

1. Use [bitnami-labs/sealed-secrets](https://github.com/bitnami-labs/sealed-secrets) when:
- You want something cloud-agnostic
- You are ok with [the theoretical potential that your private key can be stolen](https://github.com/bitnami-labs/sealed-secrets/issues/123)
2. Use [future-simple/helm-secrets](https://github.com/futuresimple/helm-secrets) when:
- You don't need to share secrets across apps/namespaces/environments. 

  Assuming you're going to manage encrypted secrets within a Git repo, sharing them requires you to copy and possible re-encrypt the secret to multiple git projects.

## Acknowledgements

This project is powered by [operator-framework](https://github.com/operator-framework/operator-sdk). Thanks for building the awesome framework :)

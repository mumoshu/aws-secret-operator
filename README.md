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
>
> Run a script like [update-aws-secret-ids](https://github.com/mumoshu/aws-secret-operator/blob/master/scripts/update-aws-secret-ids) in order to automate bumping VersionId in your configuration files.

Create a custom resource `awssecret` named `example` that points the SecretsManager secret:

`your_example_awssecret.yaml`:

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

The operator then creates a Kubernetes `secret` named `example` that looks like:

```json
{
  "kind": "Secret",
  "apiVersion": "v1",
  "metadata": {
    "name": "example",
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

Secrets manager has two secret formats: key/value (where the content is a plain JSON key-value map) and plain (a raw string). 
When the operator spots the key/value version, it expands all the key-value pairs into the nested secret keys:
```json
{
  "apiVersion": "v1",
  "data": {
    "password": "cGFzc3dvcmQ=",
    "user": "dXNlcg=="
  },
  "kind": "Secret",
  "metadata": {
    "creationTimestamp": "2020-06-29T10:35:32Z",
    "name": "example",
    "namespace": "default"
  },
  "type": "Opaque"
}
``` 

In case of plain secret format, the whole content of the secret is exposed as a `data` key of the secret:
```json
{
  "apiVersion": "v1",
  "data": {
    "data": "Zm9vXG5iYXIK"
  },
  "kind": "Secret",
  "metadata": {
    "creationTimestamp": "2020-06-29T10:35:32Z",
    "name": "example",
    "namespace": "default"
  },
  "type": "Opaque"
}
``` 

## Installation

```bash
# Setup RBAC (Namespaced, more secure)
$ kubectl create -f deploy/namespaced/rbac.yaml

# Setup RBAC (Cluster-scoped, easy to use)
$ kubectl create -f deploy/cluster_scoped/rbac.yaml

# Setup the CRD
$ kubectl create -f deploy/crds/mumoshu_v1alpha1_awssecret_crd.yaml

# Deploy the app-operator
# CAUTION: replace `ap-northeast-2` with your region e.g. us-west-2, and image tag
$ cat deploy/namespaced/deployment.yaml | sed -e 's/REPLACE_THIS_WITH_YOUR_REGION/ap-northeast-1/' | kubectl create -f -
# or cluster-scoped
$ cat deploy/cluster_scoped/deployment.yaml | sed -e 's/REPLACE_THIS_WITH_YOUR_REGION/ap-northeast-1/' | kubectl create -f -

# Verify that a pod is created
$ kubectl get pod -l app=aws-secret-operator

# Create an AWSSecret resource
$ kubectl create -f your_example_awssecret.yaml

# Verify that a secret is created
$ kubectl get secret

# Cleanup
$ kubectl delete -f your_example_awssecret.yaml
$ kubectl delete -f deploy/namespaced/deployment.yaml
$ kubectl delete -f deploy/namespaced/rbac.yaml
$ kubectl delete -f deploy/crds/mumoshu_v1alpha1_awssecret_crd.yaml
```

## Why not...

1. Why not use `helm-secrets` or `sops` in combination with e.g. `kubectl`?

   Because I don't want to give my CI the permission to decrypt secrets.

   Just for example, `helm-secrets` works by calling `sops` under the hood to decrypt encrypted values.yaml files, so that `helm` is able to sees the secrets as unencrypted values.yaml files. This implies that your CI system must have an AWS credential to call `kms:Decrypt` referring your KMS key. A compromised AWS credential can be used by a malicious user to decrypt those secrets. This is especially problematic when the CI system is a publicly hosted SaaS.

   `aws-secret-operator` prevents this kind of attacks by enabling your deployment pipeline usually seen in a CI system to submit just the reference to secrets stored in AWS Secrets Manager. The operator then decrypts them to produce Kubernetes `Secret` objects.

2. Why not use AWS SSM Parameter Store as a primary source of secrets?

   **Pros:**

   Parameter Store has an efficient API to batch get multiple secrets sharing a same prefix.

   **Cons:**

   Its **API rate limit** is way too low. This has been discussed in several places in the Internet:

   - https://github.com/segmentio/chamber/issues/84#issuecomment-437728047
   - https://www.stackery.io/blog/serverless-secrets/
   - https://news.ycombinator.com/item?id=16758382

3. Why not use S3 as a primary source of secrets?

   **Pros:**

   Scalability. This project could have used S3 instead, because S3 supports efficient batch gets with filters by prefixes.
   An example of such project is [chamber](https://github.com/segmentio/chamber). chamber is a CLI wraps SSM Param Store and S3,
   [moving from Parameter Store to S3](https://github.com/segmentio/chamber/issues/84#issuecomment-438451470) due to the issue 1 explained above.

   **Cons:**

   Tooling. One of benefit of Secrets Manager over S3 is that in theory Secrets Manager has possibilities to deserve attentions of developers who
   who, for a better U/X, wraps Secrets Manager into a dedicated service/application to manager secrets.

   As using S3 for a primary storage for secrets is not a common practice, S3 can be said to have less possibilities to deserve.

## Use in combination with...

1. Use [sops](https://github.com/mozilla/sops) in an independent CI/CD pipeline so you can version-control the "latest master data" of secrets on Git repos.
   Each pull request that changes the master data results in CI workflows that deploys the master data to Secrets Manager.

   Do provide only a KMS encryption permission to the CI system, so that the compromised AWS credential won't allow the attacker to decrypt your secrets.

## Alternatives

1. Use [bitnami-labs/sealed-secrets](https://github.com/bitnami-labs/sealed-secrets) when:
- You want something cloud-agnostic
- You are ok with [the theoretical potential that your private key can be stolen](https://github.com/bitnami-labs/sealed-secrets/issues/123)
2. Use [future-simple/helm-secrets](https://github.com/futuresimple/helm-secrets) when:
- You don't need to share secrets across apps/namespaces/environments. 

  Assuming you're going to manage encrypted secrets within a Git repo, sharing them requires you to copy and possible re-encrypt the secret to multiple git projects.

## Acknowledgements

This project is powered by [operator-framework](https://github.com/operator-framework/operator-sdk). Thanks for building the awesome framework :)

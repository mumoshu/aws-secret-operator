package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operator "github.com/mumoshu/aws-secret-operator/api/mumoshu/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval = time.Second * 1
	timeout       = time.Second * 3
)

// This runs a series of AWS API calls and K8s API calls
// to see if the controller is working in concert with AWS.
// To allow the AWS session to call required AWS APIs, you need
// an AWS session bound to an user or a role with a policy that looks like the below.
//
// {
//     "Version": "2012-10-17",
//     "Statement": [
//         {
//             "Sid": "VisualEditor0",
//             "Effect": "Allow",
//             "Action": [
//                 "secretsmanager:GetSecretValue",
//                 "secretsmanager:CreateSecret",
//                 "secretsmanager:DeleteSecret",
//                 "secretsmanager:UpdateSecret"
//             ],
//             "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:aws-secret-operator-ci/mySecret-testns-?????-??????"
//         }
//     ]
// }
//
// Note that the first "?????" is for the random suffix we add to the test namespace,
// and the second "??????"(6 characters) are for the random suffix added to ARNs by AWS SecretsManager API
// For the latter, see the below for more information.
// https://docs.aws.amazon.com/secretsmanager/latest/userguide/auth-and-access_examples.html#auth-and-access_examples_wildcard
func awsSecretTest(ctx context.Context, client client.Client, namespace string) error {
	log := logf.Log

	sess := session.Must(session.NewSession())
	sm := secretsmanager.New(sess)

	secretID := fmt.Sprintf("aws-secret-operator-ci/%s-%s", "mySecret", namespace)
	secretValueV1 := `{"value":"v1value"}`
	secretValueV2 := `{"value":"v2value"}`

	secretV1, err := sm.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretID),
		SecretString: aws.String(secretValueV1),
	})
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	arn := secretV1.ARN
	if arn != nil {
		log = log.WithValues("arn", *arn)
	}

	versionIDV1 := *secretV1.VersionId

	defer func() {
		_, err := sm.DeleteSecret(&secretsmanager.DeleteSecretInput{
			ForceDeleteWithoutRecovery: aws.Bool(true),
			SecretId:                   aws.String(secretID),
		})

		if err != nil {
			log.Error(err, "deleting secret on aws")
		}
	}()

	secretV2, err := sm.UpdateSecret(&secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(secretID),
		SecretString: aws.String(secretValueV2),
	})
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	versionIDV2 := *secretV2.VersionId

	exampleAWSSecret := &operator.AWSSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-secret",
			Namespace: namespace,
		},
		Spec: operator.AWSSecretSpec{
			Type: "Opaque",
			StringDataFrom: operator.StringDataFrom{
				SecretsManagerSecretRef: operator.SecretsManagerSecretRef{
					SecretId:  secretID,
					VersionId: versionIDV1,
				},
			},
		},
	}

	err = client.Create(ctx, exampleAWSSecret)
	if err != nil {
		return err
	}

	err = waitForSecret(ctx, log, client, "creation", namespace, "example-secret", map[string]string{"value": "v1value"}, nil, nil, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = client.Get(ctx, types.NamespacedName{Name: "example-secret", Namespace: namespace}, exampleAWSSecret)
	if err != nil {
		return err
	}

	exampleAWSSecret.Spec.StringDataFrom.SecretsManagerSecretRef.VersionId = versionIDV2
	err = client.Update(ctx, exampleAWSSecret)
	if err != nil {
		return err
	}

	// wait for example-secret to be updated
	err = waitForSecret(ctx, log, client, "update", namespace, "example-secret", map[string]string{"value": "v2value"}, nil, nil, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = client.Get(ctx, types.NamespacedName{Name: "example-secret", Namespace: namespace}, exampleAWSSecret)
	if err != nil {
		return err
	}

	labels := map[string]string{"label1": "labelv1"}
	annotations := map[string]string{"annotation1": "annotationv1"}

	exampleAWSSecret.Spec.Metadata = &operator.SecretMeta{
		Labels:      labels,
		Annotations: annotations,
	}
	err = client.Update(ctx, exampleAWSSecret)
	if err != nil {
		return err
	}

	log.Info("Updated awsSecret", "spec.metadata", exampleAWSSecret.Spec.Metadata)

	// wait for example-secret to have the custom label and annotation
	err = waitForSecret(ctx, log, client, "custom label and annotation", namespace, "example-secret", map[string]string{"value": "v2value"}, labels, annotations, retryInterval, timeout)
	if err != nil {
		return err
	}

	return nil
}

func waitForSecret(ctx context.Context, log logr.Logger, client client.Client, desc, namespace, name string,
	expectedKVs map[string]string,
	labels, annotations map[string]string,
	retryInterval, timeout time.Duration) error {

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		var secret corev1.Secret
		getErr := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &secret)
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				log.Info("Waiting for availability of secret", "name", name, "namespace", namespace)
				return false, nil
			}
			return false, getErr
		}

		for k, want := range expectedKVs {
			bs := secret.Data[k]
			got := string(bs)

			if want != got {
				err := fmt.Errorf("unexpected value for key %s: want %v, got %v", k, want, got)
				log.Error(err, "unexpected secret data")
				return true, err
			}
		}

		if want, got := len(labels), len(secret.Labels); want != got {
			log.Info("Still waiting for labels to be updated", "want", want, "got", got, "observed", secret.Labels)
			return false, nil
		}
		for k, want := range labels {
			if got := secret.Labels[k]; want != got {
				log.Info("Still waiting for label to be updated", "label", k, "want", want, "got", got)
				return false, nil
			}
		}

		if want, got := len(annotations), len(secret.Annotations); want != got {
			log.Info("Still waiting for annotations to be updated", "key", want, "got", got)
			return false, nil
		}
		for k, want := range annotations {
			if got := secret.Annotations[k]; want != got {
				log.Info("Still waiting for annotation to be updated", "key", k, "want", want, "got", got)
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed while waiting for %s: %w", desc, err)
	}
	log.Info("Secret available", "name", name)
	return nil
}

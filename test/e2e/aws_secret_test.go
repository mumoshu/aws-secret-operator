package e2e

import (
	goctx "context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"testing"
	"time"

	"github.com/mumoshu/aws-secret-operator/pkg/apis"
	operator "github.com/mumoshu/aws-secret-operator/pkg/apis/mumoshu/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestAWSSecret(t *testing.T) {
	awsSecretList := &operator.AWSSecretList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, awsSecretList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("aws-secret-group", func(t *testing.T) {
		t.Run("Suite1", AWSSecretSuite)
	})
}

func awsSecretTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	sess := session.Must(session.NewSession())
	sm := secretsmanager.New(sess)

	secretID := fmt.Sprintf("%s-%s", namespace, "mySecret")
	secretValueV1 := `{"value":"v1value"}`
	secretValueV2 := `{"value":"v2value"}`

	secretV1, err := sm.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretID),
		SecretString: aws.String(secretValueV1),
	})
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	versionIDV1 := *secretV1.VersionId

	defer func() {
		_, err := sm.DeleteSecret(&secretsmanager.DeleteSecretInput{
			ForceDeleteWithoutRecovery: aws.Bool(true),
			SecretId:                   aws.String(secretID),
		})

		if err != nil {
			t.Logf("deleting secret on aws: %v", err)
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
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), exampleAWSSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}
	// wait for example-memcached to reach 3 replicas
	err = waitForSecret(t, f.KubeClient, namespace, "example-secret", map[string]string{"value": "v1value"}, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "example-secret", Namespace: namespace}, exampleAWSSecret)
	if err != nil {
		return err
	}
	exampleAWSSecret.Spec.StringDataFrom.SecretsManagerSecretRef.VersionId = versionIDV2
	err = f.Client.Update(goctx.TODO(), exampleAWSSecret)
	if err != nil {
		return err
	}

	// wait for example-secret to be updated
	return waitForSecret(t, f.KubeClient, namespace, "example-secret", map[string]string{"value": "v2value"}, retryInterval, timeout)
}

func waitForSecret(t *testing.T, kubeclient kubernetes.Interface, namespace, name string,
	expectedKVs map[string]string,
	retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		secret, getErr := kubeclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				t.Logf("Waiting for availability of corev1 Secret: %s in Namespace: %s \n", name, namespace)
				return false, nil
			}
			return false, getErr
		}

		for k, want := range expectedKVs {
			bs := secret.Data[k]
			got := string(bs)

			if want != got {
				return true, fmt.Errorf("unexpected value for key %s: want %v, got %v", k, want, got)
			}
		}

		return true, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Secret %s available\n", name)
	return nil
}

func AWSSecretSuite(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "aws-secret-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = awsSecretTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

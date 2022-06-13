package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	zaplib "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testEnvironment struct {
	Namespace *corev1.Namespace
}

// SetupIntegrationTest will set up a testing environment.
// This includes:
// * creating a Namespace to be used during the test
// * starting all the reconcilers
// * stopping all the reconcilers after the test ends
// Call this function at the start of each of your tests.
func SetupIntegrationTest(ctx2 context.Context) *testEnvironment {
	var ctx context.Context
	var cancel func()
	ns := &corev1.Namespace{}

	env := &testEnvironment{
		Namespace: ns,
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(ctx2)
		*ns = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "testns-" + randStringRunes(5)},
		}

		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

		logger := zap.New(func(o *zap.Options) {
			// For example, --log-level=debug a.k.a --log-level=-1 maps to zaplib.DebugLevel, which is associated to logr's V(1)
			// --log-level=-2 maps the specific custom log level that is associated to logr's V(2).
			level := zapcore.Level(-2)
			atomicLevel := zaplib.NewAtomicLevelAt(level)
			o.Level = &atomicLevel
			o.TimeEncoder = zapcore.TimeEncoderOfLayout(time.RFC3339)
		})

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Namespace: ns.Name,
			Logger:    logger,
		})
		Expect(err).NotTo(HaveOccurred(), "failed to create manager")

		controllerName := func(name string) string {
			return fmt.Sprintf("%s%s", ns.Name, name)
		}

		awsSecretController := &AWSSecretController{
			Name:   controllerName("awssecret"),
			Client: mgr.GetClient(),
			Scheme: scheme.Scheme,
			Log:    &logger,
		}
		err = awsSecretController.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred(), "failed to setup runner controller")

		go func() {
			defer GinkgoRecover()

			err := mgr.Start(ctx)
			Expect(err).NotTo(HaveOccurred(), "failed to start manager")
		}()
	})

	AfterEach(func() {
		defer cancel()

		err := k8sClient.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
	})

	return env
}

var _ = Context("INTEGRATION: Inside of a new namespace", func() {
	ctx := context.TODO()
	env := SetupIntegrationTest(ctx)
	ns := env.Namespace

	Describe("when no existing resources exist", func() {

		It("create Kubernetes secret from AWS Secrets Manager secret", func() {
			err := awsSecretTest(context.Background(), k8sClient, ns.Name)

			Expect(err).ToNot(HaveOccurred())
		})

	})
})

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

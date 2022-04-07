package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/mumoshu/aws-secret-operator/api"
	"github.com/mumoshu/aws-secret-operator/controllers"
	"github.com/operator-framework/operator-lib/leader"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	zaplib "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	log = logf.Log.WithName("cmd")

	logLevel string
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

type OperateOpts struct {
	ConfigMapName      string
	ConfigMapNamespace string
	WatchNamespace     string
}

var opts = OperateOpts{}

var Root = &cobra.Command{
	Use:   "aws-secret-operator",
	Short: "Creates and updates Kubernetes secrets based on secrets stored in AWS Secrets Manager",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run( /*opts*/ )
	},
}

func init() {
	Root.Flags().StringVar(&logLevel, "log-level", LogLevelDebug, `The verbosity of the logging. Valid values are "debug", "info", "warn", "error". Defaults to "debug".`)
	Root.Flags().StringVar(&opts.ConfigMapName, "configmap-name", "falco-operator", "the name of the configmap to which this operator writes the concatenated falco rules")
	Root.Flags().StringVarP(&opts.ConfigMapNamespace, "configmap-namespace", "n", "kube-system", "namespace in which falco and falco-operator are running")
	Root.Flags().StringVarP(&opts.WatchNamespace, "watch-namespace", "w", "", "namespaces on which the operator watches for changes")
}

func run() error {
	flag.Parse()

	logger := zap.New(func(o *zap.Options) {
		switch logLevel {
		case LogLevelDebug:
			o.Development = true
			lvl := zaplib.NewAtomicLevelAt(zaplib.DebugLevel) // maps to logr's V(1)
			o.Level = &lvl
		case LogLevelInfo:
			lvl := zaplib.NewAtomicLevelAt(zaplib.InfoLevel)
			o.Level = &lvl
		case LogLevelWarn:
			lvl := zaplib.NewAtomicLevelAt(zaplib.WarnLevel)
			o.Level = &lvl
		case LogLevelError:
			lvl := zaplib.NewAtomicLevelAt(zaplib.ErrorLevel)
			o.Level = &lvl
		default:
			// We use bitsize of 8 as zapcore.Level is a type alias to int8
			levelInt, err := strconv.ParseInt(logLevel, 10, 8)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse --log-level=%s: %v", logLevel, err)
				os.Exit(1)
			}

			// For example, --log-level=debug a.k.a --log-level=-1 maps to zaplib.DebugLevel, which is associated to logr's V(1)
			// --log-level=-2 maps the specific custom log level that is associated to logr's V(2).
			level := zapcore.Level(levelInt)
			atomicLevel := zaplib.NewAtomicLevelAt(level)
			o.Level = &atomicLevel
		}
		o.TimeEncoder = zapcore.TimeEncoderOfLayout(time.RFC3339)
	})

	logf.SetLogger(logger)

	printVersion()

	namespace, err := getWatchNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to get watch namespace")
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), "aws-secret-operator-lock")
	if err != nil {
		return errors.Wrap(err, "failed to became the leader")
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		return errors.Wrap(err, "failed to init manager")
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return errors.Wrap(err, "failed to add apis to scheme")
	}

	// Setup all Controllers

	awsSecretController := &controllers.AWSSecretController{
		Scheme: mgr.GetScheme(),
		Client: mgr.GetClient(),
	}

	if err := awsSecretController.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "failed to add controller(s) to manager")
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "manager exited non-zero")
	}

	return nil
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
// See https://github.com/operator-framework/operator-sdk/blob/a05f966806f1beaac3c45d28072f107a502ab253/website/content/en/docs/building-operators/golang/operator-scope.md#configuring-namespace-scoped-operators
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

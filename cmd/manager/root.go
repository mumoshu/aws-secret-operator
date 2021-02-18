package main

import (
	"os"

	"github.com/spf13/cobra"
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
	Root.Flags().StringVar(&opts.ConfigMapName, "configmap-name", "falco-operator", "the name of the configmap to which this operator writes the concatenated falco rules")
	Root.Flags().StringVarP(&opts.ConfigMapNamespace, "configmap-namespace", "n", "kube-system", "namespace in which falco and falco-operator are running")
	Root.Flags().StringVarP(&opts.WatchNamespace, "watch-namespace", "w", "", "namespaces on which the operator watches for changes")
}

func main() {
	if err := Root.Execute(); err != nil {
		os.Exit(1)
	}
}

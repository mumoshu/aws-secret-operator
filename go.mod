module github.com/mumoshu/aws-secret-operator

go 1.14

require (
	github.com/aws/aws-sdk-go v1.34.31
	github.com/google/go-cmp v0.4.0
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	k8s.io/api v0.18.5
	k8s.io/apimachinery v0.18.5
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.5
)

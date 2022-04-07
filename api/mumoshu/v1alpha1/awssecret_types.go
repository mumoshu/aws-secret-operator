package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AWSSecretSpec defines the desired state of AWSSecret
type AWSSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// DataFrom data field is used to store arbitrary data, encoded using base64.
	DataFrom DataFrom `json:"dataFrom,omitempty"`

	// StringDataFrom stringData field is provided for convenience, and allows you to provide
	// secret data as unencoded strings.
	StringDataFrom StringDataFrom `json:"stringDataFrom,omitempty"`

	// Used to facilitate programmatic handling of secret data.
	// +optional
	Type corev1.SecretType `json:"type,omitempty"`
}

// StringDataFrom defines how the resulting Secret's `stringData` is built
type StringDataFrom struct {
	SecretsManagerSecretRef SecretsManagerSecretRef `json:"secretsManagerSecretRef,omitempty"`
}

// DataFrom defines how the resulting Secret's `data` is built
type DataFrom struct {
	SecretsManagerSecretRef SecretsManagerSecretRef `json:"secretsManagerSecretRef,omitempty"`
}

// SecretsManagerSecretRef defines from which SecretsManager Secret the Kubernetes secret is built
// See https://docs.aws.amazon.com/secretsmanager/latest/userguide/terms-concepts.html for the concepts
type SecretsManagerSecretRef struct {
	// SecretId is the SecretId a.k.a `--secret-id` of the SecretsManager secret version
	SecretId string `json:"secretId,omitempty"`
	// VersionIdis the VersionId a.k.a `--version-id` of the SecretsManager secret version
	VersionId string `json:"versionId,omitempty"`
}

// AWSSecretStatus defines the observed state of AWSSecret
type AWSSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSecret is the Schema for the awssecrets API
// +k8s:openapi-gen=true
type AWSSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSecretSpec   `json:"spec,omitempty"`
	Status AWSSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSecretList contains a list of AWSSecret
type AWSSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWSSecret{}, &AWSSecretList{})
}

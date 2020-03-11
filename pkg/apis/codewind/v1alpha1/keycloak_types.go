package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeycloakSpec defines the desired state of Keycloak
type KeycloakSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// WorkspaceID : A workspace reference ID for this instance
	WorkspaceID string `json:"workspaceID"`
	// DeploymentReference : A reference name for this instance
	DeploymentReference string `json:"deploymentRef"`
}

// KeycloakStatus defines the observed state of Keycloak
type KeycloakStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Phase     string `json:"phase"`
	AccessURL string `json:"url"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Keycloak is the Schema for the keycloaks API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=keycloaks,scope=Namespaced
// +kubebuilder:printcolumn:name="Deployment",type="string",JSONPath=".spec.deploymentRef",priority=0,description="Deployment reference name"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".metadata.namespace",priority=0,description="Deployment namespace"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0,description="Age of the resource"
// +kubebuilder:printcolumn:name="Access",type="string",JSONPath=".status.url",priority=0,description="Exposed route"
type Keycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KeycloakSpec   `json:"spec,omitempty"`
	Status            KeycloakStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeycloakList contains a list of Keycloak
type KeycloakList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Keycloak `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keycloak{}, &KeycloakList{})
}

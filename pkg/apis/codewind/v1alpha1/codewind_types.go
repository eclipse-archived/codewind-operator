package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CodewindSpec defines the desired state of Codewind
type CodewindSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// WorkspaceID: ident of this deployment
	WorkspaceID string `json:"workspaceID"`
	// KeycloakDeployment : name of the keycloak deployment used by this instance of codewind
	KeycloakDeployment string `json:"keycloakDeployment"`
	// Developer username assigned to this instance
	Username string `json:"username"`
	// Ingress domain
	IngressDomain string `json:"ingressDomain"`
	// Codewind Storage size
	StorageSize string `json:"storageSize"`
}

// CodewindStatus defines the observed state of Codewind
type CodewindStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Keycloak access URL
	AuthURL string `json:"authURL"`
	// Exposed Ingress of Codewind (Gatekeeper)
	AccessURL string `json:"accessURL"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Codewind is the Schema for the codewinds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=codewinds,scope=Namespaced
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username",priority=0,description="Deployment reference name"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".metadata.namespace",priority=0,description="Deployment namespace"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0,description="Age of the resource"
// +kubebuilder:printcolumn:name="Auth",type="string",JSONPath=".spec.keycloakDeployment",priority=0,description="Deployment reference name"
// +kubebuilder:printcolumn:name="AccessURL",type="string",JSONPath=".status.accessURL",priority=0,description="Exposed route"
type Codewind struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodewindSpec   `json:"spec,omitempty"`
	Status CodewindStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodewindList contains a list of Codewind
type CodewindList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Codewind `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Codewind{}, &CodewindList{})
}

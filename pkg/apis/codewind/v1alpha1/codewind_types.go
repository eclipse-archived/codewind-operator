/*******************************************************************************
 * Copyright (c) 2020 IBM Corporation and others.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CodewindSpec defines the desired state of Codewind
type CodewindSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// KeycloakDeployment : name of the keycloak deployment used by this instance of codewind
	// +kubebuilder:validation:Pattern=^[A-Za-z0-9/-]*$
	KeycloakDeployment string `json:"keycloakDeployment"`

	// Developer username assigned to this instance
	// +kubebuilder:validation:Pattern=^[A-Za-z0-9/-]*$
	Username string `json:"username"`

	// Codewind Storage size
	// +kubebuilder:validation:Pattern=[0-9]*Gi$
	StorageSize string `json:"storageSize"`

	// LogLevel within pods
	LogLevel string `json:"logLevel"`
}

// CodewindStatus defines the observed state of Codewind
type CodewindStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Keycloak access URL
	AuthURL string `json:"authURL"`

	// Exposed Ingress of Codewind (Gatekeeper)
	AccessURL string `json:"accessURL"`

	// Keycloak Configuration status
	KeycloakStatus string `json:"keycloakStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Codewind is the Schema for the codewinds API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=codewinds,scope=Namespaced
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username",priority=0,description="Deployment reference name"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".metadata.namespace",priority=0,description="Deployment namespace"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0,description="Age of the resource"
// +kubebuilder:printcolumn:name="Keycloak",type="string",JSONPath=".spec.keycloakDeployment",priority=0,description="Deployment reference name"
// +kubebuilder:printcolumn:name="Registration",type="string",JSONPath=".status.keycloakStatus",priority=0,description="Keycloak configuration status"
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

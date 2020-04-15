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

package defaults

const (
	// PrefixCodewindPerformance : Codewind performance application
	PrefixCodewindPerformance = "codewind-performance"

	// PrefixCodewindPFE : Codewind pfe application
	PrefixCodewindPFE = "codewind-pfe"

	// PrefixCodewindGatekeeper : Codewind-gatekeeper application
	PrefixCodewindGatekeeper = "codewind-gatekeeper"

	// PrefixCodewindKeycloak : Codewind-keycloak application
	PrefixCodewindKeycloak = "codewind-keycloak"
)

const (
	// VersionNum : Operator version number
	VersionNum = "0.0.1"

	// KeycloakImage is the docker image that will be used in the Codewind-Keycloak pod
	KeycloakImage = "eclipse/codewind-keycloak-amd64"

	// KeycloakImageTag is the Image tag used by Keycloak
	KeycloakImageTag = "0.11.0"

	// CodewindImage is the docker image that will be used in the Codewind-pfe pod
	CodewindImage = "eclipse/codewind-pfe-amd64"

	// CodewindImageTag is the Image tag used by Codewind
	CodewindImageTag = "0.11.0"

	// CodewindPerformanceImage is the docker image that will be used in the Codewind-Performance pod
	CodewindPerformanceImage = "eclipse/codewind-performance-amd64"

	// CodewindPerformanceImageTag is the Image tag used by Codewind
	CodewindPerformanceImageTag = "0.11.0"

	// CodewindGatekeeperImage is the docker image that will be used in the Codewind-Gatekeeper pod
	CodewindGatekeeperImage = "eclipse/codewind-gatekeeper-amd64"

	// CodewindGatekeeperImageTag is the Image tag used by Codewind
	CodewindGatekeeperImageTag = "0.11.0"

	// CodewindAuthRealm : Codewind security realm within Keycloak
	CodewindAuthRealm = "codewind"

	// PFEContainerPort is the port at which Codewind PFE is exposed
	PFEContainerPort = 9191

	// PerformanceContainerPort is the port at which the Performance dashboard is exposed
	PerformanceContainerPort = 9095

	// KeycloakContainerPort is the port at which Keycloak is exposed
	KeycloakContainerPort = 8080

	// GatekeeperContainerPort is the port at which the Gatekeeper is exposed
	GatekeeperContainerPort = 9096

	// CodewindRoleBindingNamePrefix will include the workspaceID when deployed
	CodewindRoleBindingNamePrefix = "codewind-rolebinding"

	// CodewindTektonClusterRoleBindingName : Tekton, cluster role binding
	CodewindTektonClusterRoleBindingName = "codewind-tekton-rolebinding"

	// CodewindTektonClusterRolesName : Tekton, cluster role
	CodewindTektonClusterRolesName = "codewind-tekton"

	// CodewindRolesName will include the workspaceID when deployed
	CodewindRolesName = "eclipse-codewind-" + VersionNum
)

const (
	// ConstKeycloakConfigStarted : Keycloak config started
	ConstKeycloakConfigStarted = "Started"

	// ConstKeycloakConfigReady : Keycloak config completed
	ConstKeycloakConfigReady = "Complete"

	// ROKSStorageClass references the storage class to use on ROKS
	ROKSStorageClass = "ibmc-file-bronze"

	// ROKSStorageClassGID references the storage class to use on ROKS
	ROKSStorageClassGID = "ibmc-file-bronze-gid"

	// ConfigMapLocation : Codewind Operator config map defaults
	ConfigMapLocation = "deploy/codewind-configmap.yaml"

	// OperatorConfigMapName : Codewind operator config map name
	OperatorConfigMapName = "codewind-operator"
)

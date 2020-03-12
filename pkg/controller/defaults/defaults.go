package defaults

const (
	// VersionNum
	VersionNum = "0.0.1"

	// KeycloakImage is the docker image that will be used in the Codewind-Keycloak pod
	KeycloakImage = "eclipse/codewind-keycloak-amd64"

	// KeycloakImageTag is the Image tag used by Keycloak
	KeycloakImageTag = "0.9.0"

	// CodewindImage is the docker image that will be used in the Codewind-pfe pod
	CodewindImage = "eclipse/codewind-pfe-amd64"

	// CodewindImageTag is the Image tag used by Codewind
	CodewindImageTag = "0.9.0"

	// CodewindPerformanceImage is the docker image that will be used in the Codewind-Performance pod
	CodewindPerformanceImage = "eclipse/codewind-performance-amd64"

	// CodewindPerformanceImageTag is the Image tag used by Codewind
	CodewindPerformanceImageTag = "0.9.0"

	// CodewindGatekeeperImage is the docker image that will be used in the Codewind-Gatekeeper pod
	CodewindGatekeeperImage = "eclipse/codewind-gatekeeper-amd64"

	// CodewindGatekeeperImageTag is the Image tag used by Codewind
	CodewindGatekeeperImageTag = "0.9.0"

	// PFEStorageSize is the size of the PVC used by Codewind PFE
	PFEStorageSize = "10Gi"

	// CodewindAuthRealm : Codewind security realm within Keycloak
	CodewindAuthRealm = "codewind"

	// PFEContainerPort is the port at which Codewind-PFE is exposed
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
	ConstKeycloakConfigReady = "Completed"

	// ROKSStorageClass references the storage class to use on ROKS
	ROKSStorageClass = "ibmc-file-bronze"

	// ROKSStorageClassGID references the storage class to use on ROKS
	ROKSStorageClassGID = "ibmc-file-bronze-gid"

	// ConfigMapLocation : Codewind Operator config map defaults
	ConfigMapLocation = "deploy/codewind-configmap.yaml"

	// OperatorConfigMapName : Codewind operator config map name
	OperatorConfigMapName = "codewind-operator"
)

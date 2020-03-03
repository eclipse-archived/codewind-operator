package defaults

const (
	// PFEContainerPort is the port at which Codewind-PFE is exposed
	PFEContainerPort = 9191

	// PerformanceContainerPort is the port at which the Performance dashboard is exposed
	PerformanceContainerPort = 9095

	// KeycloakContainerPort is the port at which Keycloak is exposed
	KeycloakContainerPort = 8080

	// GatekeeperContainerPort is the port at which the Gatekeeper is exposed
	GatekeeperContainerPort = 9096

	// KeycloakImage is the docker image that will be used in the Codewind-Keycloak pod
	KeycloakImage = "eclipse/codewind-keycloak-amd64"

	// KeycloakImageTag is the Image tag used by Keycloak
	KeycloakImageTag = "0.9.0"

	// CodewindImage is the docker image that will be used in the Codewind-Keycloak pod
	CodewindImage = "eclipse/codewind-pfe-amd64"

	// CodewindImageTag is the Image tag used by Codewind
	CodewindImageTag = "0.9.0"
)

// GetCurrentIngressDomain : the current ingress domain of the cluster
func GetCurrentIngressDomain() string {
	return "10.100.111.145.nip.io"
}

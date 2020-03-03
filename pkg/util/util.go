package kubeutil

import (
	"os"

	logr "github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// DetectOpenShift determines if we're running on an OpenShift cluster
// From https://github.com/eclipse/che-operator/blob/2f639261d8b5416b2934591e12925ee0935814dd/pkg/util/util.go#L63
func DetectOpenShift(config *rest.Config) bool {
	clientConfig = config.GetConfig()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
	if err != nil {
		logr.Errorf("Unable to detect if running on OpenShift: %v\n", err)
		os.Exit(1)
	}
	apiList, err := discoveryClient.ServerGroups()
	if err != nil {
		logr.Errorf("Error attempting to retrieve list of API Groups: %v\n", err)
		os.Exit(1)
	}
	apiGroups := apiList.Groups
	for _, group := range apiGroups {
		if group.Name == "route.openshift.io" {
			return true
		}
	}
	return false
}

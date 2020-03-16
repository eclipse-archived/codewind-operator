package codewind

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/security"
	util "github.com/eclipse/codewind-operator/pkg/util"
	"github.com/go-logr/logr"
	v1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_codewind")

// Add creates a new Codewind Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	reconciler := &ReconcileCodewind{client: mgr.GetClient(), scheme: mgr.GetScheme()}
	operatorNamespace, _ := k8sutil.GetOperatorNamespace()
	if operatorNamespace == "" {
		operatorNamespace = "codewind"
	}
	return reconciler
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	// Disable certificate validation checking
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// isOpenshift, _, err := util.DetectOpenShift()
	// if err != nil {
	// 	log.Error(err, "Error detecting platfom", "")
	// }
	// log.Info("Running on Openshift", "status", isOpenshift)

	// Create a new controller
	c, err := controller.New("codewind-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Codewind
	err = c.Watch(&source.Kind{Type: &codewindv1alpha1.Codewind{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources and requeue the owner Codewind
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	})
	if err != nil {
		return err
	}

	// // Secret
	// if err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &codewindv1alpha1.Codewind{},
	// }); err != nil {
	// 	return err
	// }

	// // service
	// if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &codewindv1alpha1.Codewind{},
	// }); err != nil {
	// 	return err
	// }

	// // service account
	// err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &codewindv1alpha1.Codewind{},
	// })
	// if err != nil {
	// 	return err
	// }

	// // deployment
	// err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &codewindv1alpha1.Codewind{},
	// })
	// if err != nil {
	// 	return err
	// }

	// // persistent volume claim
	// err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &codewindv1alpha1.Codewind{},
	// })
	// if err != nil {
	// 	return err
	// }

	// // Routes
	// if isOpenshift {
	// 	err = c.Watch(&source.Kind{Type: &v1.Route{}}, &handler.EnqueueRequestForOwner{
	// 		IsController: true,
	// 		OwnerType:    &codewindv1alpha1.Codewind{},
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// } else {
	// 	// Ingress
	// 	err = c.Watch(&source.Kind{Type: &extv1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
	// 		IsController: true,
	// 		OwnerType:    &codewindv1alpha1.Codewind{},
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

// blank assignment to verify that ReconcileCodewind implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCodewind{}

// ReconcileCodewind reconciles a Codewind object
type ReconcileCodewind struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Codewind object and makes changes based on the state read
// and what is in the Codewind.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCodewind) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	isOpenshift, _, err := util.DetectOpenShift()
	if err != nil {
		reqLogger.Error(err, "An error occurred when detecting current infrastructure", "")
	}

	// Fetch the config map
	operatorNamespace := util.GetOperatorNamespace()
	operatorConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.OperatorConfigMapName, Namespace: operatorNamespace}, operatorConfigMap)
	if err != nil {
		reqLogger.Error(err, "Unable to read config map. Ensure one has been created in the same namespace as the operator", "name", defaults.OperatorConfigMapName)
		return reconcile.Result{}, err
	}

	// Get fields we need from the configmap
	ingressDomain := operatorConfigMap.Data["ingressDomain"]
	storageSize := operatorConfigMap.Data["storageCodewindSize"]
	defaultRealm := operatorConfigMap.Data["defaultRealm"]

	// get the operator config map
	configMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-config", Namespace: ""}, configMap)
	if err == nil {
		reqLogger.Error(err, "Codewind Operator config map is not available, aborting reconcile", "")
		return reconcile.Result{}, err
	}

	// Use ROKSStorageClass when it is available
	storageClassName := ""
	storageClassDef := &storagev1.StorageClass{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.ROKSStorageClass, Namespace: ""}, storageClassDef)
	if err == nil {
		reqLogger.Info("Using storageclass", "name", defaults.ROKSStorageClass)
		storageClassName = defaults.ROKSStorageClass
	}

	// Fetch the Codewind instance
	codewind := &codewindv1alpha1.Codewind{}
	err = r.client.Get(context.TODO(), request.NamespacedName, codewind)
	if err != nil {
		if k8serr.IsNotFound(err) {
			//Codewind resource not found. Ignoring since it must be deleted
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Codewind instance")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Cluster roles already exist, if not create new ones
	clusterRoles := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.CodewindRolesName, Namespace: ""}, clusterRoles)

	if err != nil && k8serr.IsNotFound(err) {
		newClusterRoles := r.clusterRolesForCodewind(codewind)
		reqLogger.Info("Creating a new Codewind cluster roles", "Namespace", "", "Name", newClusterRoles.Name)
		err = r.client.Create(context.TODO(), newClusterRoles)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind cluster roles.", "Namespace", "", "Name", newClusterRoles.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind cluster roles.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind instance Role Bindings already exist, if not create new ones
	roleBinding := &rbacv1.RoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.CodewindRoleBindingNamePrefix + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, roleBinding)
	if err != nil && k8serr.IsNotFound(err) {
		newRoleBinding := r.roleBindingForCodewind(codewind)
		reqLogger.Info("Creating a new Codewind role binding", "Namespace", newRoleBinding.Namespace, "Name", newRoleBinding.Name)
		err = r.client.Create(context.TODO(), newRoleBinding)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind role binding.", "Namespace", newRoleBinding.Namespace, "Name", newRoleBinding.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind role binding.")
		return reconcile.Result{}, err
	}

	// Check if the Tekton Cluster roles already exist, if not create new ones
	clusterRolesTekton := &rbacv1.ClusterRole{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.CodewindTektonClusterRolesName, Namespace: ""}, clusterRolesTekton)
	if err != nil && k8serr.IsNotFound(err) {
		newClusterRoles := r.clusterRolesForCodewindTekton(codewind)
		reqLogger.Info("Creating a new Codewind Tekton cluster roles", "Namespace", "", "Name", newClusterRoles.Name)
		err = r.client.Create(context.TODO(), newClusterRoles)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind Tekton cluster roles.", "Namespace", "", "Name", newClusterRoles.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Tekton cluster roles.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Tekton Cluster Role Bindings already exist, if not create new ones
	roleBindingTekton := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.CodewindTektonClusterRoleBindingName + "-" + codewind.Spec.WorkspaceID, Namespace: ""}, roleBindingTekton)
	if err != nil && k8serr.IsNotFound(err) {
		newTektonRoleBinding := r.roleBindingForCodewindTekton(codewind)
		reqLogger.Info("Creating a new Codewind Tekton ClusterRoleBinding", "Namespace", newTektonRoleBinding.Namespace, "Name", newTektonRoleBinding.Name)
		err = r.client.Create(context.TODO(), newTektonRoleBinding)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind Tekton ClusterRoleBinding.", "Namespace", newTektonRoleBinding.Namespace, "Name", newTektonRoleBinding.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Tekton ClusterRoleBinding.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Service account already exist, if not create new ones
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, serviceAccount)
	if err != nil && k8serr.IsNotFound(err) {
		newServiceAccount := r.serviceAccountForCodewind(codewind)
		reqLogger.Info("Creating a new Codewind service account", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
		err = r.client.Create(context.TODO(), newServiceAccount)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind service account.", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get service account.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PVC already exist, if not create a new one
	codewindPVC := &corev1.PersistentVolumeClaim{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindPFE + "-pvc-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, codewindPVC)
	if err != nil && k8serr.IsNotFound(err) {
		newCodewindPVC := r.pvcForCodewind(codewind, storageClassName, storageSize)
		reqLogger.Info("Creating a new Codewind PFE PVC", "Namespace", newCodewindPVC.Namespace, "Name", newCodewindPVC.Name)
		err = r.client.Create(context.TODO(), newCodewindPVC)
		if err != nil {
			reqLogger.Error(err, "Failed to create new PFE PVC.", "Namespace", newCodewindPVC.Namespace, "Name", newCodewindPVC.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get PFE PVC.")
		return reconcile.Result{}, err
	}

	keycloakPod, err := r.getKeycloakPod(reqLogger, request, codewind.Spec.KeycloakDeployment)
	if err != nil || keycloakPod == nil {
		reqLogger.Error(err, "Unable to find the requested Keycloak pod")
		return reconcile.Result{RequeueAfter: time.Second * 10}, err
	}
	reqLogger.Info("Found the running Keycloak Pod", "Labels:", keycloakPod.GetLabels())

	// Get the keycloak admin credentials
	keycloakAdminUser, keycloakAdminPass, err := r.getKeycloakAdminCredentials(codewind.Spec.KeycloakDeployment, keycloakPod.Namespace)
	if err != nil {
		reqLogger.Error(err, "Unable to retrieve the Keycloak credentials")
		return reconcile.Result{RequeueAfter: time.Second * 10}, err
	}

	keycloakRealm := defaultRealm
	keycloakAuthURL := "https://" + defaults.PrefixCodewindKeycloak + "-" + codewind.Spec.KeycloakDeployment + "." + ingressDomain
	keycloakClientID := "codewind-" + codewind.Spec.WorkspaceID
	gatekeeperPublicURL := "https://" + defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID + "." + ingressDomain
	clientKey := ""

	// Update Keycloak for user if needed
	if codewind.Status.KeycloadStatus == "" {
		codewind.Status.KeycloadStatus = defaults.ConstKeycloakConfigStarted
		clientKey, err = security.AddCodewindToKeycloak(codewind.Spec.WorkspaceID, keycloakAuthURL, keycloakRealm, keycloakAdminUser, keycloakAdminPass, gatekeeperPublicURL, codewind.Spec.Username, keycloakClientID)
		if err != nil {
			reqLogger.Error(err, "Failed to update Keycloak for deployment.", "Namespace", codewind.Namespace, "ClientID", keycloakClientID)
			return reconcile.Result{}, err
		}
		codewind.Status.KeycloadStatus = defaults.ConstKeycloakConfigReady
	}

	// Check if the Codewind PFE Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindPFE + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deployment)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Deployment
		dep := r.deploymentForCodewindPFE(codewind, isOpenshift, keycloakRealm, keycloakAuthURL, codewind.Spec.LogLevel, ingressDomain)
		reqLogger.Info("The workspace ID of this is:", "WorkspaceID", codewind.Spec.WorkspaceID)
		reqLogger.Info("Creating a new PFE Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new PFE deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get PFE Deployment.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindPFE + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, service)
	if err != nil && k8serr.IsNotFound(err) {
		newService := r.serviceForCodewindPFE(codewind)
		reqLogger.Info("Creating a new Service", "Namespace", newService.Namespace, "Name", newService.Name)
		err = r.client.Create(context.TODO(), newService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new service.", "Namespace", newService.Namespace, "Name", newService.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Deployment already exists, if not create a new one
	deploymentPerformance := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindPerformance + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deploymentPerformance)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Performance Deployment
		newDeployment := r.deploymentForCodewindPerformance(codewind, ingressDomain)
		reqLogger.Info("Creating a new Performance deployment.", "Namespace", codewind.Namespace, "Name", defaults.PrefixCodewindPerformance+"-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Performance deployment.", "Namespace", codewind.Namespace, "Name", defaults.PrefixCodewindPerformance+"-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Performance deployment")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Performance Service already exists, if not create a new one
	servicePerformance := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindPerformance + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, servicePerformance)
	if err != nil && k8serr.IsNotFound(err) {
		newService := r.serviceForCodewindPerformance(codewind)
		reqLogger.Info("Creating a new Codewind performance service", "Namespace", newService.Namespace, "Name", defaults.PrefixCodewindPerformance+"-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service.", "Namespace", newService.Namespace, "Name", defaults.PrefixCodewindPerformance+"-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Performance service")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Gatekeeper session secrets already exist, if not create new ones
	secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-codewind-session-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, secret)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		session := strings.ToUpper(strconv.FormatInt(util.CreateTimestamp(), 36))
		newSecret := r.buildGatekeeperSecretSession(codewind, session)
		reqLogger.Info("Creating a new Secret", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
		err = r.client.Create(context.TODO(), newSecret)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Gatekeeper session secret.", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Gatekeeper session secret.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Gatekeeper TLS secrets already exist, if not create new ones
	secret = &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-codewind-tls-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, secret)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		newSecret := r.buildGatekeeperSecretTLS(codewind, ingressDomain)
		reqLogger.Info("Creating a new Secret", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
		err = r.client.Create(context.TODO(), newSecret)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Gatekeeper TLS secret.", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get TLS secret.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Gatekeeper Auth secrets already exist, if not create new ones
	secret = &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-codewind-client-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, secret)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		newSecret := r.buildGatekeeperSecretAuth(codewind, clientKey)
		reqLogger.Info("Creating a new Gatekeeper Auth Secret", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
		err = r.client.Create(context.TODO(), newSecret)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Gatekeeper TLS secret.", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Gatekeeper auth secret.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Deployment already exists, if not create a new one
	deploymentGatekeeper := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deploymentGatekeeper)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Gatekeeper Deployment
		newDeployment := r.deploymentForCodewindGatekeeper(codewind, isOpenshift, keycloakRealm, keycloakClientID, keycloakAuthURL, ingressDomain)
		reqLogger.Info("Creating a new Gatekeeper deployment.", "Namespace", codewind.Namespace, "Name", defaults.PrefixCodewindGatekeeper+"-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Gatekeeper deployment.", "Namespace", codewind.Namespace, "Name", defaults.PrefixCodewindGatekeeper+"-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Gatekeeper deployment")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Service already exists, if not create a new one
	serviceGatekeeper := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, serviceGatekeeper)
	if err != nil && k8serr.IsNotFound(err) {
		newService := r.serviceForCodewindGatekeeper(codewind)
		reqLogger.Info("Creating a new Codewind gatekeeper Service", "Namespace", newService.Namespace, "Name", newService.Name)
		err = r.client.Create(context.TODO(), newService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind gatekeeper service.", "Namespace", newService.Namespace, "Name", newService.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service.")
		return reconcile.Result{}, err
	}

	if isOpenshift {
		// Check if the Codewind Gatekeeper Route already exists, if not create a new one
		routeGatekeeper := &v1.Route{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, routeGatekeeper)
		if err != nil && k8serr.IsNotFound(err) {
			newRoute := r.routeForCodewindGatekeeper(codewind, ingressDomain)
			reqLogger.Info("Creating a new Codewind gatekeeper ingress", "Namespace", newRoute.Namespace, "Name", newRoute.Name)
			err = r.client.Create(context.TODO(), newRoute)
			if err != nil {
				reqLogger.Error(err, "Failed to create new Codewind gatekeeper route.", "Namespace", newRoute.Namespace, "Name", newRoute.Name)
				return reconcile.Result{}, err
			}
			// Success, update the accessURL
			codewind.Status.AccessURL = gatekeeperPublicURL
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Codewind gatekeeper route")
			return reconcile.Result{}, err
		}
		err = r.client.Status().Update(context.TODO(), codewind)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		// Check if the Codewind Gatekeeper Ingress already exists, if not create a new one
		ingressGatekeeper := &extv1beta1.Ingress{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, ingressGatekeeper)
		if err != nil && k8serr.IsNotFound(err) {
			newIngress := r.ingressForCodewindGatekeeper(codewind, ingressDomain)
			reqLogger.Info("Creating a new Codewind gatekeeper ingress", "Namespace", newIngress.Namespace, "Name", newIngress.Name)
			err = r.client.Create(context.TODO(), newIngress)
			if err != nil {
				reqLogger.Error(err, "Failed to create new Codewind gatekeeper ingress.", "Namespace", newIngress.Namespace, "Name", newIngress.Name)
				return reconcile.Result{}, err
			}
			// Success, update the accessURL
			codewind.Status.AccessURL = gatekeeperPublicURL
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Codewind gatekeeper ingress")
			return reconcile.Result{}, err
		}

		err = r.client.Status().Update(context.TODO(), codewind)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCodewind) getKeycloakPod(reqLogger logr.Logger, request reconcile.Request, authDeploymentName string) (*corev1.Pod, error) {
	keycloaks := &corev1.PodList{}
	opts := []client.ListOption{
		client.MatchingLabels{"app": defaults.PrefixCodewindKeycloak, "authName": authDeploymentName},
	}
	err := r.client.List(context.TODO(), keycloaks, opts...)
	if len(keycloaks.Items) == 0 {
		err = fmt.Errorf("Unable to find Keycloak authName:'%s'", authDeploymentName)
		return nil, err
	}
	keycloakPod := keycloaks.Items[0]
	return &keycloakPod, nil
}

// getKeycloakAdminCredentials from the keycloak secret
func (r *ReconcileCodewind) getKeycloakAdminCredentials(keycloakName string, keycloakNamespace string) (username string, password string, err error) {
	secretUser := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-keycloak-user-" + keycloakName, Namespace: keycloakNamespace}, secretUser)
	if err != nil {
		return "", "", err
	}
	return string(secretUser.Data["keycloak-admin-user"]), string(secretUser.Data["keycloak-admin-password"]), nil
}

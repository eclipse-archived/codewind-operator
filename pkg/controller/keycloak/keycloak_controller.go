package keycloak

import (
	"context"
	"io/ioutil"
	"os"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/security"
	"github.com/eclipse/codewind-operator/pkg/util"
	v1 "github.com/openshift/api/route/v1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloak")

// Add : creates a new Keycloak Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler : returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	reconciler := &ReconcileKeycloak{client: mgr.GetClient(), scheme: mgr.GetScheme()}
	operatorNamespace := util.GetOperatorNamespace()
	createOperatorConfigMap(reconciler, operatorNamespace)
	return &ReconcileKeycloak{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func createOperatorConfigMap(reconciler *ReconcileKeycloak, operatorNamespace string) {
	// Create an intial config map if one is not already installed
	log.Info("Checking operator config map")
	configMap := &corev1.ConfigMap{}
	configMap.Namespace = operatorNamespace
	configMap.Name = defaults.OperatorConfigMapName
	fData, err := ioutil.ReadFile(defaults.ConfigMapLocation)
	if err != nil {
		log.Error(err, "Failed to read config map defaults", "Location", defaults.ConfigMapLocation)
		os.Exit(1)
	}
	err = yaml.Unmarshal(fData, configMap)
	if err != nil {
		log.Error(err, "Failed to parse defaults config map from file", "Location", defaults.ConfigMapLocation)
		os.Exit(1)
	}
	configMap.Namespace = operatorNamespace
	err = reconciler.client.Create(context.TODO(), configMap)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error(err, "Failed to create a new operator config map", "Name", configMap.Name)
		os.Exit(1)
	} else {
		log.Info("New config map created", "name", configMap.Name)
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	isOpenshift, _, err := util.DetectOpenShift()
	if err != nil {
		log.Error(err, "Error detecting platfom", "")
	}

	// Create a new controller
	c, err := controller.New("keycloak-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Keycloak
	err = c.Watch(&source.Kind{Type: &codewindv1alpha1.Keycloak{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources and requeue the owner Codewind
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	})
	if err != nil {
		return err
	}

	// Secret
	if err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	}); err != nil {
		return err
	}

	// service
	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	}); err != nil {
		return err
	}

	// Watch for Deployment changes and requeue the controlled owner Keycloak
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	})
	if err != nil {
		return err
	}

	// persistent volume claim
	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	})
	if err != nil {
		return err
	}

	if isOpenshift {
		// Watch for route changes and requeue the controlled owner Keycloak
		err = c.Watch(&source.Kind{Type: &v1.Route{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &codewindv1alpha1.Keycloak{},
		})
		if err != nil {
			return err
		}
	} else {
		// Ingress
		err = c.Watch(&source.Kind{Type: &extv1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &codewindv1alpha1.Keycloak{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// blank assignment to verify that ReconcileKeycloak implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKeycloak{}

// ReconcileKeycloak reconciles a Keycloak object
type ReconcileKeycloak struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile : Reads that state of the cluster for a Keycloak object and makes changes between the current state and required Keycloak.Spec
// Creates a Keycloak Deployment for each Keycloak CR
// Note:
// The Controller will requeue the Request to be processed again if there was an error or Result.Requeue is true,
// otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKeycloak) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Keycloak")

	isOpenshift, _, err := util.DetectOpenShift()
	if err != nil {
		reqLogger.Error(err, "An error occurred when detecting current infrastructure", "")
	}

	// Use ROKSStorageClassGID when it is available
	storageClassName := ""
	storageClassDef := &storagev1.StorageClass{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: defaults.ROKSStorageClassGID, Namespace: ""}, storageClassDef)
	if err == nil {
		reqLogger.Info("Using storageclass", "name", defaults.ROKSStorageClassGID)
		storageClassName = defaults.ROKSStorageClassGID
	}

	// Fetch the Keycloak instance
	keycloak := &codewindv1alpha1.Keycloak{}
	err = r.client.Get(context.TODO(), request.NamespacedName, keycloak)
	if err != nil {
		if k8serr.IsNotFound(err) {
			reqLogger.Info("Keycloak resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Keycloak.", "")
		return reconcile.Result{}, err
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
	reqLogger.Info("Ingress Domain", "value", ingressDomain)

	storageKeycloakSize := operatorConfigMap.Data["storageKeycloakSize"]
	reqLogger.Info("Default Storage Size Keycloak", "value", storageKeycloakSize)

	// Check if the Keycloak Service account already exist, if not create a new one
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, serviceAccount)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new serviceAccount object
		newServiceAccount := r.serviceAccountForKeycloak(keycloak)
		reqLogger.Info("Creating a new service account", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
		err = r.client.Create(context.TODO(), newServiceAccount)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Secret.", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get service account.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Secrets already exist, if not create new ones
	secretUser := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-keycloak-user-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, secretUser)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		secretUser = r.secretsForKeycloak(keycloak)
		reqLogger.Info("Creating a new Secret", "Namespace", secretUser.Namespace, "Name", secretUser.Name)
		err = r.client.Create(context.TODO(), secretUser)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Secret.", "Namespace", secretUser.Namespace, "Name", secretUser.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Secret.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak PVC already exist, if not create a new one
	keycloakPVC := &corev1.PersistentVolumeClaim{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-pvc-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, keycloakPVC)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new PVC object
		newKeycloakPVC := r.pvcForKeycloak(keycloak, storageClassName, storageKeycloakSize)
		reqLogger.Info("Creating a new PVC", "Namespace", newKeycloakPVC.Namespace, "Name", newKeycloakPVC.Name)
		err = r.client.Create(context.TODO(), newKeycloakPVC)
		if err != nil {
			reqLogger.Error(err, "Failed to create new PVC.", "Namespace", newKeycloakPVC.Namespace, "Name", newKeycloakPVC.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get PVC.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, deployment)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Deployment
		dep := r.deploymentForKeycloak(keycloak)
		reqLogger.Info("The workspace ID of this is:", "WorkspaceID", keycloak.Spec.WorkspaceID)
		reqLogger.Info("Creating a new Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		// TODO: GET the deployment object again instead of requeuing it see: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, service)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Service object
		ser := r.serviceForKeycloak(keycloak)
		reqLogger.Info("Creating a new Service", "Namespace", ser.Namespace, "Name", ser.Name)
		err = r.client.Create(context.TODO(), ser)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service.", "Namespace", ser.Namespace, "Name", ser.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service.")
		return reconcile.Result{}, err
	}

	if isOpenshift {
		// Check if the Keycloak Ingress already exists, if not create a new one
		route := &v1.Route{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, route)
		if err != nil && k8serr.IsNotFound(err) {
			// Define a new Route object
			openshiftRoute := r.routeForKeycloak(keycloak, ingressDomain)
			reqLogger.Info("Creating a new route", "Namespace", openshiftRoute.Namespace, "Name", openshiftRoute.Name)
			err = r.client.Create(context.TODO(), openshiftRoute)
			if err != nil {
				reqLogger.Error(err, "Failed to create new route.", "Namespace", openshiftRoute.Namespace, "Name", openshiftRoute.Name)
				return reconcile.Result{}, err
			}
			// Update the accessURL
			keycloak.Status.AccessURL = "https://codewind-keycloak-" + keycloak.Spec.WorkspaceID + "." + ingressDomain
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Keycloak route")
			return reconcile.Result{}, err
		}
	} else {
		// Check if the Keycloak Ingress already exists, if not create a new one
		ingress := &extv1beta1.Ingress{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, ingress)
		if err != nil && k8serr.IsNotFound(err) {
			// Define a new Ingress object
			ing := r.ingressForKeycloak(keycloak, ingressDomain)
			reqLogger.Info("Creating a new Ingress", "Namespace", ing.Namespace, "Name", ing.Name)
			err = r.client.Create(context.TODO(), ing)
			if err != nil {
				reqLogger.Error(err, "Failed to create new Ingress.", "Namespace", ing.Namespace, "Name", ing.Name)
				return reconcile.Result{}, err
			}
			// Update the accessURL
			keycloak.Status.AccessURL = "https://codewind-keycloak-" + keycloak.Spec.WorkspaceID + "." + ingressDomain
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Keycloak Ingress")
			return reconcile.Result{}, err
		}
	}

	// Update keycloak with realm configuration
	defaultRealm := operatorConfigMap.Data["defaultRealm"]
	if keycloak.Status.DefaultRealm != defaultRealm {
		keycloak.Status.DefaultRealm = defaultRealm
		err = security.AddCodewindRealmToKeycloak("https://codewind-keycloak-"+keycloak.Spec.WorkspaceID+"."+ingressDomain, defaultRealm, string(secretUser.Data["keycloak-admin-user"]), string(secretUser.Data["keycloak-admin-password"]))
		if err != nil {
			reqLogger.Error(err, "Failed configuring keycloak with codewind default realm", "Namespace", keycloak.Namespace, "keycloakRealm", operatorConfigMap.Data["defaultRealm"])
			return reconcile.Result{}, err
		}
	}

	err = r.client.Status().Update(context.TODO(), keycloak)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

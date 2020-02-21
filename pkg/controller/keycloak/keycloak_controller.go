package keycloak

import (
	"context"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	"github.com/eclipse/codewind-operator/pkg/controller/defaults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	return &ReconcileKeycloak{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

	// Watch for changes and requeue the controlled owner Keycloak
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	})
	if err != nil {
		return err
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

	// Fetch the Keycloak instance
	keycloak := &codewindv1alpha1.Keycloak{}
	err := r.client.Get(context.TODO(), request.NamespacedName, keycloak)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Keycloak resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Keycloak.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Service account already exist, if not create new ones
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, serviceAccount)
	if err != nil && errors.IsNotFound(err) {
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
	secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-keycloak-user-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secrets object
		newSecret := r.secretsForKeycloak(keycloak)
		reqLogger.Info("Creating a new Secret", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
		err = r.client.Create(context.TODO(), newSecret)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Secret.", "Namespace", newSecret.Namespace, "Name", newSecret.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Secret.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak PVC already exist, if not create a new one
	keycloakPVC := &corev1.PersistentVolumeClaim{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-pvc-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, keycloakPVC)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secrets object
		newKeycloakPVC := r.pvcForKeycloak(keycloak)
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
	if err != nil && errors.IsNotFound(err) {
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
	if err != nil && errors.IsNotFound(err) {
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

	// Check if the Keycloak Ingress already exists, if not create a new one
	ingress := &extv1beta1.Ingress{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-keycloak-" + keycloak.Spec.WorkspaceID, Namespace: keycloak.Namespace}, ingress)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Ingress object
		ing := r.ingressForKeycloak(keycloak)
		reqLogger.Info("Creating a new Ingress", "Namespace", ing.Namespace, "Name", ing.Name)

		// Update the accessURL
		keycloak.Status.AccessURL = "https://codewind-keycloak-" + keycloak.Spec.WorkspaceID + "." + defaults.GetCurrentIngressDomain()
		err = r.client.Create(context.TODO(), ing)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Ingress.", "Namespace", ing.Namespace, "Name", ing.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Ingress")
		return reconcile.Result{}, err
	}

	err = r.client.Status().Update(context.TODO(), keycloak)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

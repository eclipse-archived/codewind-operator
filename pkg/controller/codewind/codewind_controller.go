package codewind

import (
	"context"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	return &ReconcileCodewind{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

	// Watch for changes to secondary resource Pods and requeue the owner Codewind
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	})
	if err != nil {
		return err
	}

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
	reqLogger.Info("Reconciling Codewind")

	// Fetch the Codewind instance
	codewind := &codewindv1alpha1.Codewind{}
	err := r.client.Get(context.TODO(), request.NamespacedName, codewind)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Codewind resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Codewind.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Service account already exist, if not create new ones
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, serviceAccount)
	if err != nil && errors.IsNotFound(err) {
		// Define a new serviceAccount object
		newServiceAccount := r.serviceAccountForCodewind(codewind)
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

	// Check if the Codewind PVC already exist, if not create a new one
	codewindPVC := &corev1.PersistentVolumeClaim{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-pfe-pvc-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, codewindPVC)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secrets object
		newCodewindPVC := r.pvcForCodewind(codewind)
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

	// Check if the Codewind PFE Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-pfe-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Deployment

		// TODO - pull these from the keycloak service
		keycloakRealm := "codewind"
		authHost := "https://keycloak......."
		logLevel := "info"
		isOnOpenshift := false

		dep := r.deploymentForCodewindPFE(codewind, isOnOpenshift, keycloakRealm, authHost, logLevel)
		reqLogger.Info("The workspace ID of this is:", "WorkspaceID", codewind.Spec.WorkspaceID)
		reqLogger.Info("Creating a new PFE Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		// TODO: GET the deployment object again instead of requeuing it see: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get PFE Deployment.")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-pfe-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Service object
		ser := r.serviceForCodewindPFE(codewind)
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

	err = r.client.Status().Update(context.TODO(), codewind)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

package codewind

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	util "github.com/eclipse/codewind-operator/pkg/util"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
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

	isOpenshift, _, err := util.DetectOpenShift()
	if err != nil {
		logrus.Errorf("Error detecting platfom: %s", err)
	}
	logrus.Infof("Running on Openshift: %t", isOpenshift)

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

	// Secret
	if err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	}); err != nil {
		return err
	}

	// service
	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	}); err != nil {
		return err
	}

	// service account
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	})
	if err != nil {
		return err
	}

	// deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	})
	if err != nil {
		return err
	}

	// persistent volume claim
	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Codewind{},
	})
	if err != nil {
		return err
	}

	// Ingress
	err = c.Watch(&source.Kind{Type: &extensionsv1.Ingress{}}, &handler.EnqueueRequestForOwner{
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

	isOpenshift, _, err := util.DetectOpenShift()
	if err != nil {
		logrus.Errorf("An error occurred when detecting current infrastructure: %s", err)
	}

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Codewind")

	// Fetch the Codewind instance
	codewind := &codewindv1alpha1.Codewind{}
	err = r.client.Get(context.TODO(), request.NamespacedName, codewind)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Codewind resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Codewind.")
		return reconcile.Result{}, err
	}

	// Get the keycloak pod
	keycloakPod, err := r.fetchKeycloakPod(reqLogger, request, codewind.Spec.KeycloakDeployment)
	if err != nil || keycloakPod == nil {
		reqLogger.Error(err, "Unable to find the requested Keycloak pod")
		return reconcile.Result{RequeueAfter: time.Second * 10}, err
	}
	reqLogger.Info("Found the running Keycloak Pod", "Labels:", keycloakPod.GetLabels())

	// Check if the Codewind Service account already exist, if not create new ones
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, serviceAccount)
	if err != nil && errors.IsNotFound(err) {
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
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-pfe-pvc-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, codewindPVC)
	if err != nil && errors.IsNotFound(err) {
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

	// TODO - pull these from the keycloak service
	keycloakRealm := defaults.CodewindAuthRealm
	keycloakAuthURL := "https://" + "codewind-keycloak-k3a237fj.10.100.111.145.nip.io/TODO"
	keycloakClientID := "codewind-" + codewind.Spec.WorkspaceID
	logLevel := "info"

	// Check if the Codewind PFE Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-pfe-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Deployment
		dep := r.deploymentForCodewindPFE(codewind, isOpenshift, keycloakRealm, keycloakAuthURL, logLevel)
		reqLogger.Info("The workspace ID of this is:", "WorkspaceID", codewind.Spec.WorkspaceID)
		reqLogger.Info("Creating a new PFE Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new PFE deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
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
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-performance-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deploymentPerformance)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Performance Deployment
		newDeployment := r.deploymentForCodewindPerformance(codewind)
		reqLogger.Info("Creating a new Performance deployment.", "Namespace", codewind.Namespace, "Name", "codewind-performance-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Performance deployment.", "Namespace", codewind.Namespace, "Name", "codewind-performance-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Performance deployment")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Performance Service already exists, if not create a new one
	servicePerformance := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-performance-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, servicePerformance)
	if err != nil && errors.IsNotFound(err) {
		newService := r.serviceForCodewindPerformance(codewind)
		reqLogger.Info("Creating a new Codewind performance service", "Namespace", newService.Namespace, "Name", "codewind-performance-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service.", "Namespace", newService.Namespace, "Name", "codewind-performance-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Performance service")
		return reconcile.Result{}, err
	}

	// Check if the Codewind Gatekeeper session secrets already exist, if not create new ones
	secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "secret-codewind-session-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
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
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secrets object
		newSecret := r.buildGatekeeperSecretTLS(codewind)
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
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secrets object
		clientKey := "TODO:  GET THIS SECRET FROM KEYCLOAK"
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
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-gatekeeper-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, deploymentGatekeeper)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Gatekeeper Deployment
		newDeployment := r.deploymentForCodewindGatekeeper(codewind, isOpenshift, keycloakRealm, keycloakClientID, keycloakAuthURL)
		reqLogger.Info("Creating a new Gatekeeper deployment.", "Namespace", codewind.Namespace, "Name", "codewind-gatekeeper-"+codewind.Spec.WorkspaceID)
		err = r.client.Create(context.TODO(), newDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Gatekeeper deployment.", "Namespace", codewind.Namespace, "Name", "codewind-gatekeeper-"+codewind.Spec.WorkspaceID)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind Gatekeeper deployment")
		return reconcile.Result{}, err
	}

	// Check if the Codewind PFE Service already exists, if not create a new one
	serviceGatekeeper := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-gatekeeper-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, serviceGatekeeper)
	if err != nil && errors.IsNotFound(err) {
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

	// Check if the Codewind Gatekeeper Ingress already exists, if not create a new one
	ingressGatekeeper := &extensionsv1.Ingress{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "codewind-gatekeeper-" + codewind.Spec.WorkspaceID, Namespace: codewind.Namespace}, ingressGatekeeper)
	if err != nil && errors.IsNotFound(err) {
		newIngress := r.ingressForCodewindGatekeeper(codewind)
		reqLogger.Info("Creating a new Codewind gatekeeper ingress", "Namespace", newIngress.Namespace, "Name", newIngress.Name)
		err = r.client.Create(context.TODO(), newIngress)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Codewind gatekeeper ingress.", "Namespace", newIngress.Namespace, "Name", newIngress.Name)
			return reconcile.Result{}, err
		}
		// Success, update the accessURL
		codewind.Status.AccessURL = "https://codewind-gatekeeper-" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Codewind gatekeeper ingress")
		return reconcile.Result{}, err
	}

	err = r.client.Status().Update(context.TODO(), codewind)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCodewind) fetchKeycloakPod(reqLogger logr.Logger, request reconcile.Request, keycloakDeploymentRef string) (*corev1.Pod, error) {
	keycloaks := &corev1.PodList{}
	opts := []client.ListOption{
		client.MatchingLabels{"app": "codewind-keycloak", "deploymentRef": keycloakDeploymentRef},
	}
	err := r.client.List(context.TODO(), keycloaks, opts...)
	if len(keycloaks.Items) == 0 {
		err = fmt.Errorf("Unable to find Keycloak deployment '%s'", keycloakDeploymentRef)
		return nil, err
	}
	keycloakPod := keycloaks.Items[0]
	return &keycloakPod, nil
}

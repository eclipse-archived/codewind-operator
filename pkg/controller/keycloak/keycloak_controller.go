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

package keycloak

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/security"
	"github.com/eclipse/codewind-operator/pkg/util"
	routev1 "github.com/openshift/api/route/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_keycloak")

// DeploymentOptionsKeycloak : Configuration settings of a Keycloak deployment
type DeploymentOptionsKeycloak struct {
	KeycloakServiceAccountName string
	KeycloakPVCName            string
	KeycloakSecretsName        string
	KeycloakTLSSecretsName     string
	KeycloakTLSCertTitle       string
	KeycloakDeploymentName     string
	KeycloakServiceName        string
	KeycloakIngressName        string
	KeycloakIngressHost        string
	KeycloakAccessURL          string
}

// OperatorConfigMapCodewind : Configuration fields saved in the config map
type OperatorConfigMapCodewind struct {
	IngressDomain       string
	StorageSize         string
	KeycloakStorageSize string
	DefaultRealm        string
}

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
	fileData, err := ioutil.ReadFile(defaults.ConfigMapLocation)
	if err != nil {
		log.Error(err, "Failed to read config map defaults", "Location", defaults.ConfigMapLocation)
		os.Exit(1)
	}
	err = yaml.Unmarshal(fileData, configMap)
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

	// Watch for changes to secondary resources and requeue the owner
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to the Keycloak deployment to catch pod changes that require keycloak database updates
	src := &source.Kind{Type: &appsv1.Deployment{}}
	h := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codewindv1alpha1.Keycloak{},
	}
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
	err = c.Watch(src, h, pred)
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
			// Keycloak resource not found. Ignoring since object must be deleted
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

	configMapCodewind := OperatorConfigMapCodewind{
		IngressDomain:       operatorConfigMap.Data["ingressDomain"],
		StorageSize:         operatorConfigMap.Data["storageCodewindSize"],
		KeycloakStorageSize: operatorConfigMap.Data["storageKeycloakSize"],
		DefaultRealm:        operatorConfigMap.Data["defaultRealm"],
	}

	deploymentOptions := DeploymentOptionsKeycloak{
		KeycloakServiceAccountName: defaults.PrefixCodewindKeycloak + "-" + keycloak.Name,
		KeycloakPVCName:            defaults.PrefixCodewindKeycloak + "-pvc-" + keycloak.Name,
		KeycloakSecretsName:        "secret-keycloak-user-" + keycloak.Name,
		KeycloakTLSSecretsName:     "secret-keycloak-tls-" + keycloak.Name,
		KeycloakTLSCertTitle:       "Keycloak" + "-" + keycloak.Name,
		KeycloakDeploymentName:     defaults.PrefixCodewindKeycloak + "-" + keycloak.Name,
		KeycloakServiceName:        defaults.PrefixCodewindKeycloak + "-" + keycloak.Name,
		KeycloakIngressName:        defaults.PrefixCodewindKeycloak + "-" + keycloak.Name,
		KeycloakIngressHost:        defaults.PrefixCodewindKeycloak + "-" + keycloak.Name + "." + configMapCodewind.IngressDomain,
		KeycloakAccessURL:          "https://" + defaults.PrefixCodewindKeycloak + "-" + keycloak.Name + "." + configMapCodewind.IngressDomain,
	}

	// Check if the Keycloak Service account already exist, if not create a new one
	serviceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakServiceAccountName, Namespace: keycloak.Namespace}, serviceAccount)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new serviceAccount object
		newServiceAccount := r.serviceAccountForKeycloak(keycloak, deploymentOptions)
		reqLogger.Info("Creating a new service account", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
		err = r.client.Create(context.TODO(), newServiceAccount)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			reqLogger.Error(err, "Failed to create new Secret.", "Namespace", newServiceAccount.Namespace, "Name", newServiceAccount.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get service account.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Secrets already exist, if not create new ones
	secretUser := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakSecretsName, Namespace: keycloak.Namespace}, secretUser)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		secretUser = r.secretsForKeycloak(keycloak, deploymentOptions)
		reqLogger.Info("Creating a new Keycloak Secret", "Namespace", secretUser.Namespace, "Name", secretUser.Name)
		err = r.client.Create(context.TODO(), secretUser)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			reqLogger.Error(err, "Failed to create new Keycloak Secret.", "Namespace", secretUser.Namespace, "Name", secretUser.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Keycloak Secret.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak TLS Secrets already exist, if not create new ones
	secretTLS := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakTLSSecretsName, Namespace: keycloak.Namespace}, secretTLS)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Secrets object
		secretTLS = r.secretsTLSForKeycloak(keycloak, deploymentOptions)
		reqLogger.Info("Creating a new Keycloak TLS Secret", "Namespace", secretTLS.Namespace, "Name", secretTLS.Name)
		err = r.client.Create(context.TODO(), secretTLS)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			reqLogger.Error(err, "Failed to create new Keycloak TLS Secret.", "Namespace", secretTLS.Namespace, "Name", secretTLS.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Keycloak TLS Secret.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak PVC already exist, if not create a new one
	keycloakPVC := &corev1.PersistentVolumeClaim{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakPVCName, Namespace: keycloak.Namespace}, keycloakPVC)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new PVC object

		storageSize := configMapCodewind.KeycloakStorageSize
		if keycloak.Spec.StorageSize != "" {
			storageSize = keycloak.Spec.StorageSize
		}

		newKeycloakPVC := r.pvcForKeycloak(keycloak, deploymentOptions, storageClassName, storageSize)
		reqLogger.Info("Creating a new PVC", "Namespace", newKeycloakPVC.Namespace, "Name", newKeycloakPVC.Name)
		err = r.client.Create(context.TODO(), newKeycloakPVC)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			reqLogger.Error(err, "Failed to create new PVC.", "Namespace", newKeycloakPVC.Namespace, "Name", newKeycloakPVC.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get PVC.")
		return reconcile.Result{}, err
	}

	// Check if the Keycloak Deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakDeploymentName, Namespace: keycloak.Namespace}, deployment)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Deployment
		dep := r.deploymentForKeycloak(keycloak, deploymentOptions)
		reqLogger.Info("Creating a new Deployment.", "Namespace", dep.Namespace, "Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil && !k8serr.IsAlreadyExists(err) {
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
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakServiceName, Namespace: keycloak.Namespace}, service)
	if err != nil && k8serr.IsNotFound(err) {
		// Define a new Service object
		ser := r.serviceForKeycloak(keycloak)
		reqLogger.Info("Creating a new Service", "Namespace", ser.Namespace, "Name", ser.Name)
		err = r.client.Create(context.TODO(), ser)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			reqLogger.Error(err, "Failed to create new Service.", "Namespace", ser.Namespace, "Name", ser.Name)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service.")
		return reconcile.Result{}, err
	}

	if isOpenshift {
		// Check if the Keycloak Route already exists, if not create a new one
		route := &routev1.Route{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakIngressName, Namespace: keycloak.Namespace}, route)
		if err != nil && k8serr.IsNotFound(err) {
			// Define a new Route object
			openshiftRoute := r.routeForKeycloak(keycloak, deploymentOptions, configMapCodewind.IngressDomain)
			reqLogger.Info("Creating a new route", "Namespace", openshiftRoute.Namespace, "Name", openshiftRoute.Name)
			err = r.client.Create(context.TODO(), openshiftRoute)
			if err != nil && !k8serr.IsAlreadyExists(err) {
				reqLogger.Error(err, "Failed to create new route.", "Namespace", openshiftRoute.Namespace, "Name", openshiftRoute.Name)
				return reconcile.Result{}, err
			}
			// Update the accessURL
			keycloak.Status.AccessURL = deploymentOptions.KeycloakAccessURL
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Keycloak route")
			return reconcile.Result{}, err
		}
	} else {
		// Check if the Keycloak Ingress already exists, if not create a new one
		ingress := &extv1beta1.Ingress{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakIngressName, Namespace: keycloak.Namespace}, ingress)
		if err != nil && k8serr.IsNotFound(err) {
			// Define a new Ingress object
			ing := r.ingressForKeycloak(keycloak, deploymentOptions, configMapCodewind.IngressDomain)
			reqLogger.Info("Creating a new Ingress", "Namespace", ing.Namespace, "Name", ing.Name)
			err = r.client.Create(context.TODO(), ing)
			if err != nil && !k8serr.IsAlreadyExists(err) {
				reqLogger.Error(err, "Failed to create new Ingress.", "Namespace", ing.Namespace, "Name", ing.Name)
				return reconcile.Result{}, err
			}
			// Update the accessURL
			keycloak.Status.AccessURL = deploymentOptions.KeycloakAccessURL
		} else if err != nil {
			reqLogger.Error(err, "Failed to get Keycloak Ingress")
			return reconcile.Result{}, err
		}
	}

	// Update Keycloak default realm
	reqLogger.Info("Checking Keycloak Pod", "instance", keycloak.Name)
	keycloakPod, err := fetchKeycloakPod(r.client, keycloak.Name)
	if err == nil && keycloakPod != nil {
		reqLogger.Info("Keycloak Pod status", "phase", keycloakPod.Status.Phase)
		if keycloakPod.Status.Phase == "Running" {
			reqLogger.Info("Keycloak Pod", "instance", keycloak.Name, "Phase", keycloakPod.Status.Phase)
			defaultRealm := configMapCodewind.DefaultRealm
			if keycloak.Status.DefaultRealm != defaultRealm {
				keycloak.Status.DefaultRealm = defaultRealm
				secretUser := &corev1.Secret{}
				err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.KeycloakSecretsName, Namespace: keycloak.Namespace}, secretUser)
				if err != nil {
					reqLogger.Error(err, "Unable to find the Keycloak secret when adding realm", "Namespace", keycloak.Namespace, "name", deploymentOptions.KeycloakSecretsName)
					return reconcile.Result{}, err
				}
				err = security.AddCodewindRealmToKeycloak(deploymentOptions.KeycloakAccessURL, defaultRealm, string(secretUser.Data["keycloak-admin-user"]), string(secretUser.Data["keycloak-admin-password"]))
				if err != nil {
					reqLogger.Error(err, "Failed configuring keycloak with codewind default realm", "Namespace", keycloak.Namespace, "realm", defaultRealm)
					return reconcile.Result{}, err
				}
			}
		}
	}

	err = r.client.Status().Update(context.TODO(), keycloak)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func fetchKeycloakPod(currentClient client.Client, authDeploymentName string) (*corev1.Pod, error) {
	keycloaks := &corev1.PodList{}
	opts := []client.ListOption{
		client.MatchingLabels{"app": defaults.PrefixCodewindKeycloak, "authName": authDeploymentName},
	}
	err := currentClient.List(context.TODO(), keycloaks, opts...)
	if len(keycloaks.Items) == 0 {
		err = fmt.Errorf("Unable to find Keycloak authName:'%s'", authDeploymentName)
		return nil, err
	}
	keycloakPod := keycloaks.Items[0]
	return &keycloakPod, nil
}

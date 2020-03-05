package codewind

import (
	"strconv"

	"github.com/eclipse/codewind-installer/pkg/appconstants"
	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// serviceAccountForCodewind function takes in a Codewind object and returns a serviceAccount for that object.
func (r *ReconcileCodewind) serviceAccountForCodewind(codewind *codewindv1alpha1.Codewind) *corev1.ServiceAccount {
	labels := map[string]string{"app": "codewind-" + codewind.Spec.WorkspaceID, "codewind_cr": codewind.Name, "codewindWorkspace": codewind.Spec.WorkspaceID}
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    labels,
		},
		Secrets: nil,
	}

	// Set Codewind instance as the owner of the service account.
	controllerutil.SetControllerReference(codewind, serviceAccount, r.scheme)
	return serviceAccount
}

// pvcForCodewind function takes in a Codewind object and returns a PVC for that object.
func (r *ReconcileCodewind) pvcForCodewind(codewind *codewindv1alpha1.Codewind) *corev1.PersistentVolumeClaim {
	labels := labelsForCodewindPFE(codewind)

	storageSize := defaults.PFEStorageSize
	if codewind.Spec.StorageSize != "" {
		storageSize = codewind.Spec.StorageSize
	}

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-pfe-pvc-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteMany",
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}

	// Set Codewind instance as the owner of the persistent volume claim.
	controllerutil.SetControllerReference(codewind, pvc, r.scheme)
	return pvc
}

func (r *ReconcileCodewind) deploymentForCodewindPerformance(codewind *codewindv1alpha1.Codewind) *appsv1.Deployment {
	ls := labelsForCodewindPerformance(codewind)
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-performance-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "codewind-" + codewind.Spec.WorkspaceID,
					Containers: []corev1.Container{{
						Name:            "codewind-performance",
						Image:           defaults.CodewindPerformanceImage + ":" + defaults.CodewindPerformanceImageTag,
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{
								Name:  "IN_K8",
								Value: "true",
							},
							{
								Name:  "PORTAL_HTTPS",
								Value: "false",
							},
							{
								Name:  "CODEWIND_INGRESS",
								Value: "codewind-gatekeeper" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain,
							},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: int32(defaults.PerformanceContainerPort)},
						},
					}},
				},
			},
		},
	}
	// Set Codewind instance as the owner of this deployment
	controllerutil.SetControllerReference(codewind, dep, r.scheme)
	return dep
}

// deploymentForCodewindPFE returns a Codewind dployment object
func (r *ReconcileCodewind) deploymentForCodewindPFE(codewind *codewindv1alpha1.Codewind, isOnOpenshift bool, keycloakRealm string, authHost string, logLevel string /*, volumeMounts []corev1.VolumeMount*/) *appsv1.Deployment {
	ls := labelsForCodewindPFE(codewind)
	replicas := int32(1)

	loglevel := "info"
	if codewind.Spec.LogLevel != "" {
		loglevel = codewind.Spec.LogLevel
	}

	volumes := []corev1.Volume{
		{
			Name: "shared-workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "codewind-pfe-pvc-" + codewind.Spec.WorkspaceID,
				},
			},
		},
		{
			Name: "buildah-volume",
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "shared-workspace",
			MountPath: "/codewind-workspace",
			SubPath:   codewind.Spec.WorkspaceID + "/projects",
		},
		{
			Name:      "buildah-volume",
			MountPath: "/var/lib/containers",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-pfe-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "codewind-" + codewind.Spec.WorkspaceID,
					Volumes:            volumes,
					Containers: []corev1.Container{{
						Name:            "codewind-pfe",
						Image:           defaults.CodewindImage + ":" + defaults.CodewindImageTag,
						ImagePullPolicy: corev1.PullAlways,
						VolumeMounts:    volumeMounts,
						Env: []corev1.EnvVar{
							{
								Name:  "TEKTON_PIPELINE",
								Value: "tekton-pipelines",
							},
							{
								Name:  "IN_K8",
								Value: "true",
							},
							{
								Name:  "PORTAL_HTTPS",
								Value: "true",
							},
							{
								Name:  "KUBE_NAMESPACE",
								Value: codewind.Namespace,
							},
							{
								Name:  "TILLER_NAMESPACE",
								Value: codewind.Namespace,
							},
							{
								Name:  "CHE_WORKSPACE_ID",
								Value: codewind.Spec.WorkspaceID,
							},
							{
								Name:  "PVC_NAME",
								Value: "codewind-pfe-pvc-" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "SERVICE_NAME",
								Value: "codewind-" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "SERVICE_ACCOUNT_NAME",
								Value: codewind.Spec.WorkspaceID,
							},
							{
								Name:  "HOST_WORKSPACE_DIRECTORY",
								Value: "/projects",
							},
							{
								Name:  "CONTAINER_WORKSPACE_DIRECTORY",
								Value: "/codewind-workspace",
							},
							{
								Name:  "CODEWIND_VERSION",
								Value: appconstants.VersionNum,
							},
							/*
								{
									Name:  "OWNER_REF_NAME",
									Value: codewind.OwnerReferenceName,
								},
								{
									Name:  "OWNER_REF_UID",
									Value: string(codewind.OwnerReferenceUID),
								},
							*/
							{
								Name:  "CODEWIND_PERFORMANCE_SERVICE",
								Value: "codewind-performance-" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "CHE_INGRESS_HOST",
								Value: "codewind-gatekeeper-" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain,
							},
							{
								Name:  "INGRESS_PREFIX",
								Value: codewind.Namespace + "." + codewind.Spec.IngressDomain, // provides access to project containers
							},
							{
								Name:  "ON_OPENSHIFT",
								Value: strconv.FormatBool(isOnOpenshift),
							},
							{
								Name:  "CODEWIND_AUTH_REALM",
								Value: keycloakRealm,
							},
							{
								Name:  "CODEWIND_AUTH_HOST",
								Value: authHost,
							},
							{
								Name:  "LOG_LEVEL",
								Value: loglevel,
							},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: int32(defaults.PFEContainerPort)},
						},
					}},
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Deployment.
	controllerutil.SetControllerReference(codewind, dep, r.scheme)
	return dep
}

// serviceForCodewindPFE function takes in a Codewind object and returns a PFE Service for that object.
func (r *ReconcileCodewind) serviceForCodewindPFE(codewind *codewindv1alpha1.Codewind) *corev1.Service {
	ls := labelsForCodewindPFE(codewind)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-pfe-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.PFEContainerPort),
					Name: "codewind-http",
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, service, r.scheme)
	return service
}

// serviceForCodewindPerformance function takes in a Codewind  object and returns a Performance Service for that object.
func (r *ReconcileCodewind) serviceForCodewindPerformance(codewind *codewindv1alpha1.Codewind) *corev1.Service {
	ls := labelsForCodewindPerformance(codewind)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-performance-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.PerformanceContainerPort),
					Name: "codewind-performance-http",
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, service, r.scheme)
	return service
}

// serviceForCodewindGatekeeper function takes in a Codewind object and returns a Gatekeeper Service for that object.
func (r *ReconcileCodewind) serviceForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind) *corev1.Service {
	ls := labelsForCodewindGatekeeper(codewind)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-gatekeeper-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.GatekeeperContainerPort),
					Name: "codewind-gatekeeper-http",
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, service, r.scheme)
	return service
}

// deploymentForCodewindGatekeeper returns a Codewind dployment object
func (r *ReconcileCodewind) deploymentForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, isOnOpenshift bool, keycloakRealm string, keycloakClientID string, keycloakAuthURL string) *appsv1.Deployment {
	ls := labelsForCodewindGatekeeper(codewind)
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-gatekeeper-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "codewind-" + codewind.Spec.WorkspaceID,
					Containers: []corev1.Container{{
						Name:            "codewind-gatekeeper",
						Image:           defaults.CodewindGatekeeperImage + ":" + defaults.CodewindGatekeeperImageTag,
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{
								Name:  "AUTH_URL",
								Value: keycloakAuthURL,
							},
							{
								Name:  "CLIENT_ID",
								Value: keycloakClientID,
							},
							{
								Name:  "REALM",
								Value: keycloakRealm,
							},
							{
								Name:  "ENABLE_AUTH",
								Value: "1",
							},
							{
								Name:  "GATEKEEPER_HOST",
								Value: "codewind-gatekeeper-" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain,
							},

							{
								Name:  "WORKSPACE_SERVICE",
								Value: "CODEWIND_PFE_" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "WORKSPACE_ID",
								Value: codewind.Spec.WorkspaceID,
							},
							{
								Name:  "ACCESS_ROLE",
								Value: "codewind-" + codewind.Spec.WorkspaceID,
							},
							{
								Name: "CLIENT_SECRET",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-codewind-client" + "-" + codewind.Spec.WorkspaceID}, Key: "client_secret"}},
							},
							{
								Name: "SESSION_SECRET",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-codewind-session" + "-" + codewind.Spec.WorkspaceID}, Key: "session_secret"}},
							},
							{
								Name:  "PORTAL_HTTPS",
								Value: "true",
							},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: int32(defaults.PFEContainerPort)},
						},
					}},
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Deployment.
	controllerutil.SetControllerReference(codewind, dep, r.scheme)
	return dep
}

// ingressForCodewindGatekeeper function takes in a Codewind object and returns an Ingress for the gatekeeper.
func (r *ReconcileCodewind) ingressForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind) *extensionsv1beta1.Ingress {
	ls := labelsForCodewindGatekeeper(codewind)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":     "/",
		"ingress.bluemix.net/redirect-to-https":          "True",
		"ingress.bluemix.net/ssl-services":               "ssl-service=" + "codewind-gatekeeper" + "-" + codewind.Spec.WorkspaceID,
		"nginx.ingress.kubernetes.io/backend-protocol":   "HTTPS",
		"kubernetes.io/ingress.class":                    "nginx",
		"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
	}

	ingress := &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "codewind-gatekeeper" + "-" + codewind.Spec.WorkspaceID,
			Annotations: annotations,
			Namespace:   codewind.Namespace,
			Labels:      ls,
		},
		Spec: extensionsv1beta1.IngressSpec{
			TLS: []extensionsv1beta1.IngressTLS{
				{
					Hosts:      []string{"codewind-gatekeeper" + "-" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain},
					SecretName: "secret-codewind-tls" + "-" + codewind.Spec.WorkspaceID,
				},
			},
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: "codewind-gatekeeper" + "-" + codewind.Spec.WorkspaceID + "." + codewind.Spec.IngressDomain,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: "codewind-gatekeeper" + "-" + codewind.Spec.WorkspaceID,
										ServicePort: intstr.FromInt(defaults.GatekeeperContainerPort),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, ingress, r.scheme)
	return ingress
}

// buildGatekeeperSessionSecret :  builds a session secret for gatekeeper
func (r *ReconcileCodewind) buildGatekeeperSecretSession(codewind *codewindv1alpha1.Codewind, sessionSecretValue string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(codewind)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-codewind-session-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    metaLabels,
		},
		StringData: map[string]string{
			"session_secret": sessionSecretValue,
		},
	}
	// Set Codewind instance as the owner of this Secret.
	controllerutil.SetControllerReference(codewind, secret, r.scheme)
	return secret
}

// buildGatekeeperSecretTLS :  builds a TLS secret for gatekeeper
func (r *ReconcileCodewind) buildGatekeeperSecretTLS(codewind *codewindv1alpha1.Codewind) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(codewind)

	pemPrivateKey, pemPublicCert, _ := util.GenerateCertificate("codewind-gatekeeper-"+codewind.Spec.WorkspaceID+"."+codewind.Spec.IngressDomain, "Codewind"+"-"+codewind.Spec.WorkspaceID)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-codewind-tls-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    metaLabels,
		},
		StringData: map[string]string{
			"tls.crt": pemPublicCert,
			"tls.key": pemPrivateKey,
		},
	}
	// Set Codewind instance as the owner of this Secret.
	controllerutil.SetControllerReference(codewind, secret, r.scheme)
	return secret
}

// buildGatekeeperSecretAuth :  builds an authentication detail secret for gatekeeper
func (r *ReconcileCodewind) buildGatekeeperSecretAuth(codewind *codewindv1alpha1.Codewind, keycloakClientKey string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(codewind)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-codewind-client-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    metaLabels,
		},
		StringData: map[string]string{
			"client_secret": keycloakClientKey,
		},
	}
	// Set Codewind instance as the owner of this secret.
	controllerutil.SetControllerReference(codewind, secret, r.scheme)
	return secret
}

// labelsForCodewindPFE returns the labels for selecting the resources
// belonging to the given codewind CR name.
func labelsForCodewindPFE(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": "codewind-pfe", "codewindWorkspace": codewind.Spec.WorkspaceID}
}

func labelsForCodewindPerformance(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": "codewind-performance", "codewindWorkspace": codewind.Spec.WorkspaceID}
}

func labelsForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": "codewind-gatekeeper", "codewindWorkspace": codewind.Spec.WorkspaceID}
}

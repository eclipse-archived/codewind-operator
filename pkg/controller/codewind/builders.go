package codewind

import (
	"strconv"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/util"
	v1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
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
func (r *ReconcileCodewind) pvcForCodewind(codewind *codewindv1alpha1.Codewind, storageClassName string, storageSize string) *corev1.PersistentVolumeClaim {
	labels := labelsForCodewindPFE(codewind)
	if codewind.Spec.StorageSize != "" {
		storageSize = codewind.Spec.StorageSize
	}
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindPFE + "-pvc-" + codewind.Spec.WorkspaceID,
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

	// If a storage class was passed in, set it in the PVC
	if storageClassName != "" {
		pvc.Spec.StorageClassName = &storageClassName
	}

	// Set Codewind instance as the owner of the persistent volume claim.
	controllerutil.SetControllerReference(codewind, pvc, r.scheme)
	return pvc
}

func (r *ReconcileCodewind) deploymentForCodewindPerformance(codewind *codewindv1alpha1.Codewind, ingressDomain string) *appsv1.Deployment {
	ls := labelsForCodewindPerformance(codewind)
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindPerformance + "-" + codewind.Spec.WorkspaceID,
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
						Name:            defaults.PrefixCodewindPerformance,
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
								Value: defaults.PrefixCodewindGatekeeper + codewind.Spec.WorkspaceID + "." + ingressDomain,
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
func (r *ReconcileCodewind) deploymentForCodewindPFE(codewind *codewindv1alpha1.Codewind, isOnOpenshift bool, keycloakRealm string, authHost string, logLevel string, ingressDomain string) *appsv1.Deployment {
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
					ClaimName: defaults.PrefixCodewindPFE + "-pvc-" + codewind.Spec.WorkspaceID,
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
			Name:      defaults.PrefixCodewindPFE + "-" + codewind.Spec.WorkspaceID,
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
						Name:            defaults.PrefixCodewindPFE,
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
								Value: defaults.PrefixCodewindPFE + "-pvc-" + codewind.Spec.WorkspaceID,
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
								Value: defaults.CodewindImageTag,
							},
							{
								Name:  "OWNER_REF_NAME",
								Value: "codewind-" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "OWNER_REF_UID",
								Value: string(codewind.GetUID()),
							},
							{
								Name:  "CODEWIND_PERFORMANCE_SERVICE",
								Value: defaults.PrefixCodewindPerformance + "-" + codewind.Spec.WorkspaceID,
							},
							{
								Name:  "CHE_INGRESS_HOST",
								Value: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID + "." + ingressDomain,
							},
							{
								Name:  "INGRESS_PREFIX",
								Value: codewind.Namespace + "." + ingressDomain, // provides access to project containers
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
			Name:      defaults.PrefixCodewindPFE + "-" + codewind.Spec.WorkspaceID,
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
			Name:      defaults.PrefixCodewindPerformance + "-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.PerformanceContainerPort),
					Name: defaults.PrefixCodewindPerformance + "-http",
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
			Name:      defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.GatekeeperContainerPort),
					Name: defaults.PrefixCodewindGatekeeper + "-http",
				},
			},
		},
	}
	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, service, r.scheme)
	return service
}

// deploymentForCodewindGatekeeper returns a Codewind dployment object
func (r *ReconcileCodewind) deploymentForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, isOnOpenshift bool, keycloakRealm string, keycloakClientID string, keycloakAuthURL string, ingressDomain string) *appsv1.Deployment {
	ls := labelsForCodewindGatekeeper(codewind)
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
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
						Name:            defaults.PrefixCodewindGatekeeper,
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
								Value: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID + "." + ingressDomain,
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

// ingressForCodewindGatekeeper function takes in a Codewind object and returns an Openshift Route for the gatekeeper
func (r *ReconcileCodewind) routeForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, ingressDomain string) *v1.Route {
	ls := labelsForCodewindGatekeeper(codewind)
	weight := int32(100)
	route := &v1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
			Namespace: codewind.Namespace,
			Labels:    ls,
		},
		Spec: v1.RouteSpec{
			Host: defaults.PrefixCodewindGatekeeper + "-" + ingressDomain,
			Port: &v1.RoutePort{
				TargetPort: intstr.FromInt(defaults.GatekeeperContainerPort),
			},
			TLS: &v1.TLSConfig{
				InsecureEdgeTerminationPolicy: v1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   v1.TLSTerminationPassthrough,
			},
			To: v1.RouteTargetReference{
				Kind:   "Service",
				Name:   defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
				Weight: &weight,
			},
		},
	}
	// Set Codewind instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, route, r.scheme)
	return route
}

// ingressForCodewindGatekeeper function takes in a Codewind object and returns an Ingress for the gatekeeper
func (r *ReconcileCodewind) ingressForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, ingressDomain string) *extv1beta1.Ingress {
	ls := labelsForCodewindGatekeeper(codewind)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":     "/",
		"ingress.bluemix.net/redirect-to-https":          "True",
		"ingress.bluemix.net/ssl-services":               "ssl-service=" + defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
		"nginx.ingress.kubernetes.io/backend-protocol":   "HTTPS",
		"kubernetes.io/ingress.class":                    "nginx",
		"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
	}
	ingress := &extv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
			Annotations: annotations,
			Namespace:   codewind.Namespace,
			Labels:      ls,
		},
		Spec: extv1beta1.IngressSpec{
			TLS: []extv1beta1.IngressTLS{
				{
					Hosts:      []string{defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID + "." + ingressDomain},
					SecretName: "secret-codewind-tls" + "-" + codewind.Spec.WorkspaceID,
				},
			},
			Rules: []extv1beta1.IngressRule{
				{
					Host: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID + "." + ingressDomain,
					IngressRuleValue: extv1beta1.IngressRuleValue{
						HTTP: &extv1beta1.HTTPIngressRuleValue{
							Paths: []extv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extv1beta1.IngressBackend{
										ServiceName: defaults.PrefixCodewindGatekeeper + "-" + codewind.Spec.WorkspaceID,
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
func (r *ReconcileCodewind) buildGatekeeperSecretTLS(codewind *codewindv1alpha1.Codewind, ingressDomain string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(codewind)
	pemPrivateKey, pemPublicCert, _ := util.GenerateCertificate(defaults.PrefixCodewindGatekeeper+"-"+codewind.Spec.WorkspaceID+"."+ingressDomain, "Codewind"+"-"+codewind.Spec.WorkspaceID)
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
	return map[string]string{"app": defaults.PrefixCodewindPFE, "codewindWorkspace": codewind.Spec.WorkspaceID}
}

func labelsForCodewindPerformance(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindPerformance, "codewindWorkspace": codewind.Spec.WorkspaceID}
}

func labelsForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindGatekeeper, "codewindWorkspace": codewind.Spec.WorkspaceID}
}

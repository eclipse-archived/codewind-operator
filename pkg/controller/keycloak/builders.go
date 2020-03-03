package keycloak

import (
	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// serviceAccountForKeycloak function takes in a Keycloak object and returns a serviceAccount for that object.
func (r *ReconcileKeycloak) serviceAccountForKeycloak(keycloak *codewindv1alpha1.Keycloak) *corev1.ServiceAccount {
	ls := labelsForKeycloak(keycloak.Name)

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		Secrets: nil,
	}

	// Set Keycloak instance as the owner of the service account.
	controllerutil.SetControllerReference(keycloak, serviceAccount, r.scheme)
	return serviceAccount
}

// pvcForKeycloak function takes in a Keycloak object and returns a PVC for that object.
func (r *ReconcileKeycloak) pvcForKeycloak(keycloak *codewindv1alpha1.Keycloak) *corev1.PersistentVolumeClaim {
	ls := labelsForKeycloak(keycloak.Name)

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-keycloak-pvc-" + keycloak.Spec.WorkspaceID,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	// Set Keycloak instance as the owner of the persistent volume claim.
	controllerutil.SetControllerReference(keycloak, pvc, r.scheme)
	return pvc
}

// secretsForKeycloak function takes in a Keycloak object and returns a Secret for that object.
func (r *ReconcileKeycloak) secretsForKeycloak(keycloak *codewindv1alpha1.Keycloak) *corev1.Secret {
	ls := labelsForKeycloak(keycloak.Name)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-keycloak-user-" + keycloak.Spec.WorkspaceID,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		StringData: map[string]string{
			"keycloak-admin-user":     "admin",
			"keycloak-admin-password": "admin",
		},
	}
	// Set Keycloak instance as the owner of the Secret.
	controllerutil.SetControllerReference(keycloak, secret, r.scheme)
	return secret
}

// serviceForKeycloak function takes in a Keycloak object and returns a Service for that object.
func (r *ReconcileKeycloak) serviceForKeycloak(keycloak *codewindv1alpha1.Keycloak) *corev1.Service {
	ls := labelsForKeycloak(keycloak.Name)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
			Namespace: keycloak.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.KeycloakContainerPort),
					Name: "codewind-keycloak-http",
				},
			},
		},
	}
	// Set Keycloak instance as the owner of the Service.
	controllerutil.SetControllerReference(keycloak, service, r.scheme)
	return service
}

// deploymentForKeycloak returns a Keycloak object
func (r *ReconcileKeycloak) deploymentForKeycloak(keycloak *codewindv1alpha1.Keycloak) *appsv1.Deployment {
	ls := labelsForKeycloak(keycloak.Name)
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
			Namespace: keycloak.Namespace,
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
					ServiceAccountName: "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
					Volumes: []corev1.Volume{
						{
							Name: "keycloak-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "codewind-keycloak-pvc-" + keycloak.Spec.WorkspaceID,
								},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:            "codewind-keycloak",
						Image:           defaults.KeycloakImage + ":" + defaults.KeycloakImageTag,
						ImagePullPolicy: corev1.PullAlways,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "keycloak-data",
								MountPath: "/opt/jboss/keycloak/standalone/data",
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "KEYCLOAK_USER",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-keycloak-user" + "-" + keycloak.Spec.WorkspaceID}, Key: "keycloak-admin-user"}},
							},
							{
								Name: "KEYCLOAK_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-keycloak-user" + "-" + keycloak.Spec.WorkspaceID}, Key: "keycloak-admin-password"}},
							},
							{
								Name:  "PROXY_ADDRESS_FORWARDING",
								Value: "true",
							},
							{
								Name:  "DB_VENDOR",
								Value: "h2",
							},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: int32(defaults.KeycloakContainerPort)},
						},
					}},
				},
			},
		},
	}
	// Set Keycloak instance as the owner of the Deployment.
	controllerutil.SetControllerReference(keycloak, dep, r.scheme)
	return dep
}

// serviceForKeycloak function takes in a Keycloak object and returns a Service for that object.
func (r *ReconcileKeycloak) ingressForKeycloak(keycloak *codewindv1alpha1.Keycloak) *extv1beta1.Ingress {
	ls := labelsForKeycloak(keycloak.Name)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":     "/",
		"nginx.ingress.kubernetes.io/backend-protocol":   "HTTP",
		"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
		"kubernetes.io/ingress.class":                    "nginx",
	}
	ingress := &extv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
			Namespace:   keycloak.Namespace,
			Annotations: annotations,
			Labels:      ls,
		},
		Spec: extv1beta1.IngressSpec{
			TLS: []extv1beta1.IngressTLS{
				{
					Hosts:      []string{"codewind-keycloak-" + keycloak.Spec.WorkspaceID + "." + keycloak.Spec.IngressDomain},
					SecretName: "secret-keycloak-tls" + "-" + keycloak.Spec.WorkspaceID,
				},
			},
			Rules: []extv1beta1.IngressRule{
				{
					Host: "codewind-keycloak-" + keycloak.Spec.WorkspaceID + "." + keycloak.Spec.IngressDomain,
					IngressRuleValue: extv1beta1.IngressRuleValue{
						HTTP: &extv1beta1.HTTPIngressRuleValue{
							Paths: []extv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extv1beta1.IngressBackend{
										ServiceName: "codewind-keycloak-" + keycloak.Spec.WorkspaceID,
										ServicePort: intstr.FromInt(defaults.KeycloakContainerPort),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set Keycloak instance as the owner of the Service.
	controllerutil.SetControllerReference(keycloak, ingress, r.scheme)
	return ingress
}

// labelsForKeycloak returns the labels for selecting the resources
// belonging to the given keycloak CR name.
func labelsForKeycloak(name string) map[string]string {
	return map[string]string{"app": "codewind-keycloak", "keycloak_cr": name}
}

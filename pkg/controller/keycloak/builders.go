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
	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	"github.com/eclipse/codewind-operator/pkg/util"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// serviceAccountForKeycloak function takes in a Keycloak object and returns a serviceAccount for that object.
func (r *ReconcileKeycloak) serviceAccountForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *corev1.ServiceAccount {
	ls := labelsForKeycloak(keycloak)

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakServiceAccountName,
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
func (r *ReconcileKeycloak) pvcForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak, storageClassName string, storageKeycloakSize string) *corev1.PersistentVolumeClaim {
	ls := labelsForKeycloak(keycloak)

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakPVCName,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageKeycloakSize),
				},
			},
		},
	}

	// If a storage class was passed in, set it in the PVC
	if storageClassName != "" {
		pvc.Spec.StorageClassName = &storageClassName
	}

	// Set Keycloak instance as the owner of the persistent volume claim.
	controllerutil.SetControllerReference(keycloak, pvc, r.scheme)
	return pvc
}

// secretsForKeycloak function takes in a Keycloak object and returns a Secret for that object.
func (r *ReconcileKeycloak) secretsForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *corev1.Secret {
	ls := labelsForKeycloak(keycloak)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakSecretsName,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		StringData: map[string]string{
			"keycloak-admin-user":     "admin",
			"keycloak-admin-password": "admin",
		},
	}
	// Set Keycloak instance as the owner of the secret.
	controllerutil.SetControllerReference(keycloak, secret, r.scheme)
	return secret
}

// secretsTLSForKeycloak function takes in a Keycloak object and returns a TLS Secret for that object.
func (r *ReconcileKeycloak) secretsTLSForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *corev1.Secret {
	ls := labelsForKeycloak(keycloak)
	pemPrivateKey, pemPublicCert, _ := util.GenerateCertificate(deploymentOptions.KeycloakIngressHost, deploymentOptions.KeycloakTLSCertTitle)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakTLSSecretsName,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		StringData: map[string]string{
			"tls.crt": pemPublicCert,
			"tls.key": pemPrivateKey,
		},
	}
	// Set Keycloak instance as the owner of the secret.
	controllerutil.SetControllerReference(keycloak, secret, r.scheme)
	return secret
}

// serviceForKeycloak function takes in a Keycloak object and returns a Service for that object.
func (r *ReconcileKeycloak) serviceForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *corev1.Service {
	ls := labelsForKeycloak(keycloak)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakServiceName,
			Namespace: keycloak.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{
					Port: int32(defaults.KeycloakContainerPort),
					Name: defaults.PrefixCodewindKeycloak + "-http",
				},
			},
		},
	}
	// Set Keycloak instance as the owner of the service.
	controllerutil.SetControllerReference(keycloak, service, r.scheme)
	return service
}

// deploymentForKeycloak returns a Keycloak object
func (r *ReconcileKeycloak) deploymentForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *appsv1.Deployment {
	ls := labelsForKeycloak(keycloak)
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.KeycloakDeploymentName,
			Namespace: keycloak.Namespace,
			Labels:    ls,
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
					ServiceAccountName: deploymentOptions.KeycloakServiceAccountName,
					Volumes: []corev1.Volume{
						{
							Name: "keycloak-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: deploymentOptions.KeycloakPVCName,
								},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:            defaults.PrefixCodewindKeycloak,
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
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: deploymentOptions.KeycloakSecretsName}, Key: "keycloak-admin-user"}},
							},
							{
								Name: "KEYCLOAK_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: deploymentOptions.KeycloakSecretsName}, Key: "keycloak-admin-password"}},
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
	// Set Keycloak instance as the owner of the deployment.
	controllerutil.SetControllerReference(keycloak, dep, r.scheme)
	return dep
}

// serviceForKeycloak function takes in a Keycloak object and returns a Service for that object.
func (r *ReconcileKeycloak) routeForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *routev1.Route {
	ls := labelsForKeycloak(keycloak)
	weight := int32(100)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":     "/",
		"nginx.ingress.kubernetes.io/backend-protocol":   "HTTP",
		"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
		"kubernetes.io/ingress.class":                    "nginx",
	}
	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentOptions.KeycloakIngressName,
			Namespace:   keycloak.Namespace,
			Annotations: annotations,
			Labels:      ls,
		},
		Spec: routev1.RouteSpec{
			Host: deploymentOptions.KeycloakIngressHost,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(defaults.KeycloakContainerPort),
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   deploymentOptions.KeycloakServiceName,
				Weight: &weight,
			},
		},
	}

	// Set Keycloak instance as the owner of the route.
	controllerutil.SetControllerReference(keycloak, route, r.scheme)
	return route
}

// serviceForKeycloak function takes in a Keycloak object and returns a Service for that object.
func (r *ReconcileKeycloak) ingressForKeycloak(keycloak *codewindv1alpha1.Keycloak, deploymentOptions DeploymentOptionsKeycloak) *extv1beta1.Ingress {
	ls := labelsForKeycloak(keycloak)
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
			Name:        deploymentOptions.KeycloakIngressName,
			Namespace:   keycloak.Namespace,
			Annotations: annotations,
			Labels:      ls,
		},
		Spec: extv1beta1.IngressSpec{
			TLS: []extv1beta1.IngressTLS{
				{
					Hosts:      []string{deploymentOptions.KeycloakIngressHost},
					SecretName: deploymentOptions.KeycloakTLSSecretsName,
				},
			},
			Rules: []extv1beta1.IngressRule{
				{
					Host: deploymentOptions.KeycloakIngressHost,
					IngressRuleValue: extv1beta1.IngressRuleValue{
						HTTP: &extv1beta1.HTTPIngressRuleValue{
							Paths: []extv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extv1beta1.IngressBackend{
										ServiceName: deploymentOptions.KeycloakServiceName,
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

	// Set Keycloak instance as the owner of the ingress.
	controllerutil.SetControllerReference(keycloak, ingress, r.scheme)
	return ingress
}

// labelsForKeycloak returns the labels for selecting the resources
// belonging to the given keycloak CR name.
func labelsForKeycloak(keycloak *codewindv1alpha1.Keycloak) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindKeycloak, "authName": keycloak.Name, "authID": keycloak.GetAnnotations()["authID"]}
}

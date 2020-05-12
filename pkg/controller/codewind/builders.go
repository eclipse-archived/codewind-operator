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

package codewind

import (
	"strconv"
	"strings"

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

// serviceAccountForCodewind function takes in a Codewind object and returns a serviceAccount for that object.
func (r *ReconcileCodewind) serviceAccountForCodewind(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *corev1.ServiceAccount {
	labels := map[string]string{"app": "codewind-" + deploymentOptions.WorkspaceID, "codewind_cr": codewind.Name, "codewindWorkspace": deploymentOptions.WorkspaceID}
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindServiceAccountName,
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
func (r *ReconcileCodewind) pvcForCodewind(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, storageClassName string, storageSize string) *corev1.PersistentVolumeClaim {
	labels := labelsForCodewindPFE(deploymentOptions)
	if codewind.Spec.StorageSize != "" {
		storageSize = codewind.Spec.StorageSize
	}
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindPFEPVCName,
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

func (r *ReconcileCodewind) deploymentForCodewindPerformance(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, ingressDomain string) *appsv1.Deployment {
	ls := labelsForCodewindPerformance(deploymentOptions)
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindPerformance + "-" + deploymentOptions.WorkspaceID,
			Namespace: codewind.Namespace,
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
					ServiceAccountName: deploymentOptions.CodewindServiceAccountName,
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
								Value: defaults.PrefixCodewindGatekeeper + deploymentOptions.WorkspaceID + "." + ingressDomain,
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
func (r *ReconcileCodewind) deploymentForCodewindPFE(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, isOnOpenshift bool, keycloakRealm string, authHost string, logLevel string, ingressDomain string) *appsv1.Deployment {
	ls := labelsForCodewindPFE(deploymentOptions)
	replicas := int32(1)
	runAsPrivileged := true
	loglevel := "info"
	if codewind.Spec.LogLevel != "" {
		loglevel = codewind.Spec.LogLevel
	}
	volumes := []corev1.Volume{
		{
			Name: "shared-workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: defaults.PrefixCodewindPFE + "-pvc-" + deploymentOptions.WorkspaceID,
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
			SubPath:   deploymentOptions.WorkspaceID + "/projects",
		},
		{
			Name:      "buildah-volume",
			MountPath: "/var/lib/containers",
		},
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindPFEDeploymentName,
			Namespace: codewind.Namespace,
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
					ServiceAccountName: deploymentOptions.CodewindServiceAccountName,
					Volumes:            volumes,
					Containers: []corev1.Container{{
						Name:            defaults.PrefixCodewindPFE,
						Image:           defaults.CodewindImage + ":" + defaults.CodewindImageTag,
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: &corev1.SecurityContext{
							Privileged: &runAsPrivileged,
						},
						VolumeMounts: volumeMounts,
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
								Value: deploymentOptions.WorkspaceID,
							},
							{
								Name:  "PVC_NAME",
								Value: deploymentOptions.CodewindPFEPVCName,
							},
							{
								Name:  "SERVICE_NAME",
								Value: deploymentOptions.CodewindPFEServiceName,
							},
							{
								Name:  "SERVICE_ACCOUNT_NAME",
								Value: deploymentOptions.CodewindServiceAccountName,
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
								Value: "codewind-" + deploymentOptions.WorkspaceID,
							},
							{
								Name:  "OWNER_REF_UID",
								Value: string(codewind.GetUID()),
							},
							{
								Name:  "CODEWIND_PERFORMANCE_SERVICE",
								Value: defaults.PrefixCodewindPerformance + "-" + deploymentOptions.WorkspaceID,
							},
							{
								Name:  "CHE_INGRESS_HOST",
								Value: defaults.PrefixCodewindGatekeeper + "-" + deploymentOptions.WorkspaceID + "." + ingressDomain,
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
func (r *ReconcileCodewind) serviceForCodewindPFE(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *corev1.Service {
	ls := labelsForCodewindPFE(deploymentOptions)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindPFEServiceName,
			Namespace: codewind.Namespace,
			Labels:    ls,
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
func (r *ReconcileCodewind) serviceForCodewindPerformance(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *corev1.Service {
	ls := labelsForCodewindPerformance(deploymentOptions)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindPerformanceServiceName,
			Namespace: codewind.Namespace,
			Labels:    ls,
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
func (r *ReconcileCodewind) serviceForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *corev1.Service {
	ls := labelsForCodewindGatekeeper(deploymentOptions)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindGatekeeperServiceName,
			Namespace: codewind.Namespace,
			Labels:    ls,
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

// deploymentForCodewindGatekeeper returns a Codewind deployment object
func (r *ReconcileCodewind) deploymentForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, isOnOpenshift bool, keycloakRealm string, keycloakClientID string, keycloakAuthURL string, ingressDomain string) *appsv1.Deployment {
	ls := labelsForCodewindGatekeeper(deploymentOptions)
	replicas := int32(1)

	// Replace any dash characters in the WorkspaceID to understore characters to match variable formats created by Kubernetes
	workspaceServiceSuffix := strings.ReplaceAll(strings.ToUpper(deploymentOptions.WorkspaceID), "-", "_")

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.PrefixCodewindGatekeeper + "-" + deploymentOptions.WorkspaceID,
			Namespace: codewind.Namespace,
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
					ServiceAccountName: deploymentOptions.CodewindServiceAccountName,
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
								Value: deploymentOptions.CodewindGatekeeperIngressHost,
							},
							{
								Name:  "WORKSPACE_SERVICE",
								Value: "CODEWIND_PFE_" + workspaceServiceSuffix,
							},
							{
								Name:  "WORKSPACE_ID",
								Value: deploymentOptions.WorkspaceID,
							},
							{
								Name:  "ACCESS_ROLE", // Keycloak access role that grants user to this Codewind deployment
								Value: "codewind-" + deploymentOptions.WorkspaceID,
							},
							{
								Name: "CLIENT_SECRET",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: deploymentOptions.CodewindGatekeeperSecretAuthName}, Key: "client_secret"}},
							},
							{
								Name: "SESSION_SECRET",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: deploymentOptions.CodewindGatekeeperSecretSessionName}, Key: "session_secret"}},
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
func (r *ReconcileCodewind) routeForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, ingressDomain string) *routev1.Route {
	ls := labelsForCodewindGatekeeper(deploymentOptions)
	weight := int32(100)
	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindGatekeeperIngressName,
			Namespace: codewind.Namespace,
			Labels:    ls,
		},
		Spec: routev1.RouteSpec{
			Host: deploymentOptions.CodewindGatekeeperIngressHost,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(defaults.GatekeeperContainerPort),
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationPassthrough,
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   deploymentOptions.CodewindGatekeeperServiceName,
				Weight: &weight,
			},
		},
	}
	// Set Codewind instance as the owner of the route.
	controllerutil.SetControllerReference(codewind, route, r.scheme)
	return route
}

// ingressForCodewindGatekeeper function takes in a Codewind object and returns an Ingress for the gatekeeper
func (r *ReconcileCodewind) ingressForCodewindGatekeeper(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, ingressDomain string) *extv1beta1.Ingress {
	ls := labelsForCodewindGatekeeper(deploymentOptions)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target":     "/",
		"ingress.bluemix.net/redirect-to-https":          "True",
		"ingress.bluemix.net/ssl-services":               "ssl-service=" + deploymentOptions.CodewindGatekeeperServiceName,
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
			Name:        deploymentOptions.CodewindGatekeeperIngressName,
			Annotations: annotations,
			Namespace:   codewind.Namespace,
			Labels:      ls,
		},
		Spec: extv1beta1.IngressSpec{
			TLS: []extv1beta1.IngressTLS{
				{
					Hosts:      []string{deploymentOptions.CodewindGatekeeperIngressHost},
					SecretName: deploymentOptions.CodewindGatekeeperSecretTLSName,
				},
			},
			Rules: []extv1beta1.IngressRule{
				{
					Host: deploymentOptions.CodewindGatekeeperIngressHost,
					IngressRuleValue: extv1beta1.IngressRuleValue{
						HTTP: &extv1beta1.HTTPIngressRuleValue{
							Paths: []extv1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extv1beta1.IngressBackend{
										ServiceName: deploymentOptions.CodewindGatekeeperServiceName,
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
	// Set Codewind instance as the owner of the ingress.
	controllerutil.SetControllerReference(codewind, ingress, r.scheme)
	return ingress
}

// buildGatekeeperSessionSecret :  builds a session secret for gatekeeper
func (r *ReconcileCodewind) buildGatekeeperSecretSession(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, sessionSecretValue string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(deploymentOptions)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindGatekeeperSecretSessionName,
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
func (r *ReconcileCodewind) buildGatekeeperSecretTLS(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, ingressDomain string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(deploymentOptions)
	pemPrivateKey, pemPublicCert, _ := util.GenerateCertificate(deploymentOptions.CodewindGatekeeperIngressHost, deploymentOptions.CodewindGatekeeperTLSCertTitle)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindGatekeeperSecretTLSName,
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
func (r *ReconcileCodewind) buildGatekeeperSecretAuth(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, keycloakClientKey string) *corev1.Secret {
	metaLabels := labelsForCodewindGatekeeper(deploymentOptions)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindGatekeeperSecretAuthName,
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
func labelsForCodewindPFE(deploymentOptions DeploymentOptionsCodewind) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindPFE, "codewindWorkspace": deploymentOptions.WorkspaceID}
}

func labelsForCodewindPerformance(deploymentOptions DeploymentOptionsCodewind) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindPerformance, "codewindWorkspace": deploymentOptions.WorkspaceID}
}

func labelsForCodewindGatekeeper(deploymentOptions DeploymentOptionsCodewind) map[string]string {
	return map[string]string{"app": defaults.PrefixCodewindGatekeeper, "codewindWorkspace": deploymentOptions.WorkspaceID}
}

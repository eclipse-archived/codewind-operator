package codewind

import (
	"strconv"

	"github.com/eclipse/codewind-installer/pkg/appconstants"
	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	// Set Keycloak instance as the owner of the service account.
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

	// Set Keycloak instance as the owner of the persistent volume claim.
	controllerutil.SetControllerReference(codewind, pvc, r.scheme)
	return pvc
}

// deploymentForCodewindPFE returns a Codewind dployment object
func (r *ReconcileCodewind) deploymentForCodewindPFE(codewind *codewindv1alpha1.Codewind, isOnOpenshift bool, keycloakRealm string, authHost string, logLevel string /*, volumeMounts []corev1.VolumeMount*/) *appsv1.Deployment {
	ls := labelsForCodewindPFE(codewind)
	replicas := int32(1)

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
								Value: "codewind-gatekeeper" + "." + codewind.Spec.IngressDomain,
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
								Value: logLevel,
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
	// Set Keycloak instance as the owner of the Deployment.
	controllerutil.SetControllerReference(codewind, dep, r.scheme)
	return dep
}

// serviceForCodewindPFE function takes in a Codewind PFE object and returns a Service for that object.
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
	// Set Keycloak instance as the owner of the Service.
	controllerutil.SetControllerReference(codewind, service, r.scheme)
	return service
}

// labelsForCodewindPFE returns the labels for selecting the resources
// belonging to the given codewind CR name.
func labelsForCodewindPFE(codewind *codewindv1alpha1.Codewind) map[string]string {
	return map[string]string{"app": "codewind-pfe", "codewind_cr": codewind.Name, "codewindWorkspace": codewind.Spec.WorkspaceID}
}

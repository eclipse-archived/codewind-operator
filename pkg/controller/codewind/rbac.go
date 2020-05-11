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
	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// clusterRolesForCodewind : takes in a Codewind object and returns Cluster roles for that object.
func (r *ReconcileCodewind) clusterRolesForCodewind(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *rbacv1.ClusterRole {
	ourRoles := []rbacv1.PolicyRule{
		rbacv1.PolicyRule{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
			ResourceNames: []string{"privileged", "anyuid"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"extensions", ""},
			Resources: []string{"ingresses", "ingresses/status", "podsecuritypolicies"},
			Verbs:     []string{"delete", "create", "patch", "get", "list", "update", "watch", "use"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"delete", "create", "patch", "get", "list"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"pods", "pods/portforward", "pods/log", "pods/exec"},
			Verbs:     []string{"get", "list", "create", "delete", "watch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "create", "watch", "delete", "patch", "update"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"serviceaccounts"},
			Verbs:     []string{"get", "patch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"services"},
			Verbs:     []string{"get", "list", "create", "delete", "patch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"get", "list", "create", "update", "delete", "patch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims", "persistentvolumeclaims/finalizers", "persistentvolumeclaims/status"},
			Verbs:     []string{"*"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"icp.ibm.com"},
			Resources: []string{"images"},
			Verbs:     []string{"get", "list", "create", "watch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"apps", "extensions"},
			Resources: []string{"deployments", "deployments/finalizers"},
			Verbs:     []string{"watch", "get", "list", "create", "update", "delete", "patch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"extensions", "apps"},
			Resources: []string{"replicasets", "replicasets/finalizers"},
			Verbs:     []string{"get", "list", "update", "delete"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"rolebindings", "roles", "clusterroles"},
			Verbs:     []string{"create", "get", "patch", "list"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch", "update"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"route.openshift.io"},
			Resources: []string{"routes", "routes/custom-host"},
			Verbs:     []string{"get", "list", "create", "delete", "watch", "patch", "update"},
		},
	}
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: codewind.Namespace,
			Name:      deploymentOptions.CodewindRolesName,
		},
		Rules: ourRoles,
	}
}

// clusterRolesForCodewindTekton : create Codewind Tekton cluster roles
func (r *ReconcileCodewind) clusterRolesForCodewindTekton(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *rbacv1.ClusterRole {
	ourRoles := []rbacv1.PolicyRule{
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"services"},
			Verbs:     []string{"get", "list"},
		},
	}
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentOptions.CodewindTektonClusterRolesName,
		},
		Rules: ourRoles,
	}
}

//roleBindingForCodewind : create Codewind role bindings in the deployment namespace
func (r *ReconcileCodewind) roleBindingForCodewind(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *rbacv1.RoleBinding {
	labels := labelsForCodewindPFE(deploymentOptions)
	rolebinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindRoleBindingName,
			Labels:    labels,
			Namespace: codewind.Namespace,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      deploymentOptions.CodewindServiceAccountName,
				Namespace: codewind.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     deploymentOptions.CodewindRolesName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	// Set Codewind instance as the owner of these role bindings.
	controllerutil.SetControllerReference(codewind, rolebinding, r.scheme)
	return rolebinding
}

//roleBindingForCodewindTekton : create Codewind Tekton cluster role bindings
func (r *ReconcileCodewind) roleBindingForCodewindTekton(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind) *rbacv1.ClusterRoleBinding {
	labels := labelsForCodewindPFE(deploymentOptions)
	rolebinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentOptions.CodewindTektonRoleBindingName,
			Labels:    labels,
			Namespace: codewind.Namespace,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      deploymentOptions.CodewindServiceAccountName,
				Namespace: codewind.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     deploymentOptions.CodewindTektonClusterRolesName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return rolebinding
}

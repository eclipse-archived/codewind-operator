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
	"context"

	codewindv1alpha1 "github.com/eclipse/codewind-operator/pkg/apis/codewind/v1alpha1"
	defaults "github.com/eclipse/codewind-operator/pkg/controller/defaults"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// addCodewindFinalizer : Adds the finalizer to the metadata of the Codewind CR
func (r *ReconcileCodewind) addCodewindFinalizer(reqLogger logr.Logger, codewind *codewindv1alpha1.Codewind, request reconcile.Request) error {
	if len(codewind.GetFinalizers()) < 1 && codewind.GetDeletionTimestamp() == nil {
		reqLogger.Info("Adding Finalizer to Codewind", "namespace", codewind.Namespace, "name", codewind.Name, "finalizer", defaults.CodewindFinalizerName)
		codewind.SetFinalizers([]string{defaults.CodewindFinalizerName})
		err := r.client.Update(context.TODO(), codewind)
		if err != nil {
			reqLogger.Error(err, "Failed to update Codewind with the CRB finalizer", "namespace", codewind.Namespace, "name", codewind.Name, "finalizer", defaults.CodewindFinalizerName)
			return err
		}
	}
	return nil
}

// removeFinalizers  : Removes all the finalizers from the Codewind CR
func (r *ReconcileCodewind) removeFinalizers(codewind *codewindv1alpha1.Codewind) error {
	codewind.SetFinalizers(nil)
	err := r.client.Update(context.TODO(), codewind)
	if err != nil {
		return err
	}
	return nil
}

// handleCodewindCRBFinalizer : Perform cleanup of cluster role bindings
func (r *ReconcileCodewind) handleCodewindCRBFinalizer(codewind *codewindv1alpha1.Codewind, deploymentOptions DeploymentOptionsCodewind, reqLogger logr.Logger, request reconcile.Request) error {
	reqLogger.Info("Processing Finalizer", "namespace", codewind.Namespace, "name", codewind.Name, "finalizer", defaults.CodewindFinalizerName)
	if len(codewind.GetFinalizers()) == 0 && codewind.GetDeletionTimestamp() == nil {
		return nil
	}

	// Delete the ODO CRB
	reqLogger.Info("Removing ODO CRB", "namespace", codewind.Namespace, "name", deploymentOptions.CodewindODORoleBindingName, "finalizer", defaults.CodewindFinalizerName)
	crbODO := &rbacv1.ClusterRoleBinding{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.CodewindODORoleBindingName, Namespace: ""}, crbODO)
	if err == nil {
		deleteErr := r.client.Delete(context.TODO(), crbODO)
		if deleteErr != nil {
			reqLogger.Error(err, "Unable to remove the cluster role binding", "namespace", codewind.Namespace, "name", codewind.Name, "crb", deploymentOptions.CodewindODORoleBindingName)
			return err
		}
		reqLogger.Info("Successfully removed ODO CRB", "namespace", codewind.Namespace, "name", deploymentOptions.CodewindODORoleBindingName, "finalizer", defaults.CodewindFinalizerName)
	}

	// Delete the Tekton CRB
	reqLogger.Info("Removing TEKTON CRB", "namespace", codewind.Namespace, "name", deploymentOptions.CodewindTektonRoleBindingName, "finalizer", defaults.CodewindFinalizerName)
	crbTekton := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deploymentOptions.CodewindTektonRoleBindingName, Namespace: ""}, crbTekton)
	if err == nil {
		deleteErr := r.client.Delete(context.TODO(), crbTekton)
		if deleteErr != nil {
			reqLogger.Error(err, "Unable to remove the cluster role binding", "namespace", codewind.Namespace, "name", codewind.Name, "crb", deploymentOptions.CodewindTektonRoleBindingName)
			return err
		}
		reqLogger.Info("Successfully removed TEKTON CRB", "namespace", codewind.Namespace, "name", deploymentOptions.CodewindTektonRoleBindingName, "finalizer", defaults.CodewindFinalizerName)
	}

	err = r.removeFinalizers(codewind)
	if err != nil {
		reqLogger.Error(err, "Failed to remove the Codewind finalizer", "namespace", codewind.Namespace, "name", codewind.Name)
		return err
	}
	reqLogger.Info("Finalizer cleared", "namespace", codewind.Namespace, "name", codewind.Name, "finalizer", defaults.CodewindFinalizerName)

	return nil
}

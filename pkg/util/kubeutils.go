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

package util

import (
	"math/rand"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// DetectOpenShift determines if we're running on an OpenShift cluster
func DetectOpenShift() (isOpenshift bool, isOpenshift4 bool, anError error) {
	apiGroups, err := getAPIList()
	if err != nil {
		return false, false, err
	}
	for _, apiGroup := range apiGroups {
		if apiGroup.Name == "route.openshift.io" {
			isOpenshift = true
		}
		if apiGroup.Name == "config.openshift.io" {
			isOpenshift4 = true
		}
	}
	return
}

func getAPIList() ([]v1.APIGroup, error) {
	discoveryClient, err := getDiscoveryClient()
	if err != nil {
		return nil, err
	}
	apiList, err := discoveryClient.ServerGroups()
	if err != nil {
		return nil, err
	}
	return apiList.Groups, nil
}

func getDiscoveryClient() (*discovery.DiscoveryClient, error) {
	kubeconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(kubeconfig)
}

// CreateTimestamp : Create a timestamp
func CreateTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GetOperatorNamespace : Operator namespace
func GetOperatorNamespace() string {
	operatorNamespace, _ := k8sutil.GetOperatorNamespace()
	if operatorNamespace == "" {
		operatorNamespace = "codewind"
	}
	return operatorNamespace
}

// GenerateRandomString : Generates random characters
func GenerateRandomString(length int) string {
	var options = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	bytes := make([]rune, length)
	rand.Seed(time.Now().UTC().UnixNano())
	for i := range bytes {
		bytes[i] = options[rand.Intn(len(options))]
	}
	return string(bytes)
}

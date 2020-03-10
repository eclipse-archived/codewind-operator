/*******************************************************************************
 * Copyright (c) 2019 IBM Corporation and others.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

package security

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	utils "github.com/eclipse/codewind-operator/pkg/util"
)

// Role : Access role
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Composite   bool   `json:"composite"`
	ClientRole  bool   `json:"clientRole"`
	ContainerID string `json:"containerId"`
}

// SecRoleCreate : Create a new role in Keycloak
// Can return an error and an HTTP code
func SecRoleCreate(httpClient utils.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string, roleName string) (*SecError, int) {

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/roles"

	// Role : Access role
	type NewRole struct {
		Name        string `json:"name"`
		Composite   bool   `json:"composite"`
		ClientRole  bool   `json:"clientRole"`
		ContainerID string `json:"containerId"`
	}

	tempRole := &NewRole{
		Name:        roleName,
		Composite:   false,
		ClientRole:  false,
		ContainerID: keycloakConfig.RealmName,
	}
	jsonRole, err := json.Marshal(tempRole)

	payload := strings.NewReader(string(jsonRole))
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}, 0
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}, res.StatusCode
	}

	if res.StatusCode != http.StatusCreated {
		secErr := errors.New("HTTP " + res.Status)
		return &SecError{errOpConnection, secErr, secErr.Error()}, res.StatusCode
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if string(body) != "" {
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		keycloakAPIError.Error = errOpResponseFormat
		kcError := errors.New(keycloakAPIError.ErrorDescription)
		return &SecError{keycloakAPIError.Error, kcError, kcError.Error()}, res.StatusCode
	}
	return nil, res.StatusCode
}

func getRoleByName(httpClient utils.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string, roleName string) (*Role, *SecError) {

	requestedRole := roleName

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/roles/" + requestedRole
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}

	// check we received a valid response
	if res.StatusCode != http.StatusOK {
		unableToReadErr := errors.New("Bad response")
		return nil, &SecError{errOpConnection, unableToReadErr, unableToReadErr.Error()}
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	// parse the result
	var role *Role
	err = json.Unmarshal([]byte(body), &role)
	if err != nil {
		return nil, &SecError{errOpResponseFormat, err, textUnableToParse}
	}

	// found role
	return role, nil
}

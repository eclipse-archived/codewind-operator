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

package security

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/eclipse/codewind-operator/pkg/util"
	logrus "github.com/sirupsen/logrus"
)

// RegisteredUsers : A collection of registered users
type RegisteredUsers struct {
	Collection []RegisteredUser
}

// RegisteredUser : details of a registered user
type RegisteredUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// SecUserGet : Get user from Keycloak
func SecUserGet(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) (*RegisteredUser, *SecError) {

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/users?username=" + keycloakConfig.DevUsername
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Cache-Control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}

	defer res.Body.Close()

	// handle HTTP status codes
	if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		err = errors.New(string(body))
		return nil, &SecError{errOpResponse, err, err.Error()}
	}

	registeredUsers := RegisteredUsers{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal([]byte(body), &registeredUsers.Collection)
	if err != nil {
		return nil, &SecError{errOpResponseFormat, err, err.Error()}
	}

	registeredUser := RegisteredUser{}

	if len(registeredUsers.Collection) > 0 {
		registeredUser = registeredUsers.Collection[0]
		return &registeredUser, nil
	}

	// user not found
	errNotFound := errors.New(textUserNotFound)
	return nil, &SecError{errOpNotFound, errNotFound, errNotFound.Error()}

}

// SecUserAddRole : Adds a role to a specified user
func SecUserAddRole(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string, roleName string) *SecError {

	// lookup an existing user
	logrus.Tracef("Looking up user : %v", keycloakConfig.DevUsername)
	registeredUser, secErr := SecUserGet(httpClient, keycloakConfig, accessToken)
	if secErr != nil {
		return secErr
	}

	// get the existing role
	existingRole, secErr := getRoleByName(httpClient, keycloakConfig, accessToken, roleName)
	if secErr != nil {
		return secErr
	}

	// build REST request
	logrus.Printf("Adding role '%v' to user : '%v'", existingRole.Name, registeredUser.ID)
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/users/" + registeredUser.ID + "/role-mappings/realm"

	type PayloadRole struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	listOfRoles := []PayloadRole{{ID: existingRole.ID, Name: existingRole.Name}}
	jsonRolesToAdd, err := json.Marshal(listOfRoles)
	payload := strings.NewReader(string(jsonRolesToAdd))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Cache-Control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}

	// handle HTTP status codes (success returns status code StatusNoContent)
	if res.StatusCode != http.StatusNoContent {
		errNotFound := errors.New(res.Status)
		return &SecError{errOpNotFound, errNotFound, errNotFound.Error()}
	}

	return nil
}

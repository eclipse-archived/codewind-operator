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

	"github.com/eclipse/codewind-operator/pkg/util"
)

// AuthToken from the keycloak server after successfully authenticating
type AuthToken struct {
	AccessToken     string `json:"access_token"`
	ExpiresIn       int    `json:"expires_in"`
	RefreshToken    string `json:"refresh_token"`
	TokenType       string `json:"token_type"`
	NotBeforePolicy int    `json:"not-before-policy"`
	SessionState    string `json:"session_state"`
	Scope           string `json:"scope"`
}

// KeycloakConfiguration : Keycloak configuration for an instance of codewind
type KeycloakConfiguration struct {
	RealmName             string
	AuthURL               string
	WorkspaceID           string
	KeycloakAdminPassword string
	KeycloakAdminUsername string
	DevUsername           string
	GatekeeperPublicURL   string
	ClientName            string
}

// SecAuthenticate - sends credentials to the auth server for a specific realm and returns an AuthToken
// connectionRealm can be used to override the supplied context arguments
func SecAuthenticate(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration) (*AuthToken, *SecError) {

	// build REST request to Keycloak
	url := keycloakConfig.AuthURL + "/auth/realms/master/protocol/openid-connect/token"
	payload := strings.NewReader("grant_type=password&client_id=" + KeycloakAdminClientID + "&username=" + keycloakConfig.KeycloakAdminUsername + "&password=" + keycloakConfig.KeycloakAdminPassword)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")

	// send request
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	// Handle special case http status codes
	switch httpCode := res.StatusCode; {
	case httpCode == http.StatusBadRequest, httpCode == http.StatusUnauthorized:
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		kcError := errors.New(string(keycloakAPIError.ErrorDescription))
		return nil, &SecError{keycloakAPIError.Error, kcError, kcError.Error()}
	case httpCode == http.StatusNotFound:
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		kcError := errors.New(string(keycloakAPIError.Error))
		return nil, &SecError{errOpResponse, kcError, kcError.Error()}
	case httpCode == http.StatusServiceUnavailable:
		txtError := errors.New(textAuthIsDown)
		return nil, &SecError{errOpResponse, txtError, txtError.Error()}
	case httpCode != http.StatusOK:
		err = errors.New(string(body))
		return nil, &SecError{errOpResponse, err, err.Error()}
	}

	// Parse and return authtoken
	authToken := AuthToken{}
	err = json.Unmarshal([]byte(body), &authToken)
	if err != nil {
		return nil, &SecError{errOpResponseFormat, err, textUnableToParse}
	}

	return &authToken, nil

}

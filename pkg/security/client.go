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

// RegisteredClients : A collection of registered clients
type RegisteredClients struct {
	Collection []RegisteredClient
}

// RegisteredClient : Registered client
type RegisteredClient struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"clientId"`
	Name         string   `json:"name"`
	RedirectUris []string `json:"redirectUris"`
	WebOrigins   []string `json:"webOrigins"`
}

// RegisteredClientSecret : Client secret
type RegisteredClientSecret struct {
	Type   string `json:"type"`
	Secret string `json:"value"`
}

// SecClientCreate : Create a new client in Keycloak
func SecClientCreate(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string, redirectURL string) *SecError {

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/clients"

	// build the payload (JSON)
	type PayloadClient struct {
		DirectAccessGrantsEnabled bool      `json:"directAccessGrantsEnabled"`
		PublicClient              bool      `json:"publicClient"`
		ClientID                  string    `json:"clientId"`
		Name                      string    `json:"name"`
		RedirectUris              [1]string `json:"redirectUris"`
	}

	tempClient := &PayloadClient{
		DirectAccessGrantsEnabled: true,
		PublicClient:              true,
		ClientID:                  keycloakConfig.ClientName,
		Name:                      keycloakConfig.ClientName,
	}

	tempClient.RedirectUris = [...]string{redirectURL}
	jsonClient, err := json.Marshal(tempClient)
	payload := strings.NewReader(string(jsonClient))
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	if string(body) != "" {
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		keycloakAPIError.Error = errOpResponseFormat
		kcError := errors.New(string(keycloakAPIError.ErrorDescription))
		return &SecError{keycloakAPIError.Error, kcError, kcError.Error()}
	}
	return nil
}

// SecClientGet : Retrieve Client information
func SecClientGet(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) (*RegisteredClient, *SecError) {

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/clients?clientId=" + keycloakConfig.ClientName
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
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

	registeredClients := RegisteredClients{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal([]byte(body), &registeredClients.Collection)
	if err != nil {
		return nil, &SecError{errOpResponseFormat, err, err.Error()}
	}

	registeredClient := RegisteredClient{}
	if len(registeredClients.Collection) > 0 {
		registeredClient = registeredClients.Collection[0]
		return &registeredClient, nil
	}

	return nil, nil
}

// SecClientGetSecret : Retrieve the client secret for the supplied clientID
func SecClientGetSecret(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) (*RegisteredClientSecret, *SecError) {

	registeredClient, secError := SecClientGet(httpClient, keycloakConfig, accessToken)
	if secError != nil {
		return nil, secError
	}

	if registeredClient == nil {
		return nil, nil
	}

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/clients/" + registeredClient.ID + "/client-secret"
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

	registeredClientSecret := RegisteredClientSecret{}
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal([]byte(body), &registeredClientSecret)
	if err != nil {
		return nil, &SecError{errOpResponseFormat, err, err.Error()}
	}

	return &registeredClientSecret, nil
}

// SecClientAppendURL : Append an additional url to the whitelist
func SecClientAppendURL(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) *SecError {

	registeredClient, secErr := SecClientGet(httpClient, keycloakConfig, accessToken)
	if secErr != nil {
		return secErr
	}

	redirectURIs := registeredClient.RedirectUris
	webOrigins := registeredClient.WebOrigins

	redirectURIs = append(redirectURIs, (keycloakConfig.GatekeeperPublicURL + "/*"))
	webOrigins = append(webOrigins, keycloakConfig.GatekeeperPublicURL)

	registeredClient.RedirectUris = redirectURIs
	registeredClient.WebOrigins = webOrigins

	// save the updated client
	jsonClient, err := json.Marshal(registeredClient)
	payload := strings.NewReader(string(jsonClient))
	url := keycloakConfig.AuthURL + "/auth/admin/realms/" + keycloakConfig.RealmName + "/clients/" + registeredClient.ID
	req, err := http.NewRequest("PUT", url, payload)

	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// send request
	res, err := httpClient.Do(req)
	if err != nil {
		return &SecError{errOpConnection, err, err.Error()}
	}
	defer res.Body.Close()
	return nil
}

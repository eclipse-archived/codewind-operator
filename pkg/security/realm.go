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
)

// KeycloakRealm : A Keycloak Realm
type KeycloakRealm struct {
	ID          string `json:"id"`
	Realm       string `json:"realm"`
	DisplayName string `json:"displayName"`
	Enabled     bool   `json:"enabled"`
	LoginTheme  string `json:"loginTheme"`
}

// SecRealmGet : Reads a realm in Keycloak
func SecRealmGet(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) (*KeycloakRealm, *SecError) {
	req, err := http.NewRequest("GET", keycloakConfig.AuthURL+"/auth/admin/realms/"+keycloakConfig.RealmName, nil)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// send request
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, &SecError{errOpConnection, err, err.Error()}
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if res.StatusCode == http.StatusOK {
		// Parse Realm
		keycloakRealm := KeycloakRealm{}
		err = json.Unmarshal([]byte(body), &keycloakRealm)
		if err != nil {
			kcError := errors.New("Error parsing")
			return nil, &SecError{errOpResponseFormat, kcError, kcError.Error()}
		}
		return &keycloakRealm, nil
	}

	if string(body) != "" {
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		keycloakAPIError.Error = errOpResponseFormat
		kcError := errors.New(keycloakAPIError.ErrorDescription)
		return nil, &SecError{keycloakAPIError.Error, kcError, kcError.Error()}
	}

	return nil, nil
}

// SecRealmCreate : Create a new realm in Keycloak
func SecRealmCreate(httpClient util.HTTPClient, keycloakConfig *KeycloakConfiguration, accessToken string) *SecError {

	themeToUse, secErr := GetSuggestedTheme(keycloakConfig.AuthURL, accessToken)
	if secErr != nil {
		return secErr
	}

	// build REST request
	url := keycloakConfig.AuthURL + "/auth/admin/realms"

	// build the payload (JSON)
	type PayloadRealm struct {
		Realm                 string `json:"realm"`
		DisplayName           string `json:"displayName"`
		Enabled               bool   `json:"enabled"`
		LoginTheme            string `json:"loginTheme"`
		AccessTokenLifespan   int    `json:"accessTokenLifespan"`
		SSOSessionIdleTimeout int    `json:"ssoSessionIdleTimeout"`
		SSOSessionMaxLifespan int    `json:"ssoSessionMaxLifespan"`
	}
	tempRealm := &PayloadRealm{
		Realm:                 keycloakConfig.RealmName,
		DisplayName:           keycloakConfig.RealmName,
		Enabled:               true,
		LoginTheme:            themeToUse,
		AccessTokenLifespan:   (1 * 24 * 60 * 60), // access tokens last 1 day
		SSOSessionIdleTimeout: (5 * 24 * 60 * 60), // refresh tokens last 5 days
		SSOSessionMaxLifespan: (5 * 24 * 60 * 60), // refresh tokens last 5 days
	}

	jsonRealm, err := json.Marshal(tempRealm)
	payload := strings.NewReader(string(jsonRealm))
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
	body, err := ioutil.ReadAll(res.Body)
	if string(body) != "" {
		keycloakAPIError := parseKeycloakError(string(body), res.StatusCode)
		keycloakAPIError.Error = errOpResponseFormat
		kcError := errors.New(keycloakAPIError.ErrorDescription)
		return &SecError{keycloakAPIError.Error, kcError, kcError.Error()}
	}
	return nil
}

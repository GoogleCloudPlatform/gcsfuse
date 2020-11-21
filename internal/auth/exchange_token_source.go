// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/oauth2"
)

func fetchIDPool() (idPool string, err error) {
	projectID, err := metadata.ProjectID()
	if err != nil {
		return
	}

	idPool = fmt.Sprintf("%s.svc.id.goog", projectID)
	return
}

func fetchIDProvider() (provider string, err error) {
	projectID, err := metadata.ProjectID()
	if err != nil {
		return
	}

	clusterLocation, err := metadata.InstanceAttributeValue("cluster-location")
	if err != nil {
		return
	}

	clusterName, err := metadata.InstanceAttributeValue("cluster-name")
	if err != nil {
		return
	}

	provider = fmt.Sprintf(
		"https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
		projectID,
		clusterLocation,
		clusterName)

	return
}

func getAudience() (audience string, err error) {
	idPool, err := fetchIDPool()
	if err != nil {
		err = fmt.Errorf("fetchIDPool: %v", err)
		return
	}

	idProvider, err := fetchIDProvider()
	if err != nil {
		err = fmt.Errorf("fetchIDProvider: %v", err)
		return
	}

	audience = fmt.Sprintf("identitynamespace:%s:%s", idPool, idProvider)
	return
}

func newExchangeTokenSource(
	ctx context.Context,
	exchangeToken string,
) (ts oauth2.TokenSource, err error) {
	audience, err := getAudience()
	if err != nil {
		err = fmt.Errorf("getAudience: %v", err)
		return
	}

	k8sTs := &identityBindingTokenSource{
		ctx:           ctx,
		audience:      audience,
		exchangeToken: exchangeToken,
	}
	ts = oauth2.ReuseTokenSource(nil, k8sTs)
	return
}

// Generates identitybindingtoken from securetoken.googleapis.com with
// the exchangeToken.
type identityBindingTokenSource struct {
	ctx           context.Context
	audience      string
	exchangeToken string
}

func (ts *identityBindingTokenSource) Token() (token *oauth2.Token, err error) {
	body, err := json.Marshal(map[string]string{
		"grant_type":           "urn:ietf:params:oauth:grant-type:token-exchange",
		"subject_token_type":   "urn:ietf:params:oauth:token-type:jwt",
		"requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
		"subject_token":        ts.exchangeToken,
		"audience":             ts.audience,
		"scope":                "https://www.googleapis.com/auth/cloud-platform",
	})
	if err != nil {
		return
	}

	req, err := http.NewRequest(
		"POST",
		"https://securetoken.googleapis.com/v1/identitybindingtoken",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Add logging for token client
	logger := log.New(os.Stdout, "TokenSource: ", 0)
	transport := httputil.DebuggingRoundTripper(
		http.DefaultTransport.(httputil.CancellableRoundTripper),
		logger)

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(
			"could not get identitybindingtoken, status: %v", resp.StatusCode)
		defer resp.Body.Close()
		respBody, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Printf("Body: %v\n", respBody)
		}
		return
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if err = json.Unmarshal(respBody, token); err != nil {
		return
	}
	fmt.Printf("ItendityBindingToken: %v\n", token)
	return
}

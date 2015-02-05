// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"
	"google.golang.org/cloud"
)

const (
	// TODO(jacobsa): Change these two.
	clientID     = "862099979392-0oppqb36povstoiadd6aafcr8pa1utfh.apps.googleusercontent.com"
	clientSecret = "-mOOwbKKhqOwUSh8YNCblo5c"

	// Cf. https://developers.google.com/accounts/docs/OAuth2InstalledApp#choosingredirecturi
	clientRedirectUrl = "urn:ietf:wg:oauth:2.0:oob"
)

var fProjectId = flag.String("project_id", "", "GCS project ID owning the bucket.")
var authCode = flag.String("authorization_code", "", "Authorization code provided when authorizing this app. Run without flag set to print URL.")

func getProjectId() (string, error)
func getAuthCode() string

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error) {
	// Set up the OAuth config object.
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  clientRedirectUrl,
		Scopes:       []string{storage.DevstorageFull_controlScope},
		Endpoint:     google.Endpoint,
	}

	// Attempt to exchange the auth code for a token.
	token, err := config.Exchange(oauth2.NoContext, getAuthCode())
	if err != nil {
		return nil, err
	}

	return config.Client(oauth2.NoContext, token), nil
}

// Return a context containing Cloud authentication parameters derived from
// command-line flags. May block on network traffic.
func getAuthContext() (context.Context, error) {
	// Create the HTTP client.
	httpClient, err := getAuthenticatedHttpClient()
	if err != nil {
		return nil, err
	}

	// Find the project ID.
	projectId, err := getProjectId()
	if err != nil {
		return nil, err
	}

	// Create the context.
	return cloud.NewContext(projectId, httpClient), nil
}

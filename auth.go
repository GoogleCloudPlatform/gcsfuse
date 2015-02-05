// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

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
var fAuthCode = flag.String("authorization_code", "", "Authorization code provided when authorizing this app. Run without flag set to print URL.")

func getProjectId() string {
	s := *fProjectId
	if s == "" {
		fmt.Println("You must set -project_id.")
		os.Exit(1)
	}

	return s
}

func getAuthCode(config *oauth2.Config) string {
	s := *fAuthCode
	if s == "" {
		// NOTE(jacobsa): As of 2015-02-05 the documentation for
		// oauth2.Config.AuthCodeURL says that it is required to set this, but as
		// far as I can tell (cf. RFC 6749 §10.12) it is irrelevant for an
		// installed application that doesn't have a meaningful redirect URL.
		const csrfToken = ""

		fmt.Println("You must set -authorization_code.")
		fmt.Println("Visit this URL to obtain a code:")
		fmt.Println("    ", config.AuthCodeURL(csrfToken, oauth2.AccessTypeOffline))
		os.Exit(1)
	}

	return s
}

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
	token, err := config.Exchange(oauth2.NoContext, getAuthCode(config))
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

	// Create the context.
	return cloud.NewContext(getProjectId(), httpClient), nil
}

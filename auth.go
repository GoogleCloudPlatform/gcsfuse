// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/oauthutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"
)

const (
	// TODO(jacobsa): Change these two.
	clientID     = "862099979392-0oppqb36povstoiadd6aafcr8pa1utfh.apps.googleusercontent.com"
	clientSecret = "-mOOwbKKhqOwUSh8YNCblo5c"

	// Cf. https://developers.google.com/accounts/docs/OAuth2InstalledApp#choosingredirecturi
	clientRedirectUrl = "urn:ietf:wg:oauth:2.0:oob"
)

var _ = flag.String("auth_code", "", "Authorization code provided when authorizing this app. Must be set if not in cache. Run without flag set to print URL.")

// Return an OAuth token source based on command-line flags.
func getTokenSource() (ts oauth2.TokenSource, err error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  clientRedirectUrl,
		Scopes:       []string{storage.DevstorageFull_controlScope},
		Endpoint:     google.Endpoint,
	}

	return oauthutil.NewTerribleTokenSource(
		config,
		flag.Lookup("auth_code"),
		".gcsfs.token_cache.json")
}

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error) {
	// Set up a token source.
	tokenSource, err := getTokenSource()
	if err != nil {
		return nil, err
	}

	// Ensure that we fail early if misconfigured by requesting an initial token.
	log.Println("Requesting initial OAuth token.")
	if _, err := tokenSource.Token(); err != nil {
		return nil, fmt.Errorf("Getting initial OAuth token: %v", err)
	}

	// Create the HTTP transport.
	transport := &oauth2.Transport{
		Source: tokenSource,
	}

	// Create the HTTP client.
	client := &http.Client{Transport: transport}

	return client, nil
}

// Return a GCS connection pre-bound with authentication parameters derived
// from command-line flags. May block on network traffic.
func getConn() (gcs.Conn, error) {
	// TODO(jacobsa): A project ID is apparently only needed for creating and
	// listing buckets, presumably since a bucket ID already maps to a unique
	// project ID (cf. http://goo.gl/Plh3rb). So do we need this at all for our
	// use case? Probably not.
	const projectId = "fixme"

	// Create the HTTP client.
	httpClient, err := getAuthenticatedHttpClient()
	if err != nil {
		return nil, err
	}

	// Create the connection.
	return gcs.NewConn(projectId, httpClient)
}

// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"net/http"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/oauthutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"
)

const (
	// TODO(jacobsa): Change these two.
	clientID     = "501259388845-j47fftkfn6lhp4o80ajg38cs8jed2dmj.apps.googleusercontent.com"
	clientSecret = "-z3_0mx4feP2mqOGhRIEk_DN"

	// Cf. https://developers.google.com/accounts/docs/OAuth2InstalledApp#choosingredirecturi
	clientRedirectUrl = "urn:ietf:wg:oauth:2.0:oob"
)

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  clientRedirectUrl,
		Scopes:       []string{storage.DevstorageRead_writeScope},
		Endpoint:     google.Endpoint,
	}

	return oauthutil.NewTerribleHttpClient(
		config,
		".gcsfuse.token_cache.json")
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

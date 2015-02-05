// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/api/storage/v1"
	"google.golang.org/cloud"
)

// TODO(jacobsa): Change these.
const (
	clientId          = "862099979392-0oppqb36povstoiadd6aafcr8pa1utfh.apps.googleusercontent.com"
	clientSecret      = "-mOOwbKKhqOwUSh8YNCblo5c"
	clientRedirectUrl = "urn:ietf:wg:oauth:2.0:oob"
	authURL           = "https://accounts.google.com/o/oauth2/auth"
	tokenURL          = "https://accounts.google.com/o/oauth2/token"
	tokenCachePath    = "~/.gscfs.token_cache.json"
)

var fProjectId = flag.String("project_id", "", "GCS project ID owning the bucket.")

func getProjectId() (string, error)

func configureToken(transport *oauth2.Transport, cache oauth2.TokenCache) error

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error) {
	// Set up the OAuth config object.
	config := &oauth2.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  clientRedirectUrl,
		Scope:        storage.DevstorageFull_controlScope,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		TokenCache:   oauth2.CacheFile(tokenCachePath),
	}

	// Create a transport.
	transport := &oauth2.Transport{
		Config:    config,
		Transport: http.DefaultTransport,
	}

	// Attempt to configure an OAuth token.
	if err := configureToken(transport, config.TokenCache); err != nil {
		return err
	}

	return transport.Client(), nil
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

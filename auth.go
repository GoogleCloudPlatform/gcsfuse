// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
)

func getProjectId() (string, error)

// Return an HTTP client configured with OAuth credentials from command-line
// flags. May block on network traffic.
func getAuthenticatedHttpClient() (*http.Client, error)

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

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
	tokenCachePath = "~/.gcfs.token_cache.json"

	// TODO(jacobsa): Change these two.
	clientID     = "862099979392-0oppqb36povstoiadd6aafcr8pa1utfh.apps.googleusercontent.com"
	clientSecret = "-mOOwbKKhqOwUSh8YNCblo5c"

	// Cf. https://developers.google.com/accounts/docs/OAuth2InstalledApp#choosingredirecturi
	clientRedirectUrl = "urn:ietf:wg:oauth:2.0:oob"
)

var fProjectId = flag.String("project_id", "", "GCS project ID owning the bucket.")
var fAuthCode = flag.String("authorization_code", "", "Authorization code provided when authorizing this app. Must be set if not in cache. Run without flag set to print URL.")

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

// A TokenSource that asks an oauth2.Config object to derive a token based on
// the flag-supplied auth code when necessary.
type authCodeTokenSource struct {
	config *oauth2.Config
}

func (ts *authCodeTokenSource) Token() (*oauth2.Token, error)

// A TokenSource that loads from and writes to an on-disk cache. On cache miss
// or on hit for an invalid token, it falls through to a wrapped source. Should
// be wrapped using oauth2.ReuseTokenSource to avoid reading/writing the cache
// for every request.
type cachingTokenSource struct {
	wrapped oauth2.TokenSource
}

func (ts *cachingTokenSource) Token() (*oauth2.Token, error)

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

	var tokenSource oauth2.TokenSource

	// As a last resort, we need to ask the config object to exchange the
	// flag-supplied authorization code for a token.
	tokenSource = &authCodeTokenSource{config}

	// Ideally though, we'd prefer to retrieve the token from cache to avoid
	// asking the user for a code.
	tokenSource = &cachingTokenSource{tokenSource}

	// Make sure not to consult the cache when a valid token is already lying
	// around.
	tokenSource = oauth2.ReuseTokenSource(nil, tokenSource)

	// Create the HTTP transport.
	transport := &oauth2.Transport{
		Source: tokenSource,
	}

	// Create the HTTP client.
	client := &http.Client{Transport: transport}

	return client, nil
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

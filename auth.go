// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

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

var fAuthCode = flag.String("authorization_code", "", "Authorization code provided when authorizing this app. Must be set if not in cache. Run without flag set to print URL.")

var gTokenCachePath = path.Join(getHomeDir(), ".gcfs.token_cache.json")

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal("user.Current: ", err)
	}

	return usr.HomeDir
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

func (ts *authCodeTokenSource) Token() (*oauth2.Token, error) {
	log.Println("Exchanging auth code for token.")
	return ts.config.Exchange(oauth2.NoContext, getAuthCode(ts.config))
}

// A TokenSource that loads from and writes to an on-disk cache. On cache miss
// or on hit for an invalid token, it falls through to a wrapped source. Should
// be wrapped using oauth2.ReuseTokenSource to avoid reading/writing the cache
// for every request.
type cachingTokenSource struct {
	wrapped oauth2.TokenSource
}

func (ts *cachingTokenSource) Token() (*oauth2.Token, error) {
	// First consult the cache.
	log.Println("Looking up OAuth token in cache.")
	t, err := ts.LookUp()
	if err != nil {
		// Log the error and ignore it.
		log.Println("Error loading from token cache: ", err)
	}

	// Was there a cache hit?
	if t != nil {
		if t.Valid() {
			log.Println("Cache hit when asked for OAuth token.")
			return t, nil
		}

		log.Println("Ignoring invalid (expired?) token from cache.")
	}

	// Ask the wrapped source.
	t, err = ts.wrapped.Token()
	if err != nil {
		return nil, err
	}

	// Insert into cache, then return the token.
	err = ts.Insert(t)
	if err != nil {
		log.Println("Error inserting into token cache: ", err)
	}

	log.Println("Cached OAuth token for later use.")

	return t, nil
}

// Look for a token in the cache. Returns nil, nil on miss.
func (ts *cachingTokenSource) LookUp() (*oauth2.Token, error) {
	// Open the cache file.
	file, err := os.Open(gTokenCachePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the token.
	t := &oauth2.Token{}
	if err := json.NewDecoder(file).Decode(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (ts *cachingTokenSource) Insert(t *oauth2.Token) error {
	const flags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	const perm = 0600

	// Open the cache file.
	file, err := os.OpenFile(gTokenCachePath, flags, perm)
	if err != nil {
		return err
	}

	// Encode the token.
	if err := json.NewEncoder(file).Encode(t); err != nil {
		file.Close()
		return err
	}

	// Close the file.
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

// Return an OAuth token source based on command-line flags.
func getTokenSource() (ts oauth2.TokenSource, err error) {
	// Set up the OAuth config object.
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  clientRedirectUrl,
		Scopes:       []string{storage.DevstorageFull_controlScope},
		Endpoint:     google.Endpoint,
	}

	// As a last resort, we need to ask the config object to exchange the
	// flag-supplied authorization code for a token.
	ts = &authCodeTokenSource{config}

	// Ideally though, we'd prefer to retrieve the token from cache to avoid
	// asking the user for a code.
	ts = &cachingTokenSource{ts}

	// Make sure not to consult the cache when a valid token is already lying
	// around.
	ts = oauth2.ReuseTokenSource(nil, ts)

	return
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

// Return a context containing Cloud authentication parameters derived from
// command-line flags. May block on network traffic.
func getAuthContext() (context.Context, error) {
	// TODO(jacobsa): I don't know what this is for, and it doesn't seem to
	// matter.
	const projectId = "fixme"

	// Create the HTTP client.
	httpClient, err := getAuthenticatedHttpClient()
	if err != nil {
		return nil, err
	}

	// Create the context.
	return cloud.NewContext(projectId, httpClient), nil
}

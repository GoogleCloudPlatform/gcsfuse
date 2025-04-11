package auth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
)

const universeDomainDefault = "googleapis.com"

// getUniverseDomain extracts the universe domain from the credentials JSON.
func getUniverseDomain(ctx context.Context, jsonCreds []byte, scope string) (string, error) {
	creds, err := google.CredentialsFromJSON(ctx, jsonCreds, scope)
	if err != nil {
		return "", fmt.Errorf("getUniverseDomain: unable to parse credentials from JSON: %w", err)
	}

	domain, err := creds.GetUniverseDomain()
	if err != nil {
		return "", fmt.Errorf("getUniverseDomain: failed to retrieve universe domain: %w", err)
	}

	return domain, nil
}

// newTokenSourceFromPath creates an OAuth2 token source from a service account key file.
// It returns the token source and the universe domain associated with the credentials.
func newTokenSourceFromPath(ctx context.Context, keyFilePath string, scope string) (oauth2.TokenSource, string, error) {
	contents, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("newTokenSourceFromPath: failed to read file %q: %w", keyFilePath, err)
	}

	jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
	if err != nil {
		return nil, "", fmt.Errorf("newTokenSourceFromPath: failed to parse JWT config: %w", err)
	}

	domain, err := getUniverseDomain(ctx, contents, scope)
	if err != nil {
		return nil, "", err
	}

	// Use a self-signed JWT for non-default domains
	if domain != universeDomainDefault {
		ts, err := google.JWTAccessTokenSourceWithScope(contents, scope)
		if err != nil {
			return nil, domain, fmt.Errorf("newTokenSourceFromPath: failed to create JWTAccessTokenSource: %w", err)
		}
		return ts, domain, nil
	}

	// Default token source using the OAuth 2.0 flow
	return jwtConfig.TokenSource(ctx), domain, nil
}

// GetTokenSource generates a token source based on the authentication mode:
//   - service account key file
//   - proxy token endpoint
//   - application default credentials
//
// It also returns the associated universe domain.
func GetTokenSource(
	ctx context.Context,
	keyFile string,
	tokenURL string,
	reuseTokenFromURL bool,
) (oauth2.TokenSource, string, error) {
	const scope = storagev1.DevstorageFullControlScope

	var (
		tokenSrc oauth2.TokenSource
		domain   string
		err      error
		method   string
	)

	switch {
	case keyFile != "":
		tokenSrc, domain, err = newTokenSourceFromPath(ctx, keyFile, scope)
		method = "newTokenSourceFromPath"

	case tokenURL != "":
		tokenSrc, err = newProxyTokenSource(ctx, tokenURL, reuseTokenFromURL)
		method = "newProxyTokenSource"

	default:
		var creds *google.Credentials
		creds, err = google.FindDefaultCredentials(ctx, scope)
		method = "FindDefaultCredentials"
		if err == nil {
			tokenSrc = creds.TokenSource
			domain, err = creds.GetUniverseDomain()
			if err != nil {
				err = fmt.Errorf("GetUniverseDomain: %w", err)
			}
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("%s failed: %w", method, err)
	}

	return tokenSrc, domain, nil
}

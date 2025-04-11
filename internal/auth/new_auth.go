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

// TokenProvider holds the token source and the universe domain.
type TokenProvider struct {
	tokenSrc oauth2.TokenSource
	domain   string
}

// TokenSource returns the token source.
func (p *TokenProvider) TokenSource() oauth2.TokenSource {
	return p.tokenSrc
}

// Domain returns the associated universe domain.
func (p *TokenProvider) Domain() string {
	return p.domain
}

// getUniverseDomain extracts the universe domain from the credentials JSON.
func getUniverseDomain(ctx context.Context, contents []byte, scope string) (string, error) {
	creds, err := google.CredentialsFromJSON(ctx, contents, scope)
	if err != nil {
		return "", fmt.Errorf("CredentialsFromJSON(): %w", err)
	}

	domain, err := creds.GetUniverseDomain()
	if err != nil {
		return "", fmt.Errorf("GetUniverseDomain(): %w", err)
	}

	return domain, nil
}

// newTokenProviderFromPath creates a TokenProvider from a service account JSON file.
func newTokenProviderFromPath(ctx context.Context, path string, scope string) (*TokenProvider, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFile(%q): %w", path, err)
	}

	jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
	if err != nil {
		return nil, fmt.Errorf("JWTConfigFromJSON: %w", err)
	}

	domain, err := getUniverseDomain(ctx, contents, scope)
	if err != nil {
		return nil, err
	}

	var ts oauth2.TokenSource
	if domain != universeDomainDefault {
		ts, err = google.JWTAccessTokenSourceWithScope(contents, scope)
		if err != nil {
			return nil, fmt.Errorf("JWTAccessTokenSourceWithScope: %w", err)
		}
	} else {
		ts = jwtConfig.TokenSource(ctx)
	}

	return &TokenProvider{
		tokenSrc: ts,
		domain:   domain,
	}, nil
}

// GetTokenProvider returns a TokenProvider using one of three flows:
// - key file (standard or self-signed JWT)
// - proxy token URL (custom logic via newProxyTokenSource)
// - application default credentials
func GetTokenProvider(
	ctx context.Context,
	keyFile string,
	tokenURL string,
	reuseTokenFromURL bool,
) (*TokenProvider, error) {
	const scope = storagev1.DevstorageFullControlScope

	var (
		ts     oauth2.TokenSource
		domain string
		err    error
	)

	switch {
	case keyFile != "":
		return newTokenProviderFromPath(ctx, keyFile, scope)

	case tokenURL != "":
		ts, err = newProxyTokenSource(ctx, tokenURL, reuseTokenFromURL)
		if err != nil {
			return nil, fmt.Errorf("newProxyTokenSource: %w", err)
		}
		// Proxy flow has no domain info.
		return &TokenProvider{
			tokenSrc: ts,
			domain:   "", // or use a default string like "unknown"
		}, nil

	default:
		creds, err := google.FindDefaultCredentials(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("FindDefaultCredentials: %w", err)
		}
		domain, err = creds.GetUniverseDomain()
		if err != nil {
			return nil, fmt.Errorf("GetUniverseDomain: %w", err)
		}
		return &TokenProvider{
			tokenSrc: creds.TokenSource,
			domain:   domain,
		}, nil
	}
}

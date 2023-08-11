package storageutil

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/auth"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const ProdEndpoint = "https://storage.googleapis.com:443"
const CustomEndpoint = "https://localhost:9000"
const GcsProdHostName = "storage.googleapis.com"

type StorageClientConfig struct {
	ClientProtocol      mountpkg.ClientProtocol
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	TokenSrc            oauth2.TokenSource
	HttpClientTimeout   time.Duration
	MaxRetryDuration    time.Duration
	RetryMultiplier     float64
	UserAgent           string
	Endpoint            *url.URL
	KeyFile             string
	TokenUrl            string
	ReuseTokenFromUrl   bool
}

func GetDefaultStorageClientConfig() (clientConfig StorageClientConfig) {
	return StorageClientConfig{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 100,
		TokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		HttpClientTimeout:   800 * time.Millisecond,
		MaxRetryDuration:    30 * time.Second,
		RetryMultiplier:     2,
		UserAgent:           "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) (GCP:gcsfuse)",
		Endpoint:            nil,
		KeyFile:             "",
		TokenUrl:            "",
		ReuseTokenFromUrl:   true,
	}
}

func CreateHttpClientObj(flags *StorageClientConfig) (httpClient *http.Client, err error) {
	var transport *http.Transport
	// Using http1 makes the client more performant.
	if flags.ClientProtocol == mountpkg.HTTP1 {
		transport = &http.Transport{
			MaxConnsPerHost:     flags.MaxConnsPerHost,
			MaxIdleConnsPerHost: flags.MaxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		// For http2, change in MaxConnsPerHost doesn't affect the performance.
		transport = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   flags.MaxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	tokenSrc, err := createTokenSource(flags)
	if err != nil {
		err = fmt.Errorf("while fetching tokenSource: %w", err)
		return
	}

	// Custom http client for Go Client.
	httpClient = &http.Client{
		Transport: &oauth2.Transport{
			Base:   transport,
			Source: tokenSrc,
		},
		Timeout: flags.HttpClientTimeout,
	}

	// Setting UserAgent through RoundTripper middleware
	httpClient.Transport = &userAgentRoundTripper{
		wrapped:   httpClient.Transport,
		UserAgent: flags.UserAgent,
	}

	return httpClient, err
}

func IsProdEndpoint(endpoint *url.URL) bool {
	return (endpoint == nil) || (endpoint.Hostname() == GcsProdHostName)
}

func createTokenSource(flags *StorageClientConfig) (tokenSrc oauth2.TokenSource, err error) {
	if IsProdEndpoint(flags.Endpoint) {
		return auth.GetTokenSource(context.Background(), flags.KeyFile, flags.TokenUrl, flags.ReuseTokenFromUrl)
	} else {
		return oauth2.StaticTokenSource(&oauth2.Token{}), nil
	}
}

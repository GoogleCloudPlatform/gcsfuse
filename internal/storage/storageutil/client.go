// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storageutil

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const urlSchemeSeparator = "://"

type StorageClientConfig struct {
	/** Common client parameters. */

	// ClientProtocol decides the go-sdk client to create.
	ClientProtocol    cfg.Protocol
	UserAgent         string
	CustomEndpoint    string
	KeyFile           string
	TokenUrl          string
	ReuseTokenFromUrl bool
	MaxRetrySleep     time.Duration
	RetryMultiplier   float64

	/** HTTP client parameters. */
	MaxConnsPerHost            int
	MaxIdleConnsPerHost        int
	MaxRetryAttempts           int
	HttpClientTimeout          time.Duration
	ExperimentalEnableJsonRead bool
	AnonymousAccess            bool

	/** Grpc client parameters. */
	GrpcConnPoolSize int

	// Enabling new API flow for HNS bucket.
	EnableHNS bool
	// EnableGoogleLibAuth indicates whether to use the google library authentication flow
	EnableGoogleLibAuth bool

	ReadStallRetryConfig cfg.ReadStallGcsRetriesConfig

	MetricHandle metrics.MetricHandle

	TracingEnabled bool
}

var (
	// dnsCache stores a single, atomically-updated dnsCacheEntry.
	dnsCache      atomic.Value
	dnsCacheTTL   = 10 * time.Minute // Time-to-live for DNS cache entries.
	defaultDialer = &net.Dialer{}    // The underlying dialer used for network connections.
)

type dnsCacheEntry struct {
	host       string
	addrs      []string
	expiration time.Time
}

func dialContextWithDNSCache(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to split host and port for address %q: %w", addr, err)
	}

	// Atomically load the cached entry.
	cached := dnsCache.Load()
	var entry dnsCacheEntry

	// Check if the cache is valid for the current host and not expired.
	if cached != nil {
		entry = cached.(dnsCacheEntry)
		if entry.host == host && time.Now().Before(entry.expiration) {
			// Cache hit and is valid.
		} else {
			// Cache is for a different host or is expired.
			cached = nil
		}
	}

	// If the cache is empty or invalid, perform a DNS lookup and update the cache.
	if cached == nil {
		addrs, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("dns lookup failed for host %s: %w", host, err)
		}
		entry = dnsCacheEntry{host: host, addrs: addrs, expiration: time.Now().Add(dnsCacheTTL)}
		dnsCache.Store(entry)
	}

	// Dial any of the resolved addresses.
	for _, address := range entry.addrs {
		conn, err := defaultDialer.DialContext(ctx, network, net.JoinHostPort(address, port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("failed to dial any of a host's addresses for %s", host)
}

func CreateHttpClient(storageClientConfig *StorageClientConfig, tokenSrc oauth2.TokenSource) (httpClient *http.Client, err error) {
	var transport *http.Transport
	// Using http1 makes the client more performant.
	if storageClientConfig.ClientProtocol == cfg.HTTP1 {
		transport = &http.Transport{
			DialContext:         dialContextWithDNSCache,
			Proxy:               http.ProxyFromEnvironment,
			MaxConnsPerHost:     storageClientConfig.MaxConnsPerHost,
			MaxIdleConnsPerHost: storageClientConfig.MaxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		// For http2, change in MaxConnsPerHost doesn't affect the performance.
		transport = &http.Transport{
			DialContext:       dialContextWithDNSCache,
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			MaxConnsPerHost:   storageClientConfig.MaxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	if storageClientConfig.AnonymousAccess {
		// UserAgent will not be added if authentication is disabled.
		// Bypassing authentication prevents the creation of an HTTP transport
		// because it requires a token source.
		// Setting a dummy token would conflict with the "WithoutAuthentication" option.
		// While the "WithUserAgent" option could set a custom User-Agent, it's incompatible
		// with the "WithHTTPClient" option, preventing the direct injection of a user agent
		// when authentication is skipped.
		httpClient = &http.Client{
			Timeout: storageClientConfig.HttpClientTimeout,
		}
	} else {
		if tokenSrc == nil {
			// CreateTokenSource only if tokenSrc is nil, which means it wasn't provided externally.
			// This indicates the EnableGoogleLibAuth flag is disabled.
			tokenSrc, err = CreateTokenSource(storageClientConfig)
			if err != nil {
				err = fmt.Errorf("while fetching tokenSource: %w", err)
				return nil, err
			}
		}

		// Custom http client for Go Client.
		httpClient = &http.Client{
			Transport: &oauth2.Transport{
				Base:   transport,
				Source: tokenSrc,
			},
			Timeout: storageClientConfig.HttpClientTimeout,
		}
		// Setting UserAgent through RoundTripper middleware
		httpClient.Transport = &userAgentRoundTripper{
			wrapped:   httpClient.Transport,
			UserAgent: storageClientConfig.UserAgent,
		}

		if storageClientConfig.TracingEnabled {
			httpClient.Transport = otelhttp.NewTransport(httpClient.Transport, otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}), otelhttp.WithTracerProvider(otel.GetTracerProvider()))
		}
	}
	return httpClient, err
}

// It creates the token-source from the provided
// key-file or using ADC search order (https://cloud.google.com/docs/authentication/application-default-credentials#order).
func CreateTokenSource(storageClientConfig *StorageClientConfig) (tokenSrc oauth2.TokenSource, err error) {
	return auth.GetTokenSource(context.Background(), storageClientConfig.KeyFile, storageClientConfig.TokenUrl, storageClientConfig.ReuseTokenFromUrl)
}

// StripScheme strips the scheme part of given url.
func StripScheme(url string) string {
	// Don't strip off the scheme part for google-internal schemes.
	if strings.HasPrefix(url, "dns:///") || strings.HasPrefix(url, "google-c2p:///") || strings.HasPrefix(url, "google:///") {
		return url
	}
	if strings.Contains(url, urlSchemeSeparator) {
		url = strings.SplitN(url, urlSchemeSeparator, 2)[1]
	}
	return url
}

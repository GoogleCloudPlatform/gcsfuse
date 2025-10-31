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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	dns "github.com/ncruces/go-dns"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// ConfigureDialerWithLocalAddr resolves the provided socket address and returns a net.TCPAddr.
// The port can be 0, in which case the OS will choose a local port.
// The format of SocketAddress is expected to be IP address.
func ConfigureDialerWithLocalAddr(dialer *net.Dialer, socketAddress string) error {
	localAddr, err := net.ResolveTCPAddr("tcp", socketAddress+":0")
	if err != nil {
		return fmt.Errorf("failed to resolve socket address %q: %w", socketAddress, err)
	}
	dialer.LocalAddr = localAddr
	return nil
}

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
	SocketAddress     string

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

	EnableHTTPDNSCache bool
}

func CreateHttpClient(storageClientConfig *StorageClientConfig, tokenSrc oauth2.TokenSource) (httpClient *http.Client, err error) {
	dialer := net.Dialer{}
	if storageClientConfig.SocketAddress != "" {
		if err := ConfigureDialerWithLocalAddr(&dialer, storageClientConfig.SocketAddress); err != nil {
			return nil, fmt.Errorf("failed to configure dialer with socket address %q: %w", storageClientConfig.SocketAddress, err)
		}
	}
	if storageClientConfig.EnableHTTPDNSCache {
		dialer.Resolver = dns.NewCachingResolver(nil, dns.MinCacheTTL(1*time.Minute))
	}

	var transport *http.Transport
	// Using http1 makes the client more performant.
	if storageClientConfig.ClientProtocol == cfg.HTTP1 {
		transport = &http.Transport{
			DialContext:         dialer.DialContext,
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
			DialContext:       dialer.DialContext,
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

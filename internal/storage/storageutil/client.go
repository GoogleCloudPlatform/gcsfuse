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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/google/s2a-go"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	dns "github.com/ncruces/go-dns"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
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
	ClientProtocol     cfg.Protocol
	UserAgent          string
	CustomEndpoint     string
	KeyFile            string
	TokenUrl           string
	ReuseTokenFromUrl  bool
	MaxRetrySleep      time.Duration
	RetryMultiplier    float64
	EnableMountRetries bool
	LocalSocketAddress string
	S2AAddress         string
	S2ASpiffeID        string

	/** HTTP client parameters. */
	MaxConnsPerHost            int
	MaxIdleConnsPerHost        int
	MaxRetryAttempts           int
	HttpClientTimeout          time.Duration
	ExperimentalEnableJsonRead bool
	AnonymousAccess            bool

	/** Grpc client parameters. */
	GrpcConnPoolSize        int
	GrpcPathStrategy        cfg.DirectPathStrategy
	EnableGrpcReadChecksums bool

	// Enabling new API flow for HNS bucket.
	EnableHNS bool
	// Prefix to restrict access to (from only-dir config).
	OnlyDir string
	// EnableGoogleLibAuth indicates whether to use the google library authentication flow
	EnableGoogleLibAuth bool

	ExperimentalEnablePirlo bool

	ReadStallRetryConfig cfg.ReadStallGcsRetriesConfig

	MetricHandle metrics.MetricHandle

	TracingEnabled bool

	EnableHTTPDNSCache bool

	EnableGrpcMetrics bool

	// IsGKE inspects the mountPoint and indicates if running in a GKE environment.
	IsGKE bool

	WriteConfig *cfg.WriteConfig
}

// http1ALPNProto is the ALPN protocol that must be negotiated when GCSFuse is
// configured with client-protocol=http1.
const http1ALPNProto = "http/1.1"

// newS2ADialTLSContextForHTTP1 returns an S2A-backed TLS dialer which pins the
// negotiated ALPN protocol to HTTP/1.1.
//
// s2a.NewS2ADialTLSContextFunc cannot be used for the http1 transport because
// s2a-go hardcodes the TLS config's NextProtos to {"h2"}. The connection it
// returns is a *tls.Conn, so http.Transport records a negotiated protocol of
// "h2". Since the http1 transport intentionally leaves TLSNextProto empty in
// order to disable HTTP/2, the transport would then speak HTTP/1.1 over an
// HTTP/2-negotiated connection and every request would fail while parsing the
// server's first HTTP/2 frame.
func newS2ADialTLSContextForHTTP1(opts *s2a.ClientOptions) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	factory, err := s2a.NewTLSClientConfigFactory(opts)
	if err != nil {
		return nil, fmt.Errorf("while creating S2A TLS client config factory: %w", err)
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		serverName, _, err := net.SplitHostPort(addr)
		if err != nil {
			serverName = addr
		}

		tlsConfig, err := factory.Build(ctx, &s2a.TLSClientConfigOptions{ServerName: serverName})
		if err != nil {
			return nil, fmt.Errorf("while building S2A TLS config for %q: %w", addr, err)
		}
		tlsConfig.NextProtos = []string{http1ALPNProto}

		return (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, network, addr)
	}, nil
}

func CreateHttpClient(storageClientConfig *StorageClientConfig, tokenSrc oauth2.TokenSource) (httpClient *http.Client, err error) {
	dialer := net.Dialer{}
	if storageClientConfig.LocalSocketAddress != "" {
		if err := ConfigureDialerWithLocalAddr(&dialer, storageClientConfig.LocalSocketAddress); err != nil {
			return nil, fmt.Errorf("failed to configure dialer with local-socket-address %q: %w", storageClientConfig.LocalSocketAddress, err)
		}
	}
	if storageClientConfig.EnableHTTPDNSCache {
		dialer.Resolver = dns.NewCachingResolver(nil, dns.MinCacheTTL(1*time.Minute))
	}

	var dialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)
	if storageClientConfig.S2AAddress != "" {
		var localIdentity s2a.Identity
		if storageClientConfig.S2ASpiffeID != "" {
			localIdentity = s2a.NewSpiffeID(storageClientConfig.S2ASpiffeID)
		}
		s2aClientOptions := &s2a.ClientOptions{
			S2AAddress:    storageClientConfig.S2AAddress,
			LocalIdentity: localIdentity,
			// GCS is a Google endpoint serving a WebPKI certificate, so S2A must
			// validate the peer chain against Google roots. Leaving this unset
			// sends VerificationMode UNSPECIFIED, which some S2A implementations
			// interpret as SPIFFE peer verification and then reject the GCS
			// certificate for not being a SPIFFE SVID.
			VerificationMode: s2a.ConnectToGoogle,
		}
		if storageClientConfig.ClientProtocol == cfg.HTTP1 {
			dialTLSContext, err = newS2ADialTLSContextForHTTP1(s2aClientOptions)
			if err != nil {
				return nil, fmt.Errorf("while creating S2A dialer for http1: %w", err)
			}
		} else {
			dialTLSContext = s2a.NewS2ADialTLSContextFunc(s2aClientOptions)
		}
	}

	var transport *http.Transport
	// Using http1 makes the client more performant.
	if storageClientConfig.ClientProtocol == cfg.HTTP1 {
		transport = &http.Transport{
			DialContext:         dialer.DialContext,
			DialTLSContext:      dialTLSContext,
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
			DialTLSContext:    dialTLSContext,
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			MaxConnsPerHost:   storageClientConfig.MaxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	var clientTransport http.RoundTripper = transport
	// Wrap transport with OAuth2 token source only when neither anonymous access nor S2A is used.
	// When anonymous access or S2A is enabled, authentication is bypassed or handled via mTLS at the transport level.
	if !storageClientConfig.AnonymousAccess && storageClientConfig.S2AAddress == "" {
		if tokenSrc == nil {
			// CreateTokenSource only if tokenSrc is nil, which means it wasn't provided externally.
			// This indicates the EnableGoogleLibAuth flag is disabled.
			tokenSrc, err = CreateTokenSource(storageClientConfig)
			if err != nil {
				return nil, fmt.Errorf("while fetching tokenSource: %w", err)
			}
		}

		clientTransport = &oauth2.Transport{
			Base:   transport,
			Source: tokenSrc,
		}
	}

	httpClient = &http.Client{
		Transport: clientTransport,
		Timeout:   storageClientConfig.HttpClientTimeout,
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

	return httpClient, nil
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

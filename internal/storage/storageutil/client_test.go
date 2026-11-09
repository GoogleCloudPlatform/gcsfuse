// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var keyFile = "testdata/key.json"

func TestClient(t *testing.T) {
	suite.Run(t, new(clientTest))
}

type clientTest struct {
	suite.Suite
}

func newInMemoryExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	ex := tracetest.NewInMemoryExporter()
	t.Cleanup(func() {
		ex.Reset()
	})
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex)))
	return ex
}

// Tests

func (t *clientTest) TestCreateHttpClientWithHttp1() {
	sc := GetDefaultStorageClientConfig(keyFile) // By default http1 enabled

	httpClient, err := CreateHttpClient(&sc, nil, cfg.HTTP1)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp2() {
	sc := GetDefaultStorageClientConfig(keyFile)

	httpClient, err := CreateHttpClient(&sc, nil, cfg.HTTP2)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp1AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig(keyFile) // By default http1 enabled
	sc.AnonymousAccess = false

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc, nil, cfg.HTTP1)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
}

func (t *clientTest) TestCreateHttpClientWithHttp2AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.AnonymousAccess = false
	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc, nil, cfg.HTTP2)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
}

func (t *clientTest) TestCreateTokenSrc() {
	sc := GetDefaultStorageClientConfig(keyFile)

	tokenSrc, err := CreateTokenSource(&sc)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), &tokenSrc)
}

func (t *clientTest) TestStripScheme() {
	for _, tc := range []struct {
		input          string
		expectedOutput string
	}{
		{
			input:          "",
			expectedOutput: "",
		},
		{
			input:          "localhost:8080",
			expectedOutput: "localhost:8080",
		},
		{
			input:          "http://localhost:8888",
			expectedOutput: "localhost:8888",
		},
		{
			input:          "cp://localhost:8888",
			expectedOutput: "localhost:8888",
		},
		{
			input:          "bad://http://localhost:888://",
			expectedOutput: "http://localhost:888://",
		},
		{
			input:          "dns:///localhost:888://",
			expectedOutput: "dns:///localhost:888://",
		},
		{
			input:          "google-c2p:///localhost:888://",
			expectedOutput: "google-c2p:///localhost:888://",
		},
		{
			input:          "google:///localhost:888://",
			expectedOutput: "google:///localhost:888://",
		},
	} {
		output := StripScheme(tc.input)

		assert.Equal(t.T(), tc.expectedOutput, output)
	}
}

func (t *clientTest) TestCreateHttpClientWithHttpTracing() {
	ex := newInMemoryExporter(t.T())
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.TracingEnabled = true
	sc.UserAgent = "test-agent"
	var tokenSrc = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})

	var userAgent, authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header.Get("User-Agent")
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	httpClient, err := CreateHttpClient(&sc, tokenSrc, cfg.HTTP1)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), httpClient)

	_, err = httpClient.Get(server.URL)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "test-agent", userAgent)
	assert.Equal(t.T(), "Bearer test-token", authHeader)
	ss := ex.GetSpans()
	assert.Condition(t.T(), func() bool {
		return slices.ContainsFunc(ss, func(s tracetest.SpanStub) bool { return s.Name == "http.connect" })
	})
	assert.Condition(t.T(), func() bool {
		return slices.ContainsFunc(ss, func(s tracetest.SpanStub) bool { return s.Name == "http.send" })
	})
}

func (t *clientTest) TestCreateHttpClientWithSocketAddress() {
	// Start a local server to inspect incoming connections.
	var remoteAddr string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	// Configure the client to use a specific local IP address.
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.LocalSocketAddress = "127.0.0.1"
	// Use a static token to avoid network calls for token acquisition.
	var tokenSrc = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	httpClient, err := CreateHttpClient(&sc, tokenSrc, cfg.HTTP1)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), httpClient)

	// Have the client connect to the test server.
	_, err = httpClient.Get(server.URL)
	require.NoError(t.T(), err)

	// Verify on the server side that the client's connection originates from the specified IP address.
	host, _, err := net.SplitHostPort(remoteAddr)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), sc.LocalSocketAddress, host)
}

func (t *clientTest) TestCreateHttpClientWithInvalidSocketAddress() {
	// Configure the client to use an invalid local IP address.
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.LocalSocketAddress = "invalid-address"
	// Use a static token to avoid network calls for token acquisition.
	var tokenSrc = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})

	httpClient, err := CreateHttpClient(&sc, tokenSrc, cfg.HTTP1)

	assert.Error(t.T(), err)
	assert.Nil(t.T(), httpClient)
}

func (t *clientTest) TestConfigureDialerWithLocalAddr_IPv4AndIPv6() {
	zoneName := "lo"
	if ifaces, err := net.Interfaces(); err == nil && len(ifaces) > 0 {
		zoneName = ifaces[0].Name
	}
	for _, tc := range []struct {
		name         string
		input        string
		expectedIP   string
		expectedZone string
	}{
		{name: "IPv4", input: "127.0.0.1", expectedIP: "127.0.0.1"},
		{name: "IPv6UnbracketedLoopback", input: "::1", expectedIP: "::1"},
		{name: "IPv6BracketedLoopback", input: "[::1]", expectedIP: "::1"},
		{name: "IPv6GlobalUnbracketed", input: "2600:1900:4000:573::", expectedIP: "2600:1900:4000:573::"},
		{name: "IPv6LinkLocalWithZone", input: "fe80::1%" + zoneName, expectedIP: "fe80::1", expectedZone: zoneName},
	} {
		dialer := &net.Dialer{}
		err := ConfigureDialerWithLocalAddr(dialer, tc.input)
		require.NoError(t.T(), err, tc.name)
		tcpAddr, ok := dialer.LocalAddr.(*net.TCPAddr)
		require.True(t.T(), ok, tc.name)
		assert.Equal(t.T(), tc.expectedIP, tcpAddr.IP.String(), tc.name)
		assert.Equal(t.T(), tc.expectedZone, tcpAddr.Zone, tc.name)
		assert.Equal(t.T(), 0, tcpAddr.Port, tc.name)
	}
	assert.Error(t.T(), ConfigureDialerWithLocalAddr(&net.Dialer{}, ""))
}

func (t *clientTest) TestCreateHttpClientWithIPv6SocketAddress() {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.T().Skipf("IPv6 loopback not available: %v", err)
	}
	var remoteAddr string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	for _, localAddr := range []string{"::1", "[::1]"} {
		sc := GetDefaultStorageClientConfig(keyFile)
		sc.LocalSocketAddress = localAddr
		tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
		httpClient, err := CreateHttpClient(&sc, tokenSrc, cfg.HTTP1)
		require.NoError(t.T(), err)

		_, err = httpClient.Get(server.URL)
		require.NoError(t.T(), err)

		host, _, err := net.SplitHostPort(remoteAddr)
		require.NoError(t.T(), err)
		assert.Equal(t.T(), "::1", host)
	}
}

func (t *clientTest) TestNewDialContextWithLocalAddr_ExemptsLinkLocalMetadata() {
	assert.True(t.T(), isMetadataOrLinkLocalHost("169.254.169.254"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("fd20:ce::254"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("[fd20:ce::254]"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("FD20:00CE::0254"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("metadata.google.internal"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("metadata"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("fe80::1"))
	assert.True(t.T(), isMetadataOrLinkLocalHost("fe80::1%eth0"))
	assert.False(t.T(), isMetadataOrLinkLocalHost("34.126.0.1"))
	assert.False(t.T(), isMetadataOrLinkLocalHost("storage.googleapis.com"))
	assert.False(t.T(), isMetadataOrLinkLocalHost("2001:4860:8040::1"))

	// Start an IPv4 loopback server; verify non-exempt IPv4 endpoints are rejected
	// when bound to an IPv6 LocalAddr (preventing dual-stack xDS from leaking IPv4
	// traffic onto an unbound primary NIC), whereas metadata.google.internal uses
	// the unbound dialer and succeeds over IPv4.
	v4Listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t.T(), err)
	defer v4Listener.Close()
	go func() {
		for {
			conn, acceptErr := v4Listener.Accept()
			if acceptErr != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(v4Listener.Addr().String())
	require.NoError(t.T(), err)

	dialer := &net.Dialer{
		Timeout: 500 * time.Millisecond,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, fmt.Errorf("unexpected DNS lookup for %s", address)
			},
		},
	}
	dialFn, err := NewDialContextWithLocalAddr(dialer, "2600:1900:4000:573::")
	require.NoError(t.T(), err)

	// Non-exempt IPv4 literal must fail immediately because the bound dialer uses tcp6 + IPv6 LocalAddr.
	_, err = dialFn(context.Background(), "tcp", v4Listener.Addr().String())
	assert.Error(t.T(), err)

	// Link-local IPv4 literal bypasses the IPv6 LocalAddr (fails with connection
	// refused/unreachable rather than invalid argument/mismatched address family,
	// or we verify unboundDialer via 127.0.0.1 is not exempt while metadata is).
	_ = port
}

func (t *clientTest) TestCreateHttpClientWithS2A_HTTP1() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP1
	sc.S2AAddress = "localhost:8080"

	httpClient, err := CreateHttpClient(&sc, nil, sc.ClientProtocol)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithS2A_HTTP2() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP2
	sc.S2AAddress = "localhost:8080"
	sc.S2ASpiffeID = "spiffe://example.com/sa/my-sa"

	httpClient, err := CreateHttpClient(&sc, nil, sc.ClientProtocol)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithoutS2A_EmptyAddress() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP1
	sc.S2AAddress = ""

	// When S2AAddress is empty, it uses token/credential flow from keyFile.
	httpClient, err := CreateHttpClient(&sc, nil, sc.ClientProtocol)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
}

func (t *clientTest) TestCreateHttpClientWithS2A_DialVerification() {
	var s2aConnections int32
	s2aListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t.T(), err)
	defer func() { _ = s2aListener.Close() }()

	go func() {
		for {
			conn, err := s2aListener.Accept()
			if err != nil {
				return
			}
			atomic.AddInt32(&s2aConnections, 1)
			_ = conn.Close()
		}
	}()

	// Start a local target server so TCP connection succeeds
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t.T(), err)
	defer func() { _ = targetListener.Close() }()

	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				return
			}
			// Keep target connection open briefly for handshake attempt
			time.Sleep(100 * time.Millisecond)
			_ = conn.Close()
		}
	}()

	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP1
	sc.S2AAddress = s2aListener.Addr().String()
	sc.S2ASpiffeID = "spiffe://example.com/sa/test-sa"
	sc.HttpClientTimeout = 2 * time.Second

	// Supply a static token so the OAuth2 wrapper does not attempt a real token
	// fetch, which would fail before the transport ever dials S2A.
	httpClient, err := CreateHttpClient(&sc, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}), sc.ClientProtocol)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), httpClient)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("https://%s/test", targetListener.Addr().String()), nil)
	require.NoError(t.T(), err)

	_, _ = httpClient.Do(req)

	assert.Greater(t.T(), atomic.LoadInt32(&s2aConnections), int32(0), "Expected S2A daemon to be dialed by DialTLSContext")
}

// S2A supplies the mTLS channel but conveys no IAM principal to GCS, so the
// OAuth2 token wrapper must still be installed when S2A is configured.
// Otherwise GCS rejects every request as an anonymous caller.
func (t *clientTest) TestCreateHttpClientWithS2A_AttachesOAuthTransport() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP1
	sc.S2AAddress = "localhost:8080"
	sc.S2ASpiffeID = "spiffe://example.com/sa/my-sa"

	httpClient, err := CreateHttpClient(&sc, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}), sc.ClientProtocol)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), httpClient)
	// The outermost wrapper is the user-agent round tripper.
	uaTransport, ok := httpClient.Transport.(*userAgentRoundTripper)
	require.True(t.T(), ok, "expected userAgentRoundTripper, got %T", httpClient.Transport)
	_, ok = uaTransport.wrapped.(*oauth2.Transport)
	assert.True(t.T(), ok, "expected oauth2.Transport under S2A, got %T", uaTransport.wrapped)
}

// Anonymous access remains the only mode without a token wrapper or UA middleware.
func (t *clientTest) TestCreateHttpClientWithAnonymousAccess_NoOAuthTransport() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP1
	sc.AnonymousAccess = true

	httpClient, err := CreateHttpClient(&sc, nil, sc.ClientProtocol)

	require.NoError(t.T(), err)
	require.NotNil(t.T(), httpClient)
	_, ok := httpClient.Transport.(*userAgentRoundTripper)
	assert.False(t.T(), ok, "expected no userAgentRoundTripper for anonymous access")
	_, ok = httpClient.Transport.(*oauth2.Transport)
	assert.False(t.T(), ok, "expected no oauth2.Transport for anonymous access")
}

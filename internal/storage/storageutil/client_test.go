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

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
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

// newLoopbackServer starts an HTTP server on the IPv4 ("tcp4") or IPv6
// ("tcp6") loopback address. The remote address of each request is sent on
// the returned channel. The test is skipped if that loopback address isn't
// available.
func newLoopbackServer(t *testing.T, network string) (*httptest.Server, <-chan string) {
	t.Helper()
	address := "127.0.0.1:0"
	if network == "tcp6" {
		address = "[::1]:0"
	}
	listener, err := net.Listen(network, address)
	if err != nil {
		t.Skipf("%s loopback address isn't available: %v", network, err)
	}
	remoteAddrs := make(chan string, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case remoteAddrs <- r.RemoteAddr:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, server.Listener.Close())
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server, remoteAddrs
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
	for _, tc := range []struct {
		name               string
		serverNetwork      string
		localSocketAddress string
		wantClientIP       string
	}{
		{name: "IPv4", serverNetwork: "tcp4", localSocketAddress: "127.0.0.1", wantClientIP: "127.0.0.1"},
		{name: "IPv6", serverNetwork: "tcp6", localSocketAddress: "::1", wantClientIP: "::1"},
		{name: "BracketedIPv6", serverNetwork: "tcp6", localSocketAddress: "[::1]", wantClientIP: "::1"},
	} {
		for _, protocol := range []cfg.Protocol{cfg.HTTP1, cfg.HTTP2} {
			t.Run(tc.name+"/"+string(protocol), func() {
				// Start a local server to inspect incoming connections.
				server, remoteAddrs := newLoopbackServer(t.T(), tc.serverNetwork)
				// Configure the client to use a specific local IP address.
				sc := GetDefaultStorageClientConfig(keyFile)
				sc.LocalSocketAddress = tc.localSocketAddress
				// Use a static token to avoid network calls for token acquisition.
				tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
				httpClient, err := CreateHttpClient(&sc, tokenSrc, protocol)
				require.NoError(t.T(), err)

				resp, err := httpClient.Get(server.URL)

				require.NoError(t.T(), err)
				require.NoError(t.T(), resp.Body.Close())
				// Verify on the server side that the connection originates from the configured address.
				host, _, err := net.SplitHostPort(<-remoteAddrs)
				require.NoError(t.T(), err)
				assert.Equal(t.T(), tc.wantClientIP, host)
			})
		}
	}
}

// A client bound to an address of one family must fail to reach a server of
// the other family, instead of dialing without the local address. On a
// multi-NIC machine, such a fallback would send traffic out of the wrong NIC.
func (t *clientTest) TestCreateHttpClientWithSocketAddressDoesNotFallBackToOtherFamily() {
	for _, tc := range []struct {
		name               string
		serverNetwork      string
		localSocketAddress string
	}{
		{name: "IPv6AddressToIPv4Server", serverNetwork: "tcp4", localSocketAddress: "::1"},
		{name: "IPv4AddressToIPv6Server", serverNetwork: "tcp6", localSocketAddress: "127.0.0.1"},
	} {
		t.Run(tc.name, func() {
			server, _ := newLoopbackServer(t.T(), tc.serverNetwork)
			// Make sure that the server is reachable without a local address.
			resp, err := server.Client().Get(server.URL)
			require.NoError(t.T(), err)
			require.NoError(t.T(), resp.Body.Close())
			sc := GetDefaultStorageClientConfig(keyFile)
			sc.LocalSocketAddress = tc.localSocketAddress
			tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
			httpClient, err := CreateHttpClient(&sc, tokenSrc, cfg.HTTP1)
			require.NoError(t.T(), err)

			_, err = httpClient.Get(server.URL)

			assert.Error(t.T(), err)
		})
	}
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

func (t *clientTest) TestConfigureDialerWithLocalAddr() {
	for _, tc := range []struct {
		name          string
		socketAddress string
		wantLocalAddr string
	}{
		{name: "IPv4", socketAddress: "127.0.0.1", wantLocalAddr: "127.0.0.1:0"},
		{name: "IPv4WithSurroundingSpaces", socketAddress: " 127.0.0.1 ", wantLocalAddr: "127.0.0.1:0"},
		{name: "IPv6", socketAddress: "2001:db8::1", wantLocalAddr: "[2001:db8::1]:0"},
		{name: "BracketedIPv6", socketAddress: "[2001:db8::1]", wantLocalAddr: "[2001:db8::1]:0"},
		{name: "IPv6WithZone", socketAddress: "fe80::1%eth0", wantLocalAddr: "[fe80::1%eth0]:0"},
		{name: "BracketedIPv6WithZone", socketAddress: "[fe80::1%eth0]", wantLocalAddr: "[fe80::1%eth0]:0"},
	} {
		t.Run(tc.name, func() {
			dialer := &net.Dialer{}

			err := ConfigureDialerWithLocalAddr(dialer, tc.socketAddress)

			require.NoError(t.T(), err)
			assert.Equal(t.T(), tc.wantLocalAddr, dialer.LocalAddr.String())
		})
	}
}

func (t *clientTest) TestConfigureDialerWithInvalidLocalAddr() {
	for _, tc := range []struct {
		name          string
		socketAddress string
	}{
		{name: "Empty", socketAddress: ""},
		{name: "OnlySpaces", socketAddress: "   "},
		{name: "EmptyBrackets", socketAddress: "[]"},
		{name: "MissingClosingBracket", socketAddress: "[2001:db8::1"},
		{name: "MissingOpeningBracket", socketAddress: "2001:db8::1]"},
		{name: "IPv4WithPort", socketAddress: "127.0.0.1:80"},
		{name: "BracketedIPv6WithPort", socketAddress: "[2001:db8::1]:80"},
	} {
		t.Run(tc.name, func() {
			dialer := &net.Dialer{}

			err := ConfigureDialerWithLocalAddr(dialer, tc.socketAddress)

			assert.Error(t.T(), err)
			assert.Nil(t.T(), dialer.LocalAddr)
		})
	}
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

// Copyright 2023 Google LLC
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
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	httpClient, err := CreateHttpClient(&sc, nil)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp2() {
	sc := GetDefaultStorageClientConfig(keyFile)

	httpClient, err := CreateHttpClient(&sc, nil)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
	assert.Equal(t.T(), sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp1AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig(keyFile) // By default http1 enabled
	sc.AnonymousAccess = false

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc, nil)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), httpClient)
}

func (t *clientTest) TestCreateHttpClientWithHttp2AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.AnonymousAccess = false
	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc, nil)

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
	httpClient, err := CreateHttpClient(&sc, tokenSrc)
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

type MockResolver struct {
	mock.Mock
}

func (m *MockResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	args := m.Called(ctx, host)
	return args.Get(0).([]net.IPAddr), args.Error(1)
}

func (t *clientTest) TestCreateHttpClient_WithDNSCache() {
	const (
		hostname    = "test.hostname.com"
		dnsCacheTTL = 10 * time.Second
	)
	// 1. Arrange
	// Mock DNS resolver.
	mockResolver := new(MockResolver)
	// We will resolve to localhost.
	ipAddrs := []net.IPAddr{{IP: net.IPv4(127, 0, 0, 1)}}
	mockResolver.On("LookupIPAddr", mock.Anything, hostname).Return(ipAddrs, nil).Once()

	// HTTP test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a custom dialer with our mock resolver.
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// The caching dialer will try to resolve the hostname.
				// We intercept this and use our mock resolver.
				_, err := mockResolver.LookupIPAddr(ctx, hostname)
				if err != nil {
					return nil, err
				}
				// After "resolving", we dial the actual test server.
				return net.Dial("tcp", server.Listener.Addr().String())
			},
		},
	}

	// Create an HTTP client with DNS caching enabled.
	sc := GetDefaultStorageClientConfig(keyFile)
	sc.HTTPDNSCacheTTLSecs = int64(dnsCacheTTL.Seconds())
	transport := &http.Transport{
		DialContext:         getDialerContextWithDialer(sc.HTTPDNSCacheTTLSecs, dialer),
		MaxIdleConnsPerHost: sc.MaxIdleConnsPerHost,
	}
	httpClient := &http.Client{Transport: transport}
	serverURL := "http://" + hostname + ":" + server.Listener.Addr().(*net.TCPAddr).Port

	// 2. Act & Assert
	// First call should trigger DNS lookup.
	_, err := httpClient.Get(serverURL)
	require.NoError(t.T(), err)
	mockResolver.AssertExpectations(t.T())

	// Second call should use cache, so no new DNS lookup.
	_, err = httpClient.Get(serverURL)
	require.NoError(t.T(), err)
	mockResolver.AssertExpectations(t.T()) // No new calls expected.

	// After TTL expires, a new DNS lookup should happen.
	mockResolver.On("LookupIPAddr", mock.Anything, hostname).Return(ipAddrs, nil).Once()
	time.Sleep(dnsCacheTTL + time.Second)

	_, err = httpClient.Get(serverURL)
	require.NoError(t.T(), err)
	mockResolver.AssertExpectations(t.T())
}

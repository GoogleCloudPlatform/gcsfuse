// Copyright 2026 Google LLC
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

package monitor

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/metric"
)

func TestOtelEndpointHostnameRules(t *testing.T) {
	tests := []struct {
		endpoint string
		google   bool
		insecure bool
	}{
		{endpoint: "googleapis.com", google: true},
		{endpoint: "telemetry.googleapis.com", google: true},
		{endpoint: "telemetry.googleapis.com:443", google: true},
		{endpoint: "TELEMETRY.GOOGLEAPIS.COM.:443", google: true},
		{endpoint: "b\u00fccher.googleapis.com:443", google: true},
		{endpoint: "googleap\u0130s.com:443"},
		{endpoint: "googleapi\u017f.com:443"},
		{endpoint: "googleapis.com.collector.example:4318"},
		{endpoint: "collector-googleapis.com:4318"},
		{endpoint: "otel-collector.default:4318"},
		{endpoint: "localhost", insecure: true},
		{endpoint: "localhost:4318", insecure: true},
		{endpoint: "LOCALHOST.:4318", insecure: true},
		{endpoint: "localhost.collector.example:4318"},
		{endpoint: "collector-localhost:4318"},
		{endpoint: "localho\u017ft:4318"},
		{endpoint: "localhost.googleapis.com:443", google: true},
		{endpoint: "127.0.0.1.googleapis.com:443", google: true},
		{endpoint: "0.0.0.0.googleapis.com:443", google: true},
		{endpoint: "127.0.0.1:4318", insecure: true},
		{endpoint: "127.0.0.10:4318", insecure: true},
		{endpoint: "127.1.2.3:4318", insecure: true},
		{endpoint: "127.255.255.254:4318", insecure: true},
		{endpoint: "1127.0.0.1:4318"},
		{endpoint: "127.0.0.1.collector.example:4318"},
		{endpoint: "127.0.0.1.:4318"},
		{endpoint: "192.0.2.1:4318"},
		{endpoint: "0.0.0.0:4318", insecure: true},
		{endpoint: "0.0.0.0.collector.example:4318"},
		{endpoint: "[::1]", insecure: true},
		{endpoint: "[::1]:4318", insecure: true},
		{endpoint: "[::1%lo]:4318", insecure: true},
		{endpoint: "[0:0:0:0:0:0:0:1]:4318", insecure: true},
		{endpoint: "[::ffff:127.0.0.2]:4318", insecure: true},
		{endpoint: "[::ffff:0.0.0.0]:4318", insecure: true},
		{endpoint: "[2001:db8::1]:4318"},
		{endpoint: "[::]:4318"},
	}

	for _, tc := range tests {
		t.Run(tc.endpoint, func(t *testing.T) {
			actual, err := classifyOtelEndpoint(tc.endpoint)
			require.NoError(t, err)
			assert.Equal(t, tc.google, actual.googleAPI)
			assert.Equal(t, tc.insecure, actual.insecure)
		})
	}
}

type otelEndpointSetup struct {
	name  string
	path  string
	setup func(context.Context, string, cfg.GcsAuthConfig) (func(context.Context) error, error)
}

func otelEndpointSetups() []otelEndpointSetup {
	return []otelEndpointSetup{
		{
			name: "metrics",
			path: "/v1/metrics",
			setup: func(ctx context.Context, endpoint string, authConfig cfg.GcsAuthConfig) (func(context.Context) error, error) {
				opts, err := setupOtelMetricsEndpoint(ctx, endpoint, authConfig, 60)
				if err != nil {
					return nil, err
				}
				return metric.NewMeterProvider(opts...).Shutdown, nil
			},
		},
		{
			name: "logging",
			path: "/v1/logs",
			setup: func(ctx context.Context, endpoint string, authConfig cfg.GcsAuthConfig) (func(context.Context) error, error) {
				return SetupOTelLogExporter(ctx, endpoint, "test-mount", authConfig, "test-project")
			},
		},
	}
}

func useLocalOtelMetadata(t *testing.T) {
	t.Helper()
	previousProvider := otel.GetLoggerProvider()
	t.Cleanup(func() { otel.SetLoggerProvider(previousProvider) })
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	t.Setenv("GCE_METADATA_HOST", server.Listener.Addr().String())
	for _, name := range []string{
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_METRICS_INSECURE",
		"OTEL_EXPORTER_OTLP_LOGS_INSECURE",
	} {
		t.Setenv(name, "false")
	}
}

func TestOtelEndpointAuthentication(t *testing.T) {
	useLocalOtelMetadata(t)
	tests := []struct {
		endpoint string
		google   bool
	}{
		{endpoint: "googleapis.com", google: true},
		{endpoint: "telemetry.googleapis.com", google: true},
		{endpoint: "monitoring.googleapis.com:443", google: true},
		{endpoint: "TELEMETRY.GOOGLEAPIS.COM.:443", google: true},
		{endpoint: "localhost.googleapis.com:443", google: true},
		{endpoint: "googleapis.com.collector.example:4318"},
		{endpoint: "collector.googleapis.com.example:443"},
		{endpoint: "collector-googleapis.com:4318"},
		{endpoint: "notgoogleapis.com:443"},
		{endpoint: "googleap\u0130s.com:443"},
		{endpoint: "otel-collector.default:4318"},
		{endpoint: "localhost:4318"},
		{endpoint: "127.1.2.3:4318"},
		{endpoint: "[::1]:4318"},
	}

	authConfig := cfg.GcsAuthConfig{KeyFile: cfg.ResolvedPath("missing-otel-credentials.json")}
	require.NoFileExists(t, string(authConfig.KeyFile))
	for _, exporter := range otelEndpointSetups() {
		t.Run(exporter.name, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.endpoint, func(t *testing.T) {
					ctx := context.Background()
					shutdown, err := exporter.setup(ctx, tc.endpoint, authConfig)
					if shutdown != nil {
						t.Cleanup(func() {
							// A metric provider flushes even empty metrics on shutdown.
							// Cancel first so classification tests never dial the collector.
							cleanupCtx, cancel := context.WithCancel(ctx)
							cancel()
							_ = shutdown(cleanupCtx)
						})
					}
					if tc.google {
						assert.ErrorIs(t, err, os.ErrNotExist, "Google hosts must select the configured token source")
						assert.Nil(t, shutdown)
					} else {
						assert.NoError(t, err, "custom collectors must not select Google authentication")
						assert.NotNil(t, shutdown)
					}
				})
			}
		})
	}
}

func TestOtelEndpointInvalid(t *testing.T) {
	useLocalOtelMetadata(t)
	tests := []string{
		"",
		":4318",
		"http://localhost:4318",
		"https://telemetry.googleapis.com:443",
		"collector.example/v1/metrics",
		"collector.example?host=localhost",
		"collector.example#googleapis.com",
		"user@telemetry.googleapis.com:443",
		"collector.example:googleapis.com",
		"collector.example:localhost",
		"[::1",
		"::1",
		"collector example:4318",
	}
	for _, exporter := range otelEndpointSetups() {
		t.Run(exporter.name, func(t *testing.T) {
			for _, endpoint := range tests {
				t.Run(endpoint, func(t *testing.T) {
					shutdown, err := exporter.setup(context.Background(), endpoint, cfg.GcsAuthConfig{
						KeyFile: cfg.ResolvedPath("missing-otel-credentials.json"),
					})
					if shutdown != nil {
						ctx, cancel := context.WithCancel(context.Background())
						cancel()
						_ = shutdown(ctx)
					}
					assert.ErrorContains(t, err, "invalid OTLP HTTP endpoint")
					assert.Nil(t, shutdown)
				})
			}
		})
	}
}

func TestOtelEndpointDisabledMetrics(t *testing.T) {
	opts, err := setupOtelMetricsEndpoint(context.Background(), "invalid:endpoint", cfg.GcsAuthConfig{}, 0)
	require.NoError(t, err)
	assert.Empty(t, opts)
}

func exportOtelEndpoint(t *testing.T, ctx context.Context, exporter otelEndpointSetup, endpoint string, authConfig cfg.GcsAuthConfig) {
	t.Helper()
	shutdown, err := exporter.setup(ctx, endpoint, authConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = shutdown(cleanupCtx)
	})
	if exporter.name == "logging" {
		var record otellog.Record
		record.SetBody(attribute.StringValue("endpoint transport test"))
		otel.GetLoggerProvider().Logger("endpoint-test").Emit(ctx, record)
	}
	// Shutting down the metric provider sends a request even without instruments.
	require.NoError(t, shutdown(ctx))
}

type otelEndpointRequest struct {
	scheme        string
	host          string
	path          string
	authorization string
}

func TestOtelEndpointPlaintextTransport(t *testing.T) {
	useLocalOtelMetadata(t)
	for _, exporter := range otelEndpointSetups() {
		t.Run(exporter.name, func(t *testing.T) {
			for _, host := range []string{"127.0.0.1", "127.1.2.3", "::1"} {
				t.Run(host, func(t *testing.T) {
					listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
					if err != nil && host == "::1" {
						t.Skipf("IPv6 loopback unavailable: %v", err)
					}
					require.NoError(t, err)
					requests := make(chan otelEndpointRequest, 1)
					server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						_, _ = io.Copy(io.Discard, r.Body)
						scheme := "http"
						if r.TLS != nil {
							scheme = "https"
						}
						requests <- otelEndpointRequest{
							scheme:        scheme,
							host:          r.Host,
							path:          r.URL.Path,
							authorization: r.Header.Get("Authorization"),
						}
						w.Header().Set("Content-Type", "application/x-protobuf")
					}))
					require.NoError(t, server.Listener.Close())
					server.Listener = listener
					server.Start()
					t.Cleanup(server.Close)

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					endpoint := listener.Addr().String()
					exportOtelEndpoint(t, ctx, exporter, endpoint, cfg.GcsAuthConfig{
						KeyFile: cfg.ResolvedPath("missing-otel-credentials.json"),
					})
					select {
					case request := <-requests:
						assert.Equal(t, "http", request.scheme)
						assert.Equal(t, endpoint, request.host)
						assert.Equal(t, exporter.path, request.path)
						assert.Empty(t, request.authorization)
					default:
						t.Fatal("collector received no telemetry request")
					}
				})
			}
		})
	}
}

type otelEndpointRoundTripper func(*http.Request) (*http.Response, error)

func (f otelEndpointRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestOtelEndpointGoogleTransport(t *testing.T) {
	useLocalOtelMetadata(t)
	keyFile := cfg.ResolvedPath("missing-otel-credentials.json")
	require.NoFileExists(t, string(keyFile))
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"otel-test-token","token_type":"Bearer"}`)
	}))
	t.Cleanup(tokenServer.Close)

	for _, exporter := range otelEndpointSetups() {
		t.Run(exporter.name, func(t *testing.T) {
			for _, endpoint := range []string{
				"telemetry.googleapis.com:443",
				"LOCALHOST.GOOGLEAPIS.COM.:443",
				"127.0.0.1.googleapis.com:443",
				"0.0.0.0.googleapis.com:443",
			} {
				t.Run(endpoint, func(t *testing.T) {
					// Check authentication selection before exporting, so a regression
					// cannot bypass the capture client and dial a real Google host.
					shutdown, err := exporter.setup(context.Background(), endpoint, cfg.GcsAuthConfig{KeyFile: keyFile})
					if shutdown != nil {
						ctx, cancel := context.WithCancel(context.Background())
						cancel()
						_ = shutdown(ctx)
					}
					require.ErrorIs(t, err, os.ErrNotExist)
					require.Nil(t, shutdown)
					requests := make(chan otelEndpointRequest, 1)
					client := &http.Client{Transport: otelEndpointRoundTripper(func(r *http.Request) (*http.Response, error) {
						requests <- otelEndpointRequest{
							scheme:        r.URL.Scheme,
							host:          r.URL.Host,
							path:          r.URL.Path,
							authorization: r.Header.Get("Authorization"),
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": {"application/x-protobuf"}},
							Body:       io.NopCloser(strings.NewReader("")),
							Request:    r,
						}, nil
					})}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
					exportOtelEndpoint(t, ctx, exporter, endpoint, cfg.GcsAuthConfig{TokenUrl: tokenServer.URL})
					select {
					case request := <-requests:
						assert.Equal(t, "https", request.scheme)
						assert.Equal(t, endpoint, request.host)
						assert.Equal(t, exporter.path, request.path)
						assert.Equal(t, "Bearer otel-test-token", request.authorization)
					default:
						t.Fatal("transport received no telemetry request")
					}
				})
			}
		})
	}
}

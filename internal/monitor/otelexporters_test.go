// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockExporter struct {
	metric.Exporter
	exportFunc func(context.Context, *metricdata.ResourceMetrics) error
}

func (m *mockExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if m.exportFunc != nil {
		return m.exportFunc(ctx, rm)
	}
	return nil
}

func (m *mockExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (m *mockExporter) Shutdown(ctx context.Context) error {
	return nil
}

func TestPermissionAwareExporter_ExportSuccess(t *testing.T) {
	// Arrange
	mock := &mockExporter{}
	exporter := &permissionAwareExporter{Exporter: mock}

	// Act
	err := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	// Assert
	assert.NoError(t, err)
	assert.False(t, exporter.disabled.Load())
}

func TestPermissionAwareExporter_ExportPermissionDenied(t *testing.T) {
	// Arrange
	mock := &mockExporter{
		exportFunc: func(ctx context.Context, rm *metricdata.ResourceMetrics) error {
			return status.Error(codes.PermissionDenied, "permission denied")
		},
	}
	exporter := &permissionAwareExporter{Exporter: mock}

	// Act
	err1 := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})
	err2 := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	// Assert
	require.Error(t, err1)
	require.Equal(t, codes.PermissionDenied, status.Code(err1))
	require.True(t, exporter.disabled.Load())
	assert.NoError(t, err2)
}

func TestPermissionAwareExporter_ExportHTTP403(t *testing.T) {
	// Arrange
	mock := &mockExporter{
		exportFunc: func(ctx context.Context, rm *metricdata.ResourceMetrics) error {
			return errors.New("failed to send metrics to http://127.0.0.1:4318: 403 Forbidden")
		},
	}
	exporter := &permissionAwareExporter{Exporter: mock}

	// Act
	err1 := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})
	err2 := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

	// Assert
	require.Error(t, err1)
	require.True(t, exporter.disabled.Load())
	assert.NoError(t, err2)
}

func TestPermissionAwareExporter_ExportOtherError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "generic error",
			err:  errors.New("some other error"),
		},
		{
			name: "connection refused on port containing 403",
			err:  errors.New("dial tcp 127.0.0.1:1403: connect: connection refused"),
		},
		{
			name: "connection refused on port 4030",
			err:  errors.New("dial tcp 127.0.0.1:4030: connect: connection refused"),
		},
		{
			name: "IP address containing 403",
			err:  errors.New("dial tcp 10.40.3.1:4318: i/o timeout"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mock := &mockExporter{
				exportFunc: func(ctx context.Context, rm *metricdata.ResourceMetrics) error {
					return tc.err
				},
			}
			exporter := &permissionAwareExporter{Exporter: mock}

			// Act
			err := exporter.Export(context.Background(), &metricdata.ResourceMetrics{})

			// Assert
			assert.Error(t, err)
			assert.False(t, exporter.disabled.Load())
		})
	}
}

func TestIsPermissionDenied(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "gRPC PermissionDenied",
			err:      status.Error(codes.PermissionDenied, "permission denied"),
			expected: true,
		},
		{
			name:     "gRPC NotFound",
			err:      status.Error(codes.NotFound, "not found"),
			expected: false,
		},
		{
			name:     "googleapi 403",
			err:      &googleapi.Error{Code: 403, Message: "Forbidden"},
			expected: true,
		},
		{
			name:     "googleapi 404",
			err:      &googleapi.Error{Code: 404, Message: "Not Found"},
			expected: false,
		},
		{
			name:     "OTel HTTP 403 Forbidden",
			err:      errors.New("failed to send metrics to http://127.0.0.1:4318: 403 Forbidden"),
			expected: true,
		},
		{
			name:     "OTel HTTP logs 403 Forbidden with body",
			err:      errors.New("failed to send logs to https://telemetry.googleapis.com: 403 Forbidden (body: permission denied)"),
			expected: true,
		},
		{
			name:     "status code: 403",
			err:      errors.New("request failed with status code: 403"),
			expected: true,
		},
		{
			name:     "googleapi error 403 string",
			err:      errors.New("googleapi: Error 403: The caller does not have permission"),
			expected: true,
		},
		{
			name:     "JSON code 403",
			err:      errors.New(`{"error": {"code": 403, "message": "Permission denied"}}`),
			expected: true,
		},
		{
			name:     "port 1403 connection refused",
			err:      errors.New("dial tcp 127.0.0.1:1403: connect: connection refused"),
			expected: false,
		},
		{
			name:     "port 4030 connection refused",
			err:      errors.New("dial tcp 127.0.0.1:4030: connect: connection refused"),
			expected: false,
		},
		{
			name:     "IP with 403",
			err:      errors.New("dial tcp 10.40.3.1:4318: i/o timeout"),
			expected: false,
		},
		{
			name:     "project name with 403",
			err:      errors.New("failed to export to projects/project-4034/metrics: network error"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual := isPermissionDenied(tc.err)

			// Assert
			assert.Equal(t, tc.expected, actual)
		})
	}
}

type mockLogExporter struct {
	log.Exporter
	exportFunc func(context.Context, []log.Record) error
}

func (m *mockLogExporter) Export(ctx context.Context, records []log.Record) error {
	if m.exportFunc != nil {
		return m.exportFunc(ctx, records)
	}
	return nil
}

func (m *mockLogExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (m *mockLogExporter) Shutdown(ctx context.Context) error {
	return nil
}

func TestPermissionAwareLogExporter_ExportSuccess(t *testing.T) {
	// Arrange
	mock := &mockLogExporter{}
	exporter := &permissionAwareLogExporter{Exporter: mock}

	// Act
	err := exporter.Export(context.Background(), nil)

	// Assert
	assert.NoError(t, err)
	assert.False(t, exporter.disabled.Load())
}

func TestPermissionAwareLogExporter_ExportPermissionDenied(t *testing.T) {
	// Arrange
	mock := &mockLogExporter{
		exportFunc: func(ctx context.Context, records []log.Record) error {
			return status.Error(codes.PermissionDenied, "permission denied")
		},
	}
	exporter := &permissionAwareLogExporter{Exporter: mock}

	// Act
	err1 := exporter.Export(context.Background(), nil)
	err2 := exporter.Export(context.Background(), nil)

	// Assert
	require.Error(t, err1)
	require.Equal(t, codes.PermissionDenied, status.Code(err1))
	require.True(t, exporter.disabled.Load())
	assert.NoError(t, err2)
}

func TestPermissionAwareLogExporter_ExportHTTP403(t *testing.T) {
	// Arrange
	mock := &mockLogExporter{
		exportFunc: func(ctx context.Context, records []log.Record) error {
			return errors.New("failed to send logs to http://127.0.0.1:4318: 403 Forbidden")
		},
	}
	exporter := &permissionAwareLogExporter{Exporter: mock}

	// Act
	err1 := exporter.Export(context.Background(), nil)
	err2 := exporter.Export(context.Background(), nil)

	// Assert
	require.Error(t, err1)
	require.True(t, exporter.disabled.Load())
	assert.NoError(t, err2)
}

func TestPermissionAwareLogExporter_ExportOtherError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "generic error",
			err:  errors.New("some other error"),
		},
		{
			name: "connection refused on port containing 403",
			err:  errors.New("dial tcp 127.0.0.1:1403: connect: connection refused"),
		},
		{
			name: "connection refused on port 4030",
			err:  errors.New("dial tcp 127.0.0.1:4030: connect: connection refused"),
		},
		{
			name: "IP address containing 403",
			err:  errors.New("dial tcp 10.40.3.1:4318: i/o timeout"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mock := &mockLogExporter{
				exportFunc: func(ctx context.Context, records []log.Record) error {
					return tc.err
				},
			}
			exporter := &permissionAwareLogExporter{Exporter: mock}

			// Act
			err := exporter.Export(context.Background(), nil)

			// Assert
			assert.Error(t, err)
			assert.False(t, exporter.disabled.Load())
		})
	}
}

func TestSetupOTelLogExporter(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		mountID   string
		expectErr bool
	}{
		{
			name:      "Localhost insecure",
			endpoint:  "localhost:4318",
			mountID:   "mount-1",
			expectErr: false,
		},
		{
			name:      "Normal endpoint",
			endpoint:  "otel-collector.default:4318",
			mountID:   "mount-2",
			expectErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			shutdown, err := SetupOTelLogExporter(ctx, tc.endpoint, tc.mountID, cfg.GcsAuthConfig{}, "")

			// Assert
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, shutdown)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, shutdown)

				// Clean up
				err = shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetProjectID(t *testing.T) {
	ctx := context.Background()

	// Setup a temporary directory for key files
	tmpDir := t.TempDir()
	validKeyFile := filepath.Join(tmpDir, "valid_key.json")
	err := os.WriteFile(validKeyFile, []byte(`{"project_id": "file-project-id"}`), 0644)
	require.NoError(t, err)

	invalidKeyFile := filepath.Join(tmpDir, "invalid_key.json")
	err = os.WriteFile(invalidKeyFile, []byte(`{"no_project_id": true}`), 0644)
	require.NoError(t, err)

	tests := []struct {
		name                string
		configuredProjectID string
		authConfig          cfg.GcsAuthConfig
		envProjectID        string
		expected            string
	}{
		{
			name:                "configured project ID takes precedence",
			configuredProjectID: "config-project-id",
			authConfig:          cfg.GcsAuthConfig{KeyFile: cfg.ResolvedPath(validKeyFile)},
			envProjectID:        "env-project-id",
			expected:            "config-project-id",
		},
		{
			name:                "key file project ID takes precedence over env",
			configuredProjectID: "",
			authConfig:          cfg.GcsAuthConfig{KeyFile: cfg.ResolvedPath(validKeyFile)},
			envProjectID:        "env-project-id",
			expected:            "file-project-id",
		},
		{
			name:                "falls back to env var when key file is invalid",
			configuredProjectID: "",
			authConfig:          cfg.GcsAuthConfig{KeyFile: cfg.ResolvedPath(invalidKeyFile)},
			envProjectID:        "env-project-id",
			expected:            "env-project-id",
		},
		{
			name:                "falls back to env var when key file does not exist",
			configuredProjectID: "",
			authConfig:          cfg.GcsAuthConfig{KeyFile: cfg.ResolvedPath("nonexistent.json")},
			envProjectID:        "env-project-id",
			expected:            "env-project-id",
		},
		{
			name:                "falls back to env var when no auth config provided",
			configuredProjectID: "",
			authConfig:          cfg.GcsAuthConfig{TokenUrl: "some-url"}, // TokenUrl prevents default credentials
			envProjectID:        "env-project-id",
			expected:            "env-project-id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			if tc.envProjectID != "" {
				t.Setenv("GOOGLE_CLOUD_PROJECT", tc.envProjectID)
			} else {
				_ = os.Unsetenv("GOOGLE_CLOUD_PROJECT")
			}

			// Act
			actual := getProjectID(ctx, tc.authConfig, tc.configuredProjectID)

			// Assert
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSetupOTelMetricExporters(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *cfg.Config
		expectOtel    bool
		expectMetrics bool
	}{
		{
			name: "OTel Metrics Enabled",
			cfg: &cfg.Config{
				Metrics: cfg.MetricsConfig{
					ExperimentalEnableOtelMetrics:   true,
					ExperimentalOtelMetricsEndpoint: "localhost:4318",
					CloudMetricsExportIntervalSecs:  5,
				},
			},
			expectOtel: true,
		},
		{
			name: "OTel Metrics Disabled",
			cfg: &cfg.Config{
				Metrics: cfg.MetricsConfig{
					ExperimentalEnableOtelMetrics:  false,
					CloudMetricsExportIntervalSecs: 5,
				},
			},
			expectOtel: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			shutdown := SetupOTelMetricExporters(ctx, tc.cfg, "mount-id")

			// Assert
			assert.NotNil(t, shutdown)
			_ = shutdown(ctx) // Ignoring error as it forces a flush to a nonexistent local port which returns connection refused
		})
	}
}

func TestSetupOtelMetricsEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		intervalSecs int64
		expectOpts   bool
		expectErr    bool
	}{
		{
			name:         "Disabled interval",
			endpoint:     "localhost:4318",
			intervalSecs: 0,
			expectOpts:   false,
			expectErr:    false,
		},
		{
			name:         "Localhost insecure",
			endpoint:     "localhost:4318",
			intervalSecs: 10,
			expectOpts:   true,
			expectErr:    false,
		},
		{
			name:         "Normal endpoint",
			endpoint:     "otel-collector.default:4318",
			intervalSecs: 10,
			expectOpts:   true,
			expectErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			opts, err := setupOtelMetricsEndpoint(ctx, tc.endpoint, cfg.GcsAuthConfig{}, tc.intervalSecs)

			// Assert
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, opts)
			} else {
				assert.NoError(t, err)
				if tc.expectOpts {
					assert.NotEmpty(t, opts)
				} else {
					assert.Empty(t, opts)
				}
			}
		})
	}
}

func TestGetOtelResource(t *testing.T) {
	tests := []struct {
		name        string
		projectID   string
		expectFound bool
	}{
		{
			name:        "Without Project ID",
			projectID:   "",
			expectFound: false,
		},
		{
			name:        "With Project ID",
			projectID:   "test-project-123",
			expectFound: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			res, err := getOtelResource(ctx, "mount-1", tc.projectID)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, res)

			var found bool
			var val string
			for _, attr := range res.Attributes() {
				if string(attr.Key) == "gcp.project_id" {
					found = true
					val = attr.Value.AsString()
				}
			}

			if tc.expectFound {
				assert.True(t, found)
				assert.Equal(t, tc.projectID, val)
			} else {
				assert.False(t, found)
			}
		})
	}
}

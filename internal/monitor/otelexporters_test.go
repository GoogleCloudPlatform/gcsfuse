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
	"sort"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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

func TestIsRunningOnGKE(t *testing.T) {
	t.Run("KubernetesServiceHostEnvVarSet", func(t *testing.T) {
		t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")

		assert.True(t, isRunningOnGKE(nil))
	})

	t.Run("NoEnvVarAndNilResource", func(t *testing.T) {
		t.Setenv(kubernetesServiceHostEnvVar, "")

		assert.False(t, isRunningOnGKE(nil))
	})

	t.Run("DetectedViaK8sResourceLabel", func(t *testing.T) {
		t.Setenv(kubernetesServiceHostEnvVar, "")
		res, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("k8s.cluster.name", "my-cluster")))
		require.NoError(t, err)

		assert.True(t, isRunningOnGKE(res))
	})

	t.Run("DetectedViaCloudPlatformResourceLabel", func(t *testing.T) {
		t.Setenv(kubernetesServiceHostEnvVar, "")
		res, err := resource.New(t.Context(), resource.WithAttributes(semconv.CloudPlatformGCPKubernetesEngine))
		require.NoError(t, err)

		assert.True(t, isRunningOnGKE(res))
	})

	t.Run("GceResourceIsNotGKE", func(t *testing.T) {
		t.Setenv(kubernetesServiceHostEnvVar, "")
		res, err := resource.New(t.Context(), resource.WithAttributes(semconv.CloudPlatformGCPComputeEngine))
		require.NoError(t, err)

		assert.False(t, isRunningOnGKE(res))
	})
}

func TestIsSensitiveEnvKey(t *testing.T) {
	testCases := []struct {
		key      string
		expected bool
	}{
		{"HOSTNAME", false},
		{"KUBERNETES_SERVICE_HOST", false},
		{"GOOGLE_CLOUD_PROJECT", false},
		{"GOOGLE_APPLICATION_CREDENTIALS", false},
		{"SSH_AUTH_SOCK", false},
		{"XDG_SESSION_TYPE", false},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"aws_session_token", true},
		{"MY_API_KEY", true},
		{"DB_PASSWORD", true},
		{"SOME_PRIVATE_DATA", true},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			assert.Equal(t, tc.expected, isSensitiveEnvKey(tc.key))
		})
	}
}

func TestRedactEnvValue(t *testing.T) {
	t.Run("PlainValueIsReturnedAsIs", func(t *testing.T) {
		assert.Equal(t, "my-project", redactEnvValue("GOOGLE_CLOUD_PROJECT", "my-project"))
	})

	t.Run("SensitiveValueIsRedacted", func(t *testing.T) {
		got := redactEnvValue("AWS_SECRET_ACCESS_KEY", "super-secret")

		assert.NotContains(t, got, "super-secret")
		assert.Equal(t, "<redacted: 12 chars>", got)
	})

	t.Run("BooleanAndNumericValuesAreNotRedacted", func(t *testing.T) {
		assert.Equal(t, "true", redactEnvValue("CLOUDSDK_USE_APPLICATION_DEFAULT_CREDENTIALS", "true"))
		assert.Equal(t, "0", redactEnvValue("SOME_KEY_COUNT", "0"))
		assert.Equal(t, "", redactEnvValue("EMPTY_TOKEN", ""))
	})

	t.Run("LongValueIsTruncated", func(t *testing.T) {
		value := strings.Repeat("a", maxLoggedEnvValueLen+10)

		got := redactEnvValue("SOME_LONG_VAR", value)

		assert.True(t, strings.HasPrefix(got, strings.Repeat("a", maxLoggedEnvValueLen)))
		assert.True(t, strings.HasSuffix(got, "...<truncated: 1034 chars total>"))
	})
}

func TestEnvironmentVariableLines(t *testing.T) {
	t.Setenv("GCSFUSE_TEST_PLAIN_VAR", "plain-value")
	t.Setenv("GCSFUSE_TEST_SECRET_TOKEN", "do-not-log-me")

	lines := environmentVariableLines()

	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "  GCSFUSE_TEST_PLAIN_VAR = plain-value")
	assert.Contains(t, joined, "  GCSFUSE_TEST_SECRET_TOKEN = <redacted: 13 chars>")
	assert.NotContains(t, joined, "do-not-log-me")
	assert.True(t, sort.StringsAreSorted(lines))
}

func TestResourceAttributeLines(t *testing.T) {
	t.Run("NilResource", func(t *testing.T) {
		assert.Empty(t, resourceAttributeLines(nil))
	})

	t.Run("AttributesAreSorted", func(t *testing.T) {
		res, err := resource.New(t.Context(),
			resource.WithAttributes(
				attribute.String("service.name", "gcsfuse"),
				attribute.String("cloud.region", "us-central1"),
				attribute.Int64("host.id", 42),
			),
		)
		require.NoError(t, err)

		lines := resourceAttributeLines(res)

		assert.Equal(t, []string{
			"  cloud.region = us-central1",
			"  host.id = 42",
			"  service.name = gcsfuse",
		}, lines)
	})
}

func TestLogMountEnvironmentDoesNotPanic(t *testing.T) {
	res, err := resource.New(t.Context(), resource.WithAttributes(attribute.String("service.name", "gcsfuse")))
	require.NoError(t, err)

	assert.NotPanics(t, func() { logMountEnvironment(res) })
	assert.NotPanics(t, func() { logMountEnvironment(nil) })
}

// clearK8sEnv unsets every variable k8sResourceAttributes consults, so each
// test starts from a known state regardless of where it runs.
func clearK8sEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"KUBERNETES_SERVICE_HOST", "POD_NAME", "HOSTNAME", "NAMESPACE_NAME",
		"POD_NAMESPACE", "NAMESPACE", "POD_UID", "CONTAINER_NAME", "NODE_NAME",
	} {
		t.Setenv(k, "")
	}
	// Point the service-account file at a path that cannot exist, so the
	// namespace fallback stays inert unless a test opts into it.
	old := k8sSANamespaceFile
	k8sSANamespaceFile = filepath.Join(t.TempDir(), "absent")
	t.Cleanup(func() { k8sSANamespaceFile = old })
}

func attrValue(attrs []attribute.KeyValue, key attribute.Key) (string, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value.Emit(), true
		}
	}
	return "", false
}

func TestK8sResourceAttributes(t *testing.T) {
	t.Run("EmptyOutsideKubernetes", func(t *testing.T) {
		clearK8sEnv(t)
		// HOSTNAME is set on every machine and must not leak into labels.
		t.Setenv("HOSTNAME", "some-gce-vm")

		assert.Empty(t, k8sResourceAttributes())
	})

	t.Run("PodNameFromHostnameFallback", func(t *testing.T) {
		clearK8sEnv(t)
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		t.Setenv("HOSTNAME", "gcsfuse-envlog")

		got, ok := attrValue(k8sResourceAttributes(), semconv.K8SPodNameKey)

		assert.True(t, ok)
		assert.Equal(t, "gcsfuse-envlog", got)
	})

	t.Run("PodNameEnvWinsOverHostname", func(t *testing.T) {
		clearK8sEnv(t)
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		t.Setenv("HOSTNAME", "gcsfuse-envlog")
		t.Setenv("POD_NAME", "explicit-pod")

		got, _ := attrValue(k8sResourceAttributes(), semconv.K8SPodNameKey)

		assert.Equal(t, "explicit-pod", got)
	})

	t.Run("NamespaceFromEnv", func(t *testing.T) {
		clearK8sEnv(t)
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		t.Setenv("NAMESPACE_NAME", "team-ns")

		got, ok := attrValue(k8sResourceAttributes(), semconv.K8SNamespaceNameKey)

		assert.True(t, ok)
		assert.Equal(t, "team-ns", got)
	})

	t.Run("NamespaceFromServiceAccountFileFallback", func(t *testing.T) {
		clearK8sEnv(t)
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		nsFile := filepath.Join(t.TempDir(), "namespace")
		require.NoError(t, os.WriteFile(nsFile, []byte("sidecar-ns\n"), 0o600))
		k8sSANamespaceFile = nsFile

		got, ok := attrValue(k8sResourceAttributes(), semconv.K8SNamespaceNameKey)

		assert.True(t, ok)
		assert.Equal(t, "sidecar-ns", got)
	})

	t.Run("OptionalAttributesOnlyWhenSet", func(t *testing.T) {
		clearK8sEnv(t)
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		t.Setenv("HOSTNAME", "pod-1")

		attrs := k8sResourceAttributes()

		_, hasUID := attrValue(attrs, semconv.K8SPodUIDKey)
		_, hasContainer := attrValue(attrs, semconv.K8SContainerNameKey)
		_, hasNode := attrValue(attrs, semconv.K8SNodeNameKey)
		assert.False(t, hasUID)
		assert.False(t, hasContainer)
		assert.False(t, hasNode)

		t.Setenv("POD_UID", "uid-123")
		t.Setenv("CONTAINER_NAME", "gcsfuse")
		t.Setenv("NODE_NAME", "gke-node-1")
		attrs = k8sResourceAttributes()

		uid, _ := attrValue(attrs, semconv.K8SPodUIDKey)
		container, _ := attrValue(attrs, semconv.K8SContainerNameKey)
		node, _ := attrValue(attrs, semconv.K8SNodeNameKey)
		assert.Equal(t, "uid-123", uid)
		assert.Equal(t, "gcsfuse", container)
		assert.Equal(t, "gke-node-1", node)
	})
}

func TestFirstNonEmptyEnv(t *testing.T) {
	t.Setenv("GCSFUSE_TEST_EMPTY", "")
	t.Setenv("GCSFUSE_TEST_SECOND", "second")
	t.Setenv("GCSFUSE_TEST_THIRD", "third")

	assert.Equal(t, "second", firstNonEmptyEnv("GCSFUSE_TEST_EMPTY", "GCSFUSE_TEST_SECOND", "GCSFUSE_TEST_THIRD"))
	assert.Equal(t, "", firstNonEmptyEnv("GCSFUSE_TEST_EMPTY", "GCSFUSE_TEST_ABSENT"))
	assert.Equal(t, "", firstNonEmptyEnv())
}

func TestGetResourceHonoursOtelResourceAttributes(t *testing.T) {
	clearK8sEnv(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "gcsfuse.bucket_name=my-bucket")

	res, err := getResource(t.Context(), "mount-1")

	require.NoError(t, err)
	require.NotNil(t, res)
	got, ok := attrValue(res.Attributes(), attribute.Key("gcsfuse.bucket_name"))
	assert.True(t, ok, "OTEL_RESOURCE_ATTRIBUTES should reach the resource")
	assert.Equal(t, "my-bucket", got)
}

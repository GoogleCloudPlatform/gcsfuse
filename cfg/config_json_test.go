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

package cfg

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeConfigToGzippedJSONBase64_Sanitization(t *testing.T) {
	mountConfig := &Config{
		AppName:  "my-ml-worker-app",
		CacheDir: "/usr/local/google/home/cpranjal/sensitive_cache_dir", // SENSITIVE
		CloudProfiler: CloudProfilerConfig{
			Enabled:     true,
			ServiceName: "cloud-profiler-service",
		},
		FileSystem: FileSystemConfig{
			DirMode:          0755,
			FileMode:         0644,
			TempDir:          "/tmp/sensitive_temp_dir",      // SENSITIVE
			KernelParamsFile: "/etc/sensitive_kernel_params", // SENSITIVE
		},
		GcsAuth: GcsAuthConfig{
			AnonymousAccess:   false,
			KeyFile:           "/path/to/sensitive/key.json", // SENSITIVE
			TokenUrl:          "https://sensitive-token-url", // SENSITIVE
			ReuseTokenFromUrl: true,
		},
		GcsConnection: GcsConnectionConfig{
			ClientProtocol:                 GRPC,
			CustomEndpoint:                 "https://custom.endpoint.com", // SENSITIVE
			ExperimentalLocalSocketAddress: "192.168.1.50",                // SENSITIVE
			MaxIdleConnsPerHost:            100,
			HttpClientTimeout:              15 * time.Second,
		},
		Logging: LoggingConfig{
			FilePath: "/var/log/sensitive_gcsfuse.log", // SENSITIVE
			Format:   "json",
			Severity: "INFO",
			WireLog:  "true", // SENSITIVE
		},
		MetadataCache: MetadataCacheConfig{
			StatCacheMaxSizeMb: 33,
			TtlSecs:            60,
		},
	}

	// Serialize
	base64Str, err := SerializeConfigToGzippedJSONBase64(mountConfig)
	assert.NoError(t, err)
	assert.NotEmpty(t, base64Str)

	// Decode Base64
	gzippedBytes, err := base64.RawURLEncoding.DecodeString(base64Str)
	assert.NoError(t, err)

	// Decompress Gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(gzippedBytes))
	assert.NoError(t, err)
	defer gzipReader.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(gzipReader)
	assert.NoError(t, err)

	// Unmarshal JSON
	var decodedConfig Config
	err = json.Unmarshal(buf.Bytes(), &decodedConfig)
	assert.NoError(t, err)

	// 1. Assert that all sensitive/PII fields are completely empty/scrubbed!
	assert.Empty(t, decodedConfig.CacheDir)
	assert.Empty(t, decodedConfig.FileSystem.TempDir)
	assert.Empty(t, decodedConfig.FileSystem.KernelParamsFile)
	assert.Empty(t, decodedConfig.GcsAuth.KeyFile)
	assert.Empty(t, decodedConfig.GcsAuth.TokenUrl)
	assert.Empty(t, decodedConfig.GcsConnection.CustomEndpoint)
	assert.Empty(t, decodedConfig.GcsConnection.ExperimentalLocalSocketAddress)
	assert.Empty(t, decodedConfig.Logging.FilePath)
	assert.Empty(t, decodedConfig.Logging.WireLog)

	// 2. Assert that non-sensitive fields are retained correctly!
	assert.Equal(t, "my-ml-worker-app", decodedConfig.AppName)
	assert.True(t, decodedConfig.CloudProfiler.Enabled)
	assert.Equal(t, "cloud-profiler-service", decodedConfig.CloudProfiler.ServiceName)
	assert.Equal(t, Octal(0755), decodedConfig.FileSystem.DirMode)
	assert.Equal(t, Octal(0644), decodedConfig.FileSystem.FileMode)
	assert.False(t, decodedConfig.GcsAuth.AnonymousAccess)
	assert.True(t, decodedConfig.GcsAuth.ReuseTokenFromUrl)
	assert.Equal(t, Protocol(GRPC), decodedConfig.GcsConnection.ClientProtocol)
	assert.Equal(t, int64(100), decodedConfig.GcsConnection.MaxIdleConnsPerHost)
	assert.Equal(t, 15*time.Second, decodedConfig.GcsConnection.HttpClientTimeout)
	assert.Equal(t, "json", decodedConfig.Logging.Format)
	assert.Equal(t, InfoLogSeverity, decodedConfig.Logging.Severity)
	assert.Equal(t, int64(33), decodedConfig.MetadataCache.StatCacheMaxSizeMb)
	assert.Equal(t, int64(60), decodedConfig.MetadataCache.TtlSecs)
}

func TestSerializeConfigToGzippedJSONBase64_EncodingAndLimits(t *testing.T) {
	// Verify truncation of long strings.
	mountConfig := &Config{
		AppName: "this-is-a-very-long-app-name-that-definitely-exceeds-fifty-characters", // 69 chars
	}
	base64Str, err := SerializeConfigToGzippedJSONBase64(mountConfig)
	assert.NoError(t, err)

	// Check encoding doesn't contain forbidden characters
	assert.NotContains(t, base64Str, "/")
	assert.NotContains(t, base64Str, "+")
	assert.NotContains(t, base64Str, "=")

	// Decode
	gzippedBytes, err := base64.RawURLEncoding.DecodeString(base64Str)
	require.NoError(t, err)

	gzipReader, err := gzip.NewReader(bytes.NewReader(gzippedBytes))
	require.NoError(t, err)
	defer gzipReader.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(gzipReader)
	require.NoError(t, err)

	var decodedConfig Config
	err = json.Unmarshal(buf.Bytes(), &decodedConfig)
	require.NoError(t, err)

	// AppName should be truncated to 49 chars + "+" (total 50)
	expectedAppName := "this-is-a-very-long-app-name-that-definitely-exce+"
	assert.Equal(t, 50, len(decodedConfig.AppName))
	assert.Equal(t, expectedAppName, decodedConfig.AppName)
}

func TestSerializeConfigToGzippedJSONBase64_NilConfig(t *testing.T) {
	base64Str, err := SerializeConfigToGzippedJSONBase64(nil)
	assert.NoError(t, err)
	assert.Empty(t, base64Str)
}

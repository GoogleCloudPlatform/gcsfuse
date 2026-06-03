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
	"encoding/base64"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSerializeConfigToProtoBase64_Sanitization(t *testing.T) {
	// Create a dummy configuration with non-zero/non-empty values for both
	// sensitive and non-sensitive fields.
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
	base64Str, err := SerializeConfigToProtoBase64(mountConfig)
	assert.NoError(t, err)
	assert.NotEmpty(t, base64Str)

	// Decode Base64
	protoBytes, err := base64.RawURLEncoding.DecodeString(base64Str)
	assert.NoError(t, err)

	// Unmarshal Protobuf
	p := &pb.Config{}
	err = proto.Unmarshal(protoBytes, p)
	assert.NoError(t, err)

	// 1. Assert that all sensitive/PII fields are completely empty/scrubbed!
	assert.Empty(t, p.CacheDir)
	assert.Empty(t, p.FileSystem.TempDir)
	assert.Empty(t, p.FileSystem.KernelParamsFile)
	assert.Empty(t, p.GcsAuth.KeyFile)
	assert.Empty(t, p.GcsAuth.TokenUrl)
	assert.Empty(t, p.GcsConnection.CustomEndpoint)
	assert.Empty(t, p.GcsConnection.ExperimentalLocalSocketAddress)
	assert.Empty(t, p.Logging.FilePath)
	assert.Empty(t, p.Logging.WireLog)

	// 2. Assert that non-sensitive fields are retained correctly!
	assert.Equal(t, "my-ml-worker-app", p.AppName)
	assert.True(t, p.CloudProfiler.Enabled)
	assert.Equal(t, "cloud-profiler-service", p.CloudProfiler.ServiceName)
	assert.Equal(t, uint32(0755), p.FileSystem.DirMode)
	assert.Equal(t, uint32(0644), p.FileSystem.FileMode)
	assert.False(t, p.GcsAuth.AnonymousAccess)
	assert.True(t, p.GcsAuth.ReuseTokenFromUrl)
	assert.Equal(t, string(GRPC), p.GcsConnection.ClientProtocol)
	assert.Equal(t, int64(100), p.GcsConnection.MaxIdleConnsPerHost)
	assert.Equal(t, int64(15*time.Second), p.GcsConnection.HttpClientTimeout)
	assert.Equal(t, "json", p.Logging.Format)
	assert.Equal(t, "INFO", p.Logging.Severity)
	assert.Equal(t, int64(33), p.MetadataCache.StatCacheMaxSizeMb)
	assert.Equal(t, int64(60), p.MetadataCache.TtlSecs)
}

func TestSerializeConfigToProtoBase64_EncodingAndLimits(t *testing.T) {
	// 1. Verify truncation of long strings.
	mountConfig := &Config{
		AppName: "this-is-a-very-long-app-name-that-definitely-exceeds-fifty-characters", // 69 chars
	}
	base64Str, err := SerializeConfigToProtoBase64(mountConfig)
	assert.NoError(t, err)

	// Check encoding doesn't contain forbidden characters
	assert.NotContains(t, base64Str, "/")
	assert.NotContains(t, base64Str, "+")
	assert.NotContains(t, base64Str, "=")

	// Decode
	protoBytes, err := base64.RawURLEncoding.DecodeString(base64Str)
	require.NoError(t, err)

	p := &pb.Config{}
	err = proto.Unmarshal(protoBytes, p)
	require.NoError(t, err)

	// AppName should be truncated to 49 chars + "+" (total 50)
	expectedAppName := "this-is-a-very-long-app-name-that-definitely-exce+"
	assert.Equal(t, 50, len(p.AppName))
	assert.Equal(t, expectedAppName, p.AppName)
}


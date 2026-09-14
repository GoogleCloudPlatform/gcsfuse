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

func TestToProto_Nil(t *testing.T) {
	var config *Config

	protoConfig := config.ToProto()

	assert.Nil(t, protoConfig)
}

func TestToProto_AllDataTypes(t *testing.T) {
	testCases := []struct {
		name     string
		config   *Config
		expected *pb.Config
	}{
		{
			name:     "DefaultEmptyConfig",
			config:   &Config{},
			expected: &pb.Config{},
		},
		{
			name:   "ScrubbedPIIString",
			config: &Config{AppName: "custom-app"},
			expected: &pb.Config{
				IsAppNameSet: true,
			},
		},
		{
			name:   "ScrubbedPath",
			config: &Config{CacheDir: "/tmp/custom_cache"},
			expected: &pb.Config{
				IsCacheDirSet: true,
			},
		},
		{
			name:   "ScrubbedSlice",
			config: &Config{FileSystem: FileSystemConfig{FuseOptions: []string{"ro", "allow_other"}}},
			expected: &pb.Config{
				IsFileSystemFuseOptionsSet: true,
			},
		},
		{
			name:   "EmptyScrubbedSlice",
			config: &Config{FileSystem: FileSystemConfig{FuseOptions: []string{}}},
			expected: &pb.Config{
				IsFileSystemFuseOptionsSet: false,
			},
		},
		{
			name:   "WhitelistedString",
			config: &Config{MachineType: "a3-highgpu-8g"},
			expected: &pb.Config{
				MachineType: "a3-highgpu-8g",
			},
		},
		{
			name:   "WhitelistedEnum",
			config: &Config{GcsConnection: GcsConnectionConfig{ClientProtocol: "grpc"}},
			expected: &pb.Config{
				GcsConnectionClientProtocol: "grpc",
			},
		},
		{
			name:   "Boolean",
			config: &Config{CloudProfiler: CloudProfilerConfig{Enabled: true}},
			expected: &pb.Config{
				CloudProfilerEnabled: true,
			},
		},
		{
			name:   "Integer",
			config: &Config{FileCache: FileCacheConfig{MaxSizeMb: 1024}},
			expected: &pb.Config{
				FileCacheMaxSizeMb: 1024,
			},
		},
		{
			name:   "OctalFileMode",
			config: &Config{FileSystem: FileSystemConfig{DirMode: 0755}},
			expected: &pb.Config{
				FileSystemDirMode: 0755,
			},
		},
		{
			name:   "Float",
			config: &Config{Write: WriteConfig{BlockSizeMb: 32.5}},
			expected: &pb.Config{
				WriteBlockSizeMb: 32.5,
			},
		},
		{
			name:   "Duration",
			config: &Config{GcsConnection: GcsConnectionConfig{HttpClientTimeout: 30 * time.Second}},
			expected: &pb.Config{
				GcsConnectionHttpClientTimeout: int64(30 * time.Second),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.ToProto()

			require.NotNil(t, actual)
			assert.True(t, proto.Equal(tc.expected, actual))
		})
	}
}

func TestSerializeConfigToProtoBase64(t *testing.T) {
	testCases := []struct {
		name           string
		config         *Config
		expectedBytes  []byte
		expectedBase64 string
	}{
		{
			name:           "NilConfig",
			config:         nil,
			expectedBytes:  []byte{},
			expectedBase64: "",
		},
		{
			name:           "DefaultEmptyConfig",
			config:         &Config{},
			expectedBytes:  []byte{},
			expectedBase64: "",
		},
		{
			name: "Unpadded4BytePayload",
			config: &Config{
				CloudProfiler: CloudProfilerConfig{
					Cpu:     true,
					Enabled: true,
				},
			},
			// Tag 4 (cloud_profiler_cpu = true): 0x20, 0x01
			// Tag 5 (cloud_profiler_enabled = true): 0x28, 0x01
			// 4 bytes would be "IAEoAQ==" with standard padding, verifying RawURLEncoding strips "==".
			expectedBytes:  []byte{0x20, 0x01, 0x28, 0x01},
			expectedBase64: "IAEoAQ",
		},
		{
			name: "PositiveZigZagAndScrubbedString",
			config: &Config{
				AppName: "test-app",
				FileCache: FileCacheConfig{
					MaxSizeMb: 512,
				},
			},
			// Tag 1 (is_app_name_set = true): 0x08, 0x01
			// Tag 38 (file_cache_max_size_mb = 512, sint64 ZigZag(512)=1024): 0xb0, 0x02, 0x80, 0x08
			expectedBytes:  []byte{0x08, 0x01, 0xb0, 0x02, 0x80, 0x08},
			expectedBase64: "CAGwAoAI",
		},
		{
			name: "NegativeZigZagUnlimitedCache",
			config: &Config{
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			// Tag 38 (file_cache_max_size_mb = -1, sint64 ZigZag(-1)=1): 0xb0, 0x02, 0x01
			expectedBytes:  []byte{0xb0, 0x02, 0x01},
			expectedBase64: "sAIB",
		},
		{
			name: "URLSafeAlphabetCharacters",
			config: &Config{
				FileCache: FileCacheConfig{
					MaxSizeMb: 63,
				},
				FileSystem: FileSystemConfig{
					DirMode: 077,
				},
			},
			// Tag 38 (file_cache_max_size_mb = 63, sint64 ZigZag(63)=126=0x7e): 0xb0, 0x02, 0x7e -> Base64 "sAJ-" (index 62 '-')
			// Tag 43 (file_system_dir_mode = 077 = 63 = 0x3f): 0xd8, 0x02, 0x3f -> Base64 "2AI_" (index 63 '_')
			// Standard Base64 would encode this as "sAJ+2AI/".
			expectedBytes:  []byte{0xb0, 0x02, 0x7e, 0xd8, 0x02, 0x3f},
			expectedBase64: "sAJ-2AI_",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.config.SerializeConfigToProtoBase64()

			require.NoError(t, err)
			assert.Equal(t, tc.expectedBase64, encoded)
			assert.NotContains(t, encoded, "=")
			assert.NotContains(t, encoded, "+")
			assert.NotContains(t, encoded, "/")

			decodedBytes, err := base64.RawURLEncoding.DecodeString(encoded)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedBytes, decodedBytes)

			if tc.config != nil {
				var unmarshaled pb.Config

				err = proto.Unmarshal(decodedBytes, &unmarshaled)

				require.NoError(t, err)
				assert.True(t, proto.Equal(tc.config.ToProto(), &unmarshaled))
			}
		})
	}
}

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
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestToProto_Nil(t *testing.T) {
	var config *Config

	protoConfig := config.ToProto()

	assert.Nil(t, protoConfig)
}

func TestSerializeConfigToProtoBase64_Nil(t *testing.T) {
	var config *Config

	encoded, err := config.SerializeConfigToProtoBase64()

	require.NoError(t, err)
	assert.Empty(t, encoded)
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
			name:   "WhitelistedStringOverCap",
			config: &Config{MachineType: strings.Repeat("a", MaxProtoStringLen+1)},
			expected: &pb.Config{
				MachineType: strings.Repeat("a", MaxProtoStringLen),
			},
		},
		{
			// The cut at MaxProtoStringLen bytes splits the 2-byte "é", which is dropped.
			name:   "WhitelistedStringOverCapSplitsMultiByteChar",
			config: &Config{MachineType: strings.Repeat("a", MaxProtoStringLen-1) + "é"},
			expected: &pb.Config{
				MachineType: strings.Repeat("a", MaxProtoStringLen-1),
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
			name:   "NegativeInteger",
			config: &Config{FileCache: FileCacheConfig{MaxSizeMb: -1}},
			expected: &pb.Config{
				FileCacheMaxSizeMb: -1,
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
			t.Run("ToProto", func(t *testing.T) {
				actual := tc.config.ToProto()

				require.NotNil(t, actual)
				assert.True(t, proto.Equal(tc.expected, actual))
			})

			t.Run("Base64Format", func(t *testing.T) {
				encoded, err := tc.config.SerializeConfigToProtoBase64()

				require.NoError(t, err)
				// Asserts RFC 4648 URL-safe Base64 without padding (strictly [A-Za-z0-9_-]).
				assert.Regexp(t, `^[A-Za-z0-9_-]+$`, encoded)
			})
		})
	}
}

func TestTruncateProtoString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "Empty",
			input:    "",
			maxLen:   MaxProtoStringLen,
			expected: "",
		},
		{
			name:     "ShortValueUnchanged",
			input:    "n2-standard-4",
			maxLen:   MaxProtoStringLen,
			expected: "n2-standard-4",
		},
		{
			name:     "ValueAtCapUnchanged",
			input:    strings.Repeat("a", MaxProtoStringLen),
			maxLen:   MaxProtoStringLen,
			expected: strings.Repeat("a", MaxProtoStringLen),
		},
		{
			name:     "LongValueCapped",
			input:    strings.Repeat("a", 1000),
			maxLen:   MaxProtoStringLen,
			expected: strings.Repeat("a", MaxProtoStringLen),
		},
		{
			name:     "OtherMaxLen",
			input:    strings.Repeat("a", 1000),
			maxLen:   256,
			expected: strings.Repeat("a", 256),
		},
		{
			name:     "SplitMultiByteCharDropped",
			input:    strings.Repeat("a", MaxProtoStringLen-1) + "é",
			maxLen:   MaxProtoStringLen,
			expected: strings.Repeat("a", MaxProtoStringLen-1),
		},
		{
			name:     "InvalidUTF8Replaced",
			input:    "n2-standard-4\xff",
			maxLen:   MaxProtoStringLen,
			expected: "n2-standard-4\uFFFD",
		},
		{
			// The 3-byte replacement character doesn't fit, so it's dropped.
			name:     "InvalidUTF8ReplacedThenCapped",
			input:    strings.Repeat("a", MaxProtoStringLen-1) + "\xff",
			maxLen:   MaxProtoStringLen,
			expected: strings.Repeat("a", MaxProtoStringLen-1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := TruncateProtoString(tc.input, tc.maxLen)

			assert.Equal(t, tc.expected, actual)
			assert.LessOrEqual(t, len(actual), tc.maxLen)
			assert.True(t, utf8.ValidString(actual))
		})
	}
}

func TestTruncateProtoString_ValidValuesUnchanged(t *testing.T) {
	// Every valid enum and profile value must fit within the cap.
	validValues := []string{
		HTTP1, HTTP2, GRPC, HTTPMtls,
		string(DirectPathOnly), string(DirectPathWithFallback),
		string(RapidWriteStrategyEnabled), string(RapidWriteStrategyDisabled),
		ProfileAIMLTraining, ProfileAIMLServing, ProfileAIMLCheckpointing,
	}
	for severity := range severityRanking {
		validValues = append(validValues, string(severity))
	}
	// So must the longest predefined GCE machine type (Nov 2026) and the machine
	// types that have optimization rules.
	validValues = append(validValues, "z4d-highmem-384-standardlssd-metal")
	for machineType := range machineTypeToGroupMap {
		validValues = append(validValues, machineType)
	}

	for _, value := range validValues {
		t.Run(value, func(t *testing.T) {
			actual := TruncateProtoString(value, MaxProtoStringLen)

			assert.Equal(t, value, actual)
		})
	}
}

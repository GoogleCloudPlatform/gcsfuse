// Copyright 2024 Google LLC
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func validLogRotateConfig() LogRotateLoggingConfig {
	return LogRotateLoggingConfig{
		BackupFileCount: 0,
		Compress:        false,
		MaxFileSizeMb:   1,
	}
}

func validFileCacheConfig(t *testing.T) FileCacheConfig {
	t.Helper()
	return FileCacheConfig{
		CacheFileForRangeRead:    false,
		DownloadChunkSizeMb:      50,
		EnableCrc:                false,
		EnableParallelDownloads:  false,
		MaxParallelDownloads:     4,
		MaxSizeMb:                -1,
		ParallelDownloadsPerFile: 16,
		WriteBufferSize:          4 * 1024 * 1024,
	}
}

func TestValidateConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Valid Config where input and expected custom endpoint match.",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "https://bing.com/search?q=dotnet",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
			},
		},
		{
			name: "Valid Config where input and expected custom endpoint differ.",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "https://j@ne:password@google.com",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount disabled",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount async",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "async",
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount sync",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "Valid Sequential read size MB",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 10,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
		{
			name: "Valid Sequential read size MB",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 10,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
		{
			name: "valid_kernel_list_cache_TTL",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 10,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				FileSystem: FileSystemConfig{KernelListCacheTtlSecs: 30},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := ValidateConfig(tc.config)

			assert.NoError(t, actualErr)
		})
	}
}

func TestValidateConfig_ErrorScenarios(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config due to invalid custom endpoint",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "a_b://abc",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
		{
			name: "Invalid experimental-metadata-prefetch-on-mount",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "a",
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "Invalid Config due to invalid token URL",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsAuth: GcsAuthConfig{
					TokenUrl: "a_b://abc",
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "Sequential read size MB more than 1024 (max permissible value)",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 2048,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
		{
			name: "Sequential read size MB less than 1 (min permissible value)",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 0,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
		{
			name: "kernel_list_cache_TTL_negative",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				FileSystem: FileSystemConfig{KernelListCacheTtlSecs: -2},
			},
		},
		{
			name: "kernel_list_cache_TTL_too_large",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				FileSystem: FileSystemConfig{KernelListCacheTtlSecs: 88888888888888888},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, ValidateConfig(tc.config))
		})
	}
}

func Test_IsTtlInSecsValid_ErrorScenarios(t *testing.T) {
	var testCases = []struct {
		testName  string
		ttlInSecs int64
	}{
		{"negative", -5},
		{"unsupported_large_positive", 9223372037},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Error(t, isTTLInSecsValid(tc.ttlInSecs))
		})
	}
}

func Test_IsTtlInSecsValid_ValidScenarios(t *testing.T) {
	var testCases = []struct {
		testName  string
		ttlInSecs int64
	}{
		{"valid_negative", -1},
		{"positive", 8},
		{"zero", 0},
		{"valid_upper_limit", 9223372036},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.NoError(t, isTTLInSecsValid(tc.ttlInSecs))
		})
	}
}

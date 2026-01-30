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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/spf13/viper"
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
		EnableODirect:            true,
	}
}

func validFileCacheConfigWithExcludeRegex(t *testing.T, r string) FileCacheConfig {
	t.Helper()
	cfg := validFileCacheConfig(t)
	cfg.ExcludeRegex = r
	return cfg
}

func validFileCacheConfigWithIncludeRegex(t *testing.T, r string) FileCacheConfig {
	t.Helper()
	cfg := validFileCacheConfig(t)
	cfg.IncludeRegex = r
	return cfg
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
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
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				FileSystem: FileSystemConfig{KernelListCacheTtlSecs: 30},
				Mrd: MrdConfig{
					PoolSize: 4,
				},
			},
		},
		{
			name: "valid_parallel_download_config_with_file_cache_enabled",
			config: &Config{
				Logging:  LoggingConfig{LogRotate: validLogRotateConfig()},
				CacheDir: "/some/valid/path",
				FileCache: FileCacheConfig{
					DownloadChunkSizeMb:      50,
					EnableParallelDownloads:  true,
					MaxParallelDownloads:     4,
					ParallelDownloadsPerFile: 16,
					MaxSizeMb:                -1,
					WriteBufferSize:          4 * 1024 * 1024,
				},
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "https://bing.com/search?q=dotnet",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
				},
			},
		},
		{
			name: "valid_file_cache_exclude_config",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				CacheDir:  "/some/valid/path",
				FileCache: validFileCacheConfigWithExcludeRegex(t, ".*"),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "https://bing.com/search?q=dotnet",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
				},
			},
		},
		{
			name: "valid_file_cache_include_config",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				CacheDir:  "/some/valid/path",
				FileCache: validFileCacheConfigWithIncludeRegex(t, ".*"),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint:       "https://bing.com/search?q=dotnet",
					SequentialReadSizeMb: 200,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				Mrd: MrdConfig{
					PoolSize: 4,
				},
			},
		},
		{
			name: "valid_chunk_transfer_timeout_secs",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 10,
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				Metrics: MetricsConfig{
					Workers:    3,
					BufferSize: 256,
				},
				FileSystem: FileSystemConfig{KernelListCacheTtlSecs: 30},
				GcsRetries: GcsRetriesConfig{ChunkTransferTimeoutSecs: 15},
				Mrd: MrdConfig{
					PoolSize: 4,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := ValidateConfig(viper.New(), tc.config)

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
		{
			name: "read_stall_req_increase_rate_negative",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsRetries: GcsRetriesConfig{
					ReadStall: ReadStallGcsRetriesConfig{
						Enable:          true,
						ReqIncreaseRate: -1,
					},
				},
			},
		},
		{
			name: "read_stall_req_increase_rate_zero",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsRetries: GcsRetriesConfig{
					ReadStall: ReadStallGcsRetriesConfig{
						Enable:          true,
						ReqIncreaseRate: 0,
					},
				},
			},
		},
		{
			name: "read_stall_req_target_percentile_large",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsRetries: GcsRetriesConfig{
					ReadStall: ReadStallGcsRetriesConfig{
						Enable:              true,
						ReqTargetPercentile: 4,
					},
				},
			},
		},
		{
			name: "read_stall_req_target_percentile_negative",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsRetries: GcsRetriesConfig{
					ReadStall: ReadStallGcsRetriesConfig{
						Enable:              true,
						ReqTargetPercentile: -3,
					},
				},
			},
		},
		{
			//TODO: Remove this test as check is also removed when parallel download is default ON
			name: "parallel_download_config_without_file_cache_enabled",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: FileCacheConfig{
					DownloadChunkSizeMb:      50,
					EnableParallelDownloads:  true,
					MaxParallelDownloads:     4,
					ParallelDownloadsPerFile: 16,
					WriteBufferSize:          4 * 1024 * 1024,
				},
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
			name: "file_cache_exclude_regex",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfigWithExcludeRegex(t, "["),
			},
		},
		{
			name: "file_cache_include_regex",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfigWithIncludeRegex(t, "["),
			},
		},
		{
			name: "chunk_transfer_timeout_in_negative",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
				GcsRetries: GcsRetriesConfig{
					ChunkTransferTimeoutSecs: -5,
				},
			},
		},
		{
			name: "Invalid experimental-concurrent-metadata-prefetches",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					MetadataPrefetchMaxWorkers: -4,
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
		{
			name: "Invalid experimental-metadata-prefetch-count",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					MetadataPrefetchEntriesLimit: -2,
				},
				GcsConnection: GcsConnectionConfig{
					SequentialReadSizeMb: 200,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, ValidateConfig(viper.New(), tc.config))
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

func Test_isValidWriteStreamingConfig_ErrorScenarios(t *testing.T) {
	var testCases = []struct {
		testName    string
		writeConfig WriteConfig
	}{
		{"zero_block_size", WriteConfig{
			BlockSizeMb:           0,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       -1,
			MaxBlocksPerFile:      -1,
		}},
		{"very_large_block_size", WriteConfig{
			BlockSizeMb:           util.MaxMiBsInInt64 + 1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       -1,
			MaxBlocksPerFile:      -1,
		}},
		{"negative_block_size", WriteConfig{
			BlockSizeMb:           -1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       -1,
			MaxBlocksPerFile:      -1,
		}},
		{"-2_global_max_blocks", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       -2,
			MaxBlocksPerFile:      -1,
		}},
		{"-2_max_blocks_per_file", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      -2,
		}},
		{"0_max_blocks_per_file", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      0,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Error(t, isValidWriteStreamingConfig(&tc.writeConfig))
		})
	}
}

func Test_isValidBufferedReadConfig_ErrorScenarios(t *testing.T) {
	var testCases = []struct {
		testName string
		read     ReadConfig
	}{
		{"negative_block_size", ReadConfig{
			BlockSizeMb:          -1,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   4,
		}},
		{"zero_block_size", ReadConfig{
			BlockSizeMb:          0,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   4,
		}},
		{"negative_global_max_blocks", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -2,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   4,
		}},
		{"negative_max_blocks_per_handle", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -2,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   4,
		}},
		{"negative_min_blocks_per_handle", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   -4,
		}},
		{"zero_min_blocks_per_handle", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   0,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Error(t, isValidBufferedReadConfig(&tc.read))
		})
	}
}

func Test_isValidMRDConfig(t *testing.T) {
	testCases := []struct {
		name      string
		mrdConfig MrdConfig
		wantErr   bool
	}{
		{
			name: "valid_pool_size",
			mrdConfig: MrdConfig{
				PoolSize: 10,
			},
			wantErr: false,
		},
		{
			name: "invalid_pool_size_zero",
			mrdConfig: MrdConfig{
				PoolSize: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid_pool_size_negative",
			mrdConfig: MrdConfig{
				PoolSize: -1,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := isValidMRDConfig(&tc.mrdConfig)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_isValidBufferedReadConfig_ValidScenarios(t *testing.T) {
	var testCases = []struct {
		testName string
		read     ReadConfig
	}{
		{"valid_config_1", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      -1,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   1,
		}},
		{"valid_config_2", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      10,
			MaxBlocksPerHandle:   -1,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   4,
		}},
		{"valid_config_3", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   true,
			GlobalMaxBlocks:      10,
			MaxBlocksPerHandle:   5,
			StartBlocksPerHandle: 1,
			MinBlocksPerHandle:   5,
		}},
		{"valid_config_4", ReadConfig{
			BlockSizeMb:          16,
			EnableBufferedRead:   false,
			GlobalMaxBlocks:      10,
			MaxBlocksPerHandle:   5,
			StartBlocksPerHandle: 10,
			MinBlocksPerHandle:   3,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.NoError(t, isValidBufferedReadConfig(&tc.read))
		})
	}
}

func Test_isValidWriteStreamingConfig_SuccessScenarios(t *testing.T) {
	var testCases = []struct {
		testName    string
		writeConfig WriteConfig
	}{
		{"streaming_writes_disabled", WriteConfig{
			BlockSizeMb:           -1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: false,
			GlobalMaxBlocks:       -10,
			MaxBlocksPerFile:      -10,
		}},
		{"valid_write_config_1", WriteConfig{
			BlockSizeMb:           1,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       -1,
			MaxBlocksPerFile:      -1,
		}},
		{"valid_write_config_2", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      -1,
		}},
		{"valid_write_config_3", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      20,
		}},
		{"valid_write_config_4", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       40,
			MaxBlocksPerFile:      20,
		}},
		{"0_global_max_blocks", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       0,
			MaxBlocksPerFile:      -1,
		}},
		{"1_global_max_blocks", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       1,
			MaxBlocksPerFile:      -1,
		}},
		{"1_max_blocks_per_file", WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      1,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.NoError(t, isValidWriteStreamingConfig(&tc.writeConfig))
		})
	}
}

func validConfig(t *testing.T) Config {
	return Config{
		Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
		FileCache: validFileCacheConfig(t),
		GcsConnection: GcsConnectionConfig{
			CustomEndpoint:       "https://bing.com/search?q=dotnet",
			SequentialReadSizeMb: 200,
		},
		MetadataCache: MetadataCacheConfig{
			ExperimentalMetadataPrefetchOnMount: "disabled",
		},
		Metrics: MetricsConfig{
			Workers:    1,
			BufferSize: 1,
		},
		Mrd: MrdConfig{
			PoolSize: 4,
		},
	}
}

func TestValidateMetrics(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		metricsConfig MetricsConfig
		wantErr       bool
	}{

		{
			name: "both_cloud_metrics_export_interval_secs_and_stackdriver_specified",
			metricsConfig: MetricsConfig{
				CloudMetricsExportIntervalSecs: 20,
				StackdriverExportInterval:      time.Duration(30) * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "neg_cloud_metrics_export_interval",
			metricsConfig: MetricsConfig{
				CloudMetricsExportIntervalSecs: -1,
				Workers:                        10,
				BufferSize:                     100,
			},
			wantErr: false,
		},
		{
			name: "neg_stackdriver_export_interval",
			metricsConfig: MetricsConfig{
				StackdriverExportInterval: -1 * time.Second,
				Workers:                   10,
				BufferSize:                100,
			},
			wantErr: false,
		},
		{
			name: "neg_cloud_metrics_export_interval",
			metricsConfig: MetricsConfig{
				CloudMetricsExportIntervalSecs: 10,
				Workers:                        10,
				BufferSize:                     100,
			},
			wantErr: false,
		},
		{
			name: "too_high_prom_port",
			metricsConfig: MetricsConfig{
				PrometheusPort: 100000,
			},
			wantErr: true,
		},
		{
			name: "valid_prom_port",
			metricsConfig: MetricsConfig{
				PrometheusPort: 5550,
				Workers:        10,
				BufferSize:     100,
			},
			wantErr: false,
		},
		{
			name: "prom_disabled_0",
			metricsConfig: MetricsConfig{
				PrometheusPort: 0,
				Workers:        10,
				BufferSize:     100,
			},
			wantErr: false,
		},
		{
			name: "prom_disabled_less_than_0",
			metricsConfig: MetricsConfig{
				PrometheusPort: -21,
				Workers:        10,
				BufferSize:     100,
			},
			wantErr: false,
		},
		{
			name: "metrics_workers_less_than_1",
			metricsConfig: MetricsConfig{
				Workers: 0,
			},
			wantErr: true,
		},
		{
			name: "metrics_buffer_size_less_than_1",
			metricsConfig: MetricsConfig{
				BufferSize: 0,
			},
			wantErr: true,
		},
		{
			name: "valid_workers_and_buffer_size",
			metricsConfig: MetricsConfig{
				Workers:    10,
				BufferSize: 100,
			},
			wantErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := validConfig(t)
			c.Metrics = tc.metricsConfig

			err := ValidateConfig(viper.New(), &c)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMonitoring(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		MonitoringConfig MonitoringConfig
		wantErr          bool
	}{

		{
			name: "",
			MonitoringConfig: MonitoringConfig{
				ExperimentalTracingMode: []string{"stdout", "  gcptrace"},
			},
			wantErr: false,
		},
		{
			name: "",
			MonitoringConfig: MonitoringConfig{
				ExperimentalTracingMode: []string{"STDout", "  Gcptrace"},
			},
			wantErr: false,
		},
		{
			name: "",
			MonitoringConfig: MonitoringConfig{
				ExperimentalTracingMode: []string{"stdout", "  random_export"},
			},
			wantErr: true,
		},
		{
			name: "",
			MonitoringConfig: MonitoringConfig{
				ExperimentalTracingMode:          []string{},
				ExperimentalTracingProjectId:     "test-gcloud-project",
				ExperimentalTracingSamplingRatio: 0.3,
			},
			wantErr: false,
		},
		{
			name: "",
			MonitoringConfig: MonitoringConfig{
				ExperimentalTracingMode:          []string{},
				ExperimentalTracingProjectId:     "test-gcloud-project",
				ExperimentalTracingSamplingRatio: 1.4,
			},
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := validConfig(t)
			c.Monitoring = tc.MonitoringConfig

			err := ValidateConfig(viper.New(), &c)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLogSeverityRanks(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		logSev         string
		wantLogSevRank int
		wantLogSev     LogSeverity
		wantErr        bool
	}{
		{
			logSev:         "off",
			wantLogSevRank: 5,
			wantLogSev:     OffLogSeverity,
		},
		{
			logSev:         "error",
			wantLogSevRank: 4,
			wantLogSev:     ErrorLogSeverity,
		},
		{
			logSev:         "warning",
			wantLogSevRank: 3,
			wantLogSev:     WarningLogSeverity,
		},
		{
			logSev:         "info",
			wantLogSevRank: 2,
			wantLogSev:     InfoLogSeverity,
		},
		{
			logSev:         "debug",
			wantLogSevRank: 1,
			wantLogSev:     DebugLogSeverity,
		},
		{
			logSev:         "trace",
			wantLogSevRank: 0,
			wantLogSev:     TraceLogSeverity,
		},
		{
			logSev:         "invalid",
			wantLogSevRank: -1,
			wantErr:        true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.logSev, func(t *testing.T) {
			t.Parallel()
			level := LogSeverity(tc.logSev)

			err := level.UnmarshalText([]byte(tc.logSev))

			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, -1, level.Rank())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantLogSev.Rank(), level.Rank())
			}
		})
	}
}

func TestValidateProfile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{
			name:    "empty_profile",
			profile: "",
			wantErr: false,
		}, {
			name:    "profile_training",
			profile: ProfileAIMLTraining,
			wantErr: false,
		}, {
			name:    "profile_serving",
			profile: ProfileAIMLServing,
			wantErr: false,
		}, {
			name:    "profile_checkpointing",
			profile: ProfileAIMLCheckpointing,
			wantErr: false,
		}, {
			name:    "unsupported_profile",
			profile: "unsupported-profile",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := validConfig(t)
			c.Profile = tc.profile

			err := ValidateConfig(viper.New(), &c)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

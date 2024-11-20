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
			actualErr := ValidateConfig(&mockIsSet{}, tc.config)

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, ValidateConfig(&mockIsSet{}, tc.config))
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
			BlockSizeMb:                       0,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   -1,
			MaxBlocksPerFile:                  -1,
		}},
		{"negative_block_size", WriteConfig{
			BlockSizeMb:                       -1,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   -1,
			MaxBlocksPerFile:                  -1,
		}},
		{"-2_global_max_blocks", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   -2,
			MaxBlocksPerFile:                  -1,
		}},
		{"0_global_max_blocks", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   0,
			MaxBlocksPerFile:                  -1,
		}},
		{"1_global_max_blocks", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   1,
			MaxBlocksPerFile:                  -1,
		}},
		{"-2_max_blocks_per_file", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  -2,
		}},
		{"0_max_blocks_per_file", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  0,
		}},
		{"1_max_blocks_per_file", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  1,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Error(t, isValidWriteStreamingConfig(&tc.writeConfig))
		})
	}
}

func Test_isValidWriteStreamingConfig_SuccessScenarios(t *testing.T) {
	var testCases = []struct {
		testName    string
		writeConfig WriteConfig
	}{
		{"streaming_writes_disabled", WriteConfig{
			BlockSizeMb:                       -1,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: false,
			GlobalMaxBlocks:                   -10,
			MaxBlocksPerFile:                  -10,
		}},
		{"valid_write_config_1", WriteConfig{
			BlockSizeMb:                       1,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   -1,
			MaxBlocksPerFile:                  -1,
		}},
		{"valid_write_config_2", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  -1,
		}},
		{"valid_write_config_3", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  20,
		}},
		{"valid_write_config_4", WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   40,
			MaxBlocksPerFile:                  20,
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
			},
			wantErr: false,
		},
		{
			name: "neg_stackdriver_export_interval",
			metricsConfig: MetricsConfig{
				StackdriverExportInterval: -1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "neg_cloud_metrics_export_interval",
			metricsConfig: MetricsConfig{
				CloudMetricsExportIntervalSecs: 10,
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
			},
			wantErr: false,
		},
		{
			name: "prom_disabled_0",
			metricsConfig: MetricsConfig{
				PrometheusPort: 0,
			},
			wantErr: false,
		},
		{
			name: "prom_disabled_less_than_0",
			metricsConfig: MetricsConfig{
				PrometheusPort: -21,
			},
			wantErr: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := validConfig(t)
			c.Metrics = tc.metricsConfig

			err := ValidateConfig(&mockIsSet{}, &c)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

package cmd

import (
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConfigObject(t *testing.T, args []string) (*cfg.Config, error) {
	t.Helper()
	var c *cfg.Config
	cmd, err := NewRootCmd(func(config *cfg.Config, _, _ string) error {
		c = config
		return nil
	})
	require.Nil(t, err)
	cmdArgs := append([]string{"gcsfuse"}, args...)
	cmdArgs = append(cmdArgs, "a")
	cmd.SetArgs(convertToPosixArgs(cmdArgs, cmd))
	if err = cmd.Execute(); err != nil {
		return nil, err
	}

	return c, nil
}

func getConfigObjectWithConfigFile(t *testing.T, configFilePath string) (*cfg.Config, error) {
	t.Helper()
	return getConfigObject(t, []string{fmt.Sprintf("--config-file=%s", configFilePath)})
}

func defaultFileCacheConfig(t *testing.T) cfg.FileCacheConfig {
	t.Helper()
	return cfg.FileCacheConfig{
		CacheFileForRangeRead:    false,
		DownloadChunkSizeMb:      50,
		EnableCrc:                false,
		EnableParallelDownloads:  false,
		MaxParallelDownloads:     int64(max(16, 2*runtime.NumCPU())),
		MaxSizeMb:                -1,
		ParallelDownloadsPerFile: 16,
		WriteBufferSize:          4 * 1024 * 1024,
		EnableODirect:            false,
	}
}

func TestValidateConfigFile(t *testing.T) {
	testCases := []struct {
		name       string
		configFile string
		wantErr    bool
	}{
		{
			name:       "empty file",
			configFile: "testdata/empty_file.yaml",
			wantErr:    false,
		},
		{
			name:       "non-existent file",
			configFile: "testdata/nofile.yml",
			wantErr:    true,
		},
		{
			name:       "invalid config file",
			configFile: "testdata/invalid_config.yaml",
			wantErr:    true,
		},
		{
			name:       "logrotate with 0 backup file count",
			configFile: "testdata/valid_config_with_0_backup-file-count.yaml",
			wantErr:    false,
		},
		{
			name:       "unexpected field in config",
			configFile: "testdata/invalid_unexpectedfield_config.yaml",
			wantErr:    true,
		},
		{
			name:       "valid config",
			configFile: "testdata/valid_config.yaml",
			wantErr:    false,
		},
		{
			name:       "invalid log config",
			configFile: "testdata/invalid_log_config.yaml",
			wantErr:    true,
		},
		{
			name:       "invalid_logrotate_1",
			configFile: "testdata/invalid_log_rotate_config_1.yaml",
			wantErr:    true,
		},
		{
			name:       "invalid_logrotate_2",
			configFile: "testdata/invalid_log_rotate_config_2.yaml",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCliFlag(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "empty file",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "logrotate with 0 backup file count",
			args:    []string{"--log-rotate-backup-file-count=0"},
			wantErr: false,
		},
		{
			name:    "invalid log severity",
			args:    []string{"--log-severity=critical"},
			wantErr: true,
		},
		{
			name:    "invalid log-rotate-max-log-file-size-mb",
			args:    []string{"--log-rotate-max-log-file-size-mb=-1"},
			wantErr: true,
		},
		{
			name:    "invalid log-rotate-backup-file-count",
			args:    []string{"--log-rotate-backup-file-count=-1"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConfigObject(t, tc.args)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigFile_WriteConfig(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			name:       "Empty config file [default values].",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				Write: cfg.WriteConfig{
					CreateEmptyFile:                   false,
					BlockSizeMb:                       64,
					ExperimentalEnableStreamingWrites: false,
					GlobalMaxBlocks:                   math.MaxInt64,
					MaxBlocksPerFile:                  math.MaxInt64},
			},
		},
		{
			name:       "Valid config file.",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				Write: cfg.WriteConfig{
					CreateEmptyFile:                   false, // changed due to enabled streaming writes.
					BlockSizeMb:                       10,
					ExperimentalEnableStreamingWrites: true,
					GlobalMaxBlocks:                   20,
					MaxBlocksPerFile:                  2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.Write, gotConfig.Write)
			}
		})
	}
}

func TestValidateConfigFile_InvalidConfigThrowsError(t *testing.T) {
	testCases := []struct {
		name       string
		configFile string
	}{
		{
			name:       "Invalid parallel downloads per file.",
			configFile: "testdata/file_cache_config/invalid_parallel_downloads_per_file.yaml",
		},
		{
			name:       "Invalid download chunk size mb.",
			configFile: "testdata/file_cache_config/invalid_download_chunk_size_mb.yaml",
		},
		{
			name:       "Invalid max size mb.",
			configFile: "testdata/file_cache_config/invalid_max_size_mb.yaml",
		},
		{
			name:       "Invalid max parallel downloads.",
			configFile: "testdata/file_cache_config/invalid_max_parallel_downloads.yaml",
		},
		{
			name:       "Invalid zero max parallel downloads",
			configFile: "testdata/file_cache_config/invalid_zero_max_parallel_downloads.yaml",
		},
		{
			name:       "Invalid value of anonymous access.",
			configFile: "testdata/gcs_auth/invalid_anonymous_access.yaml",
		},
		{
			name:       "Invalid zero write buffer size",
			configFile: "testdata/file_cache_config/invalid_zero_write_buffer_size.yaml",
		},
		{
			name:       "Invalid write buffer size",
			configFile: "testdata/file_cache_config/invalid_write_buffer_size.yaml",
		},
		{
			name:       "invalid_ignore_interrupts",
			configFile: "testdata/file_system_config/invalid_ignore_interrupts.yaml",
		},
		{
			name:       "invalid_disable_parallel_dirops",
			configFile: "testdata/file_system_config/invalid_disable_parallel_dirops.yaml",
		},
		{
			name:       "negative_kernel_list_cache_TTL",
			configFile: "testdata/file_system_config/invalid_kernel_list_cache_ttl.yaml",
		},
		{
			name:       "unsupported_large_kernel_list_cache_TTL",
			configFile: "testdata/file_system_config/unsupported_large_kernel_list_cache_ttl.yaml",
		},
		{
			name:       "negative_stat_cache_size",
			configFile: "testdata/metadata_cache/metadata_cache_config_invalid_stat-cache-max-size-mb.yaml",
		},
		{
			name:       "negative_ttl_secs",
			configFile: "testdata/metadata_cache/metadata_cache_config_invalid_ttl.yaml",
		},
		{
			name:       "negative_type_cache_size",
			configFile: "testdata/metadata_cache/metadata_cache_config_invalid_type-cache-max-size-mb.yaml",
		},
		{
			name:       "stat_cache_size_too_high",
			configFile: "testdata/metadata_cache/metadata_cache_config_stat-cache-max-size-mb_too_high.yaml",
		},
		{
			name:       "metadata_cache_size_too_high",
			configFile: "testdata/metadata_cache/metadata_cache_config_ttl_too_high.yaml",
		},
		{
			name:       "write_block_size_0",
			configFile: "testdata/write_config/invalid_write_config_due_to_0_block_size.yaml",
		},
		{
			name:       "small_global_max_blocks",
			configFile: "testdata/write_config/invalid_write_config_due_to_small_global_max_blocks.yaml",
		},
		{
			name:       "small_max_blocks_per_file",
			configFile: "testdata/write_config/invalid_write_config_due_to_small_max_blocks_per_file.yaml",
		},
		{
			name:       "negative req_increase_rate",
			configFile: "testdata/gcs_retries/read_stall/invalid_req_increase_rate_negative.yaml",
		},
		{
			name:       "zero req_increase_rate",
			configFile: "testdata/gcs_retries/read_stall/invalid_req_increase_rate_zero.yaml",
		},
		{
			name:       "large req_target_percentile",
			configFile: "testdata/gcs_retries/read_stall/invalid_req_target_percentile_large.yaml",
		},
		{
			name:       "negative req_target_percentile",
			configFile: "testdata/gcs_retries/read_stall/invalid_req_target_percentile_negative.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConfigObjectWithConfigFile(t, tc.configFile)

			assert.Error(t, err)
		})
	}
}

func TestValidateConfigFile_FileCacheConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			name:       "Empty config file [default values].",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				FileCache: defaultFileCacheConfig(t),
			},
		},
		{
			name:       "Valid config file.",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:    true,
					DownloadChunkSizeMb:      300,
					EnableCrc:                true,
					EnableParallelDownloads:  true,
					MaxParallelDownloads:     200,
					MaxSizeMb:                40,
					ParallelDownloadsPerFile: 10,
					WriteBufferSize:          8192,
					EnableODirect:            true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.FileCache, gotConfig.FileCache)
			}
		})
	}
}

func TestValidateConfigFile_GCSAuthConfigSuccessful(t *testing.T) {
	hd, err := os.UserHomeDir()
	require.Nil(t, err)
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			name:       "Empty config file [default values].",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess:   false,
					KeyFile:           "",
					ReuseTokenFromUrl: true,
					TokenUrl:          "",
				},
			},
		},
		{
			name:       "Valid config file.",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess:   true,
					KeyFile:           cfg.ResolvedPath(path.Join(hd, "key.file")),
					ReuseTokenFromUrl: false,
					TokenUrl:          "www.abc.com",
				},
			},
		},
		{
			name:       "Valid config file with GCS Auth unset",
			configFile: "testdata/gcs_auth/unset_anonymous_access.yaml",
			expectedConfig: &cfg.Config{
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess:   false,
					KeyFile:           "",
					ReuseTokenFromUrl: true,
					TokenUrl:          "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.GcsAuth, gotConfig.GcsAuth)
			}
		})
	}
}

func TestValidateConfigFile_GCSConnectionConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			name:       "Empty config file [default values].",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				GcsConnection: cfg.GcsConnectionConfig{
					BillingProject:             "",
					ClientProtocol:             "http1",
					CustomEndpoint:             "",
					ExperimentalEnableJsonRead: false,
					GrpcConnPoolSize:           1,
					HttpClientTimeout:          0,
					LimitBytesPerSec:           -1,
					LimitOpsPerSec:             -1,
					MaxConnsPerHost:            0,
					MaxIdleConnsPerHost:        100,
					SequentialReadSizeMb:       200,
				},
			},
		},
		{
			name:       "Valid config file.",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				GcsConnection: cfg.GcsConnectionConfig{
					BillingProject:             "abc",
					ClientProtocol:             "http2",
					CustomEndpoint:             "www.abc.com",
					ExperimentalEnableJsonRead: true,
					GrpcConnPoolSize:           200,
					HttpClientTimeout:          400 * time.Second,
					LimitBytesPerSec:           20,
					LimitOpsPerSec:             30,
					MaxConnsPerHost:            400,
					MaxIdleConnsPerHost:        20,
					SequentialReadSizeMb:       450,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.GcsConnection, gotConfig.GcsConnection)
			}
		})
	}
}

func TestValidateConfigFile_FileSystemConfigSuccessful(t *testing.T) {
	hd, err := os.UserHomeDir()
	require.NoError(t, err)
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			// Test default values.
			name:       "empty_config_file",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                0755,
					DisableParallelDirops:  false,
					FileMode:               0644,
					FuseOptions:            []string{},
					Gid:                    -1,
					IgnoreInterrupts:       true,
					KernelListCacheTtlSecs: 0,
					RenameDirLimit:         0,
					TempDir:                "",
					Uid:                    -1,
				},
			},
		},
		{
			name:       "file_system_config_unset",
			configFile: "testdata/file_system_config/unset_file_system_config.yaml",
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                0755,
					DisableParallelDirops:  false,
					FileMode:               0644,
					FuseOptions:            []string{},
					Gid:                    -1,
					IgnoreInterrupts:       true,
					KernelListCacheTtlSecs: 0,
					RenameDirLimit:         0,
					TempDir:                "",
					Uid:                    -1,
				},
			},
		},
		{
			name:       "valid_config_file",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                0777,
					DisableParallelDirops:  true,
					FileMode:               0666,
					FuseOptions:            []string{"ro"},
					Gid:                    7,
					IgnoreInterrupts:       false,
					KernelListCacheTtlSecs: 300,
					RenameDirLimit:         10,
					TempDir:                cfg.ResolvedPath(path.Join(hd, "temp")),
					Uid:                    8,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.FileSystem, gotConfig.FileSystem)
			}
		})
	}
}

func TestValidateConfigFile_ListConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			// Test default values.
			name:       "empty_config_file",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				List: cfg.ListConfig{EnableEmptyManagedFolders: false},
			},
		},
		{
			name:       "valid_config_file",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				List: cfg.ListConfig{EnableEmptyManagedFolders: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.List, gotConfig.List)
			}
		})
	}
}

func TestValidateConfigFile_EnableHNSConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			// Test default values.
			name:       "empty_config_file",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				EnableHns: true,
			},
		},
		{
			name:       "valid_config_file",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				EnableHns: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.EnableHns, gotConfig.EnableHns)
			}
		})
	}
}

func TestValidateConfigFile_MetadataCacheConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			// Test default values.
			name:       "empty_config_file",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         20460,
					DeprecatedStatCacheTtl:              60 * time.Second,
					DeprecatedTypeCacheTtl:              60 * time.Second,
					EnableNonexistentTypeCache:          false,
					ExperimentalMetadataPrefetchOnMount: "disabled",
					StatCacheMaxSizeMb:                  32,
					TtlSecs:                             60,
					TypeCacheMaxSizeMb:                  4,
				},
			},
		},
		{
			name:       "valid_config_file",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         200,
					DeprecatedStatCacheTtl:              30 * time.Second,
					DeprecatedTypeCacheTtl:              20 * time.Second,
					EnableNonexistentTypeCache:          true,
					ExperimentalMetadataPrefetchOnMount: "sync",
					StatCacheMaxSizeMb:                  40,
					TtlSecs:                             100,
					TypeCacheMaxSizeMb:                  10,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.MetadataCache, gotConfig.MetadataCache)
			}
		})
	}
}

func TestValidateConfigFile_ReadStallConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			// Test default values.
			name:       "empty_config_file",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				GcsRetries: cfg.GcsRetriesConfig{
					ReadStall: cfg.ReadStallGcsRetriesConfig{
						Enable:              false,
						MinReqTimeout:       500 * time.Millisecond,
						MaxReqTimeout:       1200 * time.Second,
						InitialReqTimeout:   20 * time.Second,
						ReqTargetPercentile: 0.99,
						ReqIncreaseRate:     15,
					},
				},
			},
		},
		{
			name:       "valid_config_file",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				GcsRetries: cfg.GcsRetriesConfig{
					ReadStall: cfg.ReadStallGcsRetriesConfig{
						Enable:              true,
						MinReqTimeout:       10 * time.Second,
						MaxReqTimeout:       200 * time.Second,
						InitialReqTimeout:   20 * time.Second,
						ReqTargetPercentile: 0.99,
						ReqIncreaseRate:     15,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.GcsRetries.ReadStall, gotConfig.GcsRetries.ReadStall)
			}
		})
	}
}

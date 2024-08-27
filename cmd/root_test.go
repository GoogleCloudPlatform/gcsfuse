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
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMaxParallelDownloads(t *testing.T) {
	var actual *cfg.Config
	cmd, err := NewRootCmd(func(c *cfg.Config, _, _ string) error {
		actual = c
		return nil
	})
	require.Nil(t, err)
	cmd.SetArgs([]string{"abc", "pqr"})

	if assert.Nil(t, cmd.Execute()) {
		assert.LessOrEqual(t, int64(16), actual.FileCache.MaxParallelDownloads)
	}
}

func TestCobraArgsNumInRange(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Too many args",
			args:        []string{"gcsfuse", "abc", "pqr", "xyz"},
			expectError: true,
		},
		{
			name:        "Too few args",
			args:        []string{"gcsfuse"},
			expectError: true,
		},
		{
			name:        "Two args is okay",
			args:        []string{"gcsfuse", "abc"},
			expectError: false,
		},
		{
			name:        "Three args is okay",
			args:        []string{"gcsfuse", "abc", "pqr"},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(*cfg.Config, string, string) error { return nil })
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestArgsParsing_MountPoint(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)
	hd, err := os.UserHomeDir()
	require.Nil(t, err)
	tests := []struct {
		name               string
		args               []string
		expectedBucket     string
		expectedMountpoint string
	}{
		{
			name:               "Both bucket and mountpoint specified.",
			args:               []string{"gcsfuse", "abc", "pqr"},
			expectedBucket:     "abc",
			expectedMountpoint: path.Join(wd, "pqr"),
		},
		{
			name:               "Only mountpoint specified",
			args:               []string{"gcsfuse", "pqr"},
			expectedBucket:     "",
			expectedMountpoint: path.Join(wd, "pqr"),
		},
		{
			name:               "Absolute path for mountpoint specified",
			args:               []string{"gcsfuse", "/pqr"},
			expectedBucket:     "",
			expectedMountpoint: "/pqr",
		},
		{
			name:               "Relative path from user's home specified as mountpoint",
			args:               []string{"gcsfuse", "~/pqr"},
			expectedBucket:     "",
			expectedMountpoint: path.Join(hd, "pqr"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bucketName, mountPoint string
			cmd, err := NewRootCmd(func(_ *cfg.Config, b string, m string) error {
				bucketName = b
				mountPoint = m
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedBucket, bucketName)
				assert.Equal(t, tc.expectedMountpoint, mountPoint)
			}
		})
	}
}

func TestArgsParsing_MountOptions(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		expectedMountOptions []string
	}{
		{
			name:                 "Multiple mount options specified with multiple -o flags.",
			args:                 []string{"gcsfuse", "--o", "rw,nodev", "--o", "user=jacobsa,noauto", "abc", "pqr"},
			expectedMountOptions: []string{"rw", "nodev", "user=jacobsa", "noauto"},
		},
		{
			name:                 "Only one mount option specified.",
			args:                 []string{"gcsfuse", "--o", "rw", "abc", "pqr"},
			expectedMountOptions: []string{"rw"},
		},
		{
			name:                 "Multiple mount option specified with single flag.",
			args:                 []string{"gcsfuse", "--o", "rw,nodev", "abc", "pqr"},
			expectedMountOptions: []string{"rw", "nodev"},
		},
		{
			name:                 "Multiple mount options specified with single -o flags.",
			args:                 []string{"gcsfuse", "--o", "rw,nodev,user=jacobsa,noauto", "abc", "pqr"},
			expectedMountOptions: []string{"rw", "nodev", "user=jacobsa", "noauto"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var mountOptions []string
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				mountOptions = cfg.FileSystem.FuseOptions
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedMountOptions, mountOptions)
			}
		})
	}
}

func TestArgsParsing_CreateEmptyFileFlag(t *testing.T) {
	tests := []struct {
		name                    string
		args                    []string
		expectedCreateEmptyFile bool
	}{
		{
			name:                    "Test create-empty-file flag true.",
			args:                    []string{"gcsfuse", "--create-empty-file=true", "abc", "pqr"},
			expectedCreateEmptyFile: true,
		},
		{
			name:                    "Test create-empty-file flag false.",
			args:                    []string{"gcsfuse", "--create-empty-file=false", "abc", "pqr"},
			expectedCreateEmptyFile: false,
		},
		{
			name:                    "Test default create-empty-file flag.",
			args:                    []string{"gcsfuse", "abc", "pqr"},
			expectedCreateEmptyFile: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var createEmptyFile bool
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				createEmptyFile = cfg.Write.CreateEmptyFile
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedCreateEmptyFile, createEmptyFile)
			}
		})
	}
}

func TestArgsParsing_FileCacheFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "Test file cache flags.",
			args: []string{"gcsfuse", "--cache-file-for-range-read", "--download-chunk-size-mb=20", "--enable-crc", "--enable-parallel-downloads", "--max-parallel-downloads=40", "--file-cache-max-size-mb=100", "--parallel-downloads-per-file=2", "--disable-o-direct=false", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:    true,
					DownloadChunkSizeMb:      20,
					EnableCrc:                true,
					EnableParallelDownloads:  true,
					MaxParallelDownloads:     40,
					MaxSizeMb:                100,
					ParallelDownloadsPerFile: 2,
					WriteBufferSize:          4 * 1024 * 1024,
					DisableODirect:           false,
				},
			},
		},
		{
			name: "Test default file cache flags.",
			args: []string{"gcsfuse", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:    false,
					DownloadChunkSizeMb:      50,
					EnableCrc:                false,
					EnableParallelDownloads:  false,
					MaxParallelDownloads:     int64(max(16, 2*runtime.NumCPU())),
					MaxSizeMb:                -1,
					ParallelDownloadsPerFile: 16,
					WriteBufferSize:          4 * 1024 * 1024,
					DisableODirect:           true,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.FileCache, gotConfig.FileCache)
			}
		})
	}
}

func TestArgParsing_ExperimentalMetadataPrefetchFlag(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedValue string
	}{
		{
			name:          "set to sync",
			args:          []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount=sync", "abc", "pqr"},
			expectedValue: "sync",
		},
		{
			name:          "set to async",
			args:          []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount=async", "abc", "pqr"},
			expectedValue: "async",
		},
		{
			name:          "set to async, space-separated",
			args:          []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount", "async", "abc", "pqr"},
			expectedValue: "async",
		},
		{
			name:          "set to disabled",
			args:          []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount=disabled", "abc", "pqr"},
			expectedValue: "disabled",
		},
		{
			name:          "Test default.",
			args:          []string{"gcsfuse", "abc", "pqr"},
			expectedValue: "disabled",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var experimentalMetadataPrefetch string
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				experimentalMetadataPrefetch = cfg.MetadataCache.ExperimentalMetadataPrefetchOnMount
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedValue, experimentalMetadataPrefetch)
			}
		})
	}
}

func TestArgParsing_ExperimentalMetadataPrefetchFlag_Failed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Test invalid value 1",
			args: []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount=foo", "abc", "pqr"},
		},
		{
			name: "Test invalid value 2",
			args: []string{"gcsfuse", "--experimental-metadata-prefetch-on-mount=123", "abc", "pqr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			assert.Error(t, err)
		})
	}
}

func TestArgsParsing_GCSAuthFlags(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "Test gcs auth flags.",
			args: []string{"gcsfuse", "--anonymous-access", "--key-file=key.file", "--reuse-token-from-url", "--token-url=www.abc.com", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				GcsAuth: cfg.GcsAuthConfig{
					AnonymousAccess:   true,
					KeyFile:           cfg.ResolvedPath(path.Join(wd, "key.file")),
					ReuseTokenFromUrl: true,
					TokenUrl:          "www.abc.com",
				},
			},
		},
		{
			name: "Test default gcs auth flags.",
			args: []string{"gcsfuse", "abc", "pqr"},
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.GcsAuth, gotConfig.GcsAuth)
			}
		})
	}
}

func TestArgsParsing_GCSAuthFlagsThrowsError(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "Invalid value for anonymous-access flag",
			args: []string{"gcsfuse", "--anonymous-access=a", "abc", "pqr"},
		},
		{
			name: "Invalid value for reuse-token-from-url flag",
			args: []string{"gcsfuse", "--reuse-token-from-url", "b", "abc", "pqr"},
		},
		{
			name: "Invalid value for token-url flag",
			args: []string{"gcsfuse", "--token-url=a_b://abc", "abc", "pqr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			assert.Error(t, cmd.Execute())
		})
	}
}

func TestArgsParsing_GCSConnectionFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "Test gcs connection flags.",
			args: []string{"gcsfuse", "--billing-project=abc", "--client-protocol=http2", "--custom-endpoint=www.abc.com", "--experimental-enable-json-read", "--experimental-grpc-conn-pool-size=20", "--http-client-timeout=20s", "--limit-bytes-per-sec=30", "--limit-ops-per-sec=10", "--max-conns-per-host=1000", "--max-idle-conns-per-host=20", "--sequential-read-size-mb=70", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				GcsConnection: cfg.GcsConnectionConfig{
					BillingProject:             "abc",
					ClientProtocol:             "http2",
					CustomEndpoint:             "www.abc.com",
					ExperimentalEnableJsonRead: true,
					GrpcConnPoolSize:           20,
					HttpClientTimeout:          20 * time.Second,
					LimitBytesPerSec:           30,
					LimitOpsPerSec:             10,
					MaxConnsPerHost:            1000,
					MaxIdleConnsPerHost:        20,
					SequentialReadSizeMb:       70,
				},
			},
		},
		{
			name: "Test default gcs connection flags.",
			args: []string{"gcsfuse", "abc", "pqr"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.GcsConnection, gotConfig.GcsConnection)
			}
		})
	}
}
func TestArgsParsing_GCSConnectionFlagsThrowsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Invalid value for sequential read size flag 1",
			args: []string{"gcsfuse", "--sequential-read-size-mb=2040", "abc", "pqr"},
		},
		{
			name: "Invalid value for sequential read size flag 2",
			args: []string{"gcsfuse", "--sequential-read-size-mb=0", "abc", "pqr"},
		},
		{
			name: "Invalid value for client-protocol flag",
			args: []string{"gcsfuse", "--client-protocol=http3", "abc", "pqr"},
		},
		{
			name: "Invalid value for custom-endpoint flag",
			args: []string{"gcsfuse", "--custom-endpoint=a_b://abc", "abc", "pqr"},
		},
		{
			name: "Invalid value for http-client-timeout flag",
			args: []string{"gcsfuse", "--http-client-timeout=200", "abc", "pqr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			assert.Error(t, cmd.Execute())
		})
	}
}

func TestArgsParsing_FileSystemFlags(t *testing.T) {
	hd, err := os.UserHomeDir()
	require.NoError(t, err)
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "normal",
			args: []string{"gcsfuse", "--dir-mode=0777", "--disable-parallel-dirops", "--file-mode=0666", "--o", "ro", "--gid=7", "--ignore-interrupts=false", "--kernel-list-cache-ttl-secs=300", "--rename-dir-limit=10", "--temp-dir=~/temp", "--uid=8", "abc", "pqr"},
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
		{
			name: "mode_flags_without_0_prefix",
			args: []string{"gcsfuse", "--dir-mode=777", "--file-mode=666", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                0777,
					DisableParallelDirops:  false,
					FileMode:               0666,
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
			name: "default",
			args: []string{"gcsfuse", "abc", "pqr"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.FileSystem, gotConfig.FileSystem)
			}
		})
	}
}

func TestArgsParsing_FileSystemFlagsThrowsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "invalid_dir_mode_999",
			args: []string{"gcsfuse", "--dir-mode=999", "abc", "pqr"},
		},
		{
			name: "invalid_dir_mode_0999",
			args: []string{"gcsfuse", "--dir-mode=0999", "abc", "pqr"},
		},
		{
			name: "invalid_file_mode_888",
			args: []string{"gcsfuse", "--file-mode=888", "abc", "pqr"},
		},
		{
			name: "invalid_file_mode_0888",
			args: []string{"gcsfuse", "--file-mode=0888", "abc", "pqr"},
		},
		{
			name: "invalid_disable_parallel_dirops",
			args: []string{"gcsfuse", "--disable-parallel-dirops=abc", "abc", "pqr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			assert.Error(t, cmd.Execute())
		})
	}
}

func TestArgsParsing_ListFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "normal",
			args: []string{"gcsfuse", "--enable-empty-managed-folders", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				List: cfg.ListConfig{EnableEmptyManagedFolders: true},
			},
		},
		{
			name: "default",
			args: []string{"gcsfuse", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				List: cfg.ListConfig{EnableEmptyManagedFolders: false},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.List, gotConfig.List)
			}
		})
	}
}

func TestArgsParsing_EnableHNSFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedEnableHNS bool
	}{
		{
			name:              "normal",
			args:              []string{"gcsfuse", "--enable-hns=false", "abc", "pqr"},
			expectedEnableHNS: false,
		},
		{
			name:              "default",
			args:              []string{"gcsfuse", "abc", "pqr"},
			expectedEnableHNS: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotEnableHNS bool
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotEnableHNS = cfg.EnableHns
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedEnableHNS, gotEnableHNS)
			}
		})
	}
}

func TestArgsParsing_MetadataCacheFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "normal",
			args: []string{"gcsfuse", "--stat-cache-capacity=2000", "--stat-cache-ttl=2m", "--type-cache-ttl=1m20s", "--enable-nonexistent-type-cache", "--experimental-metadata-prefetch-on-mount=async", "--stat-cache-max-size-mb=15", "--metadata-cache-ttl-secs=25", "--type-cache-max-size-mb=30", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         2000,
					DeprecatedStatCacheTtl:              2 * time.Minute,
					DeprecatedTypeCacheTtl:              80 * time.Second,
					EnableNonexistentTypeCache:          true,
					ExperimentalMetadataPrefetchOnMount: "async",
					StatCacheMaxSizeMb:                  15,
					TtlSecs:                             25,
					TypeCacheMaxSizeMb:                  30,
				},
			},
		},
		{
			name: "default",
			args: []string{"gcsfuse", "abc", "pqr"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.MetadataCache, gotConfig.MetadataCache)
			}
		})
	}
}

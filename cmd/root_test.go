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
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMaxParallelDownloads(t *testing.T) {
	var actual *cfg.Config
	cmd, err := newRootCmd(func(c *cfg.Config, _, _ string) error {
		actual = c
		return nil
	})
	require.Nil(t, err)
	cmd.SetArgs(convertToPosixArgs([]string{"abc", "pqr"}, cmd))

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
			cmd, err := newRootCmd(func(*cfg.Config, string, string) error { return nil })
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(_ *cfg.Config, b string, m string) error {
				bucketName = b
				mountPoint = m
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				mountOptions = cfg.FileSystem.FuseOptions
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedMountOptions, mountOptions)
			}
		})
	}
}

// Lets test for ImplicitDirs which is goverened by implicit-dirs flags
func TestArgsParsing_ImplicitDirsFlag(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedImplicit bool
	}{
		{
			name:             "normal",
			args:             []string{"gcsfuse", "--implicit-dirs", "abc", "pqr"},
			expectedImplicit: true,
		},
		{
			name:             "default",
			args:             []string{"gcsfuse", "abc", "pqr"},
			expectedImplicit: false,
		},
		{
			name:             "normal_false",
			args:             []string{"gcsfuse", "--implicit-dirs=false", "abc", "pqr"},
			expectedImplicit: false,
		},
		{
			name:             "default false on high performance machine with autoconfig disabled",
			args:             []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=true", "abc", "pqr"},
			expectedImplicit: false,
		},
		{
			name:             "default true on high performance machine with autoconfig enabled",
			args:             []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "abc", "pqr"},
			expectedImplicit: true,
		},
		{
			name:             "default overriden on high performance machine",
			args:             []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "--implicit-dirs=false", "abc", "pqr"},
			expectedImplicit: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotImplicit bool
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotImplicit = cfg.ImplicitDirs
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedImplicit, gotImplicit)
			}
		})
	}
}
func TestArgsParsing_WriteConfigFlags(t *testing.T) {
	tests := []struct {
		name                          string
		args                          []string
		expectedCreateEmptyFile       bool
		expectedEnableStreamingWrites bool
		expectedEnableRapidAppends    bool
		expectedWriteBlockSizeMB      int64
		expectedWriteGlobalMaxBlocks  int64
		expectedWriteMaxBlocksPerFile int64
	}{
		{
			name:                          "Test create-empty-file flag true works when streaming writes are explicitly disabled.",
			args:                          []string{"gcsfuse", "--create-empty-file=true", "--enable-streaming-writes=false", "abc", "pqr"},
			expectedCreateEmptyFile:       true,
			expectedEnableStreamingWrites: false,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test create-empty-file flag false.",
			args:                          []string{"gcsfuse", "--create-empty-file=false", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test default flags.",
			args:                          []string{"gcsfuse", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test enable-streaming-writes flag true.",
			args:                          []string{"gcsfuse", "--enable-streaming-writes", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test enable-streaming-writes flag false.",
			args:                          []string{"gcsfuse", "--enable-streaming-writes=false", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: false,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test enable-rapid-appends flag true.",
			args:                          []string{"gcsfuse", "--enable-rapid-appends=false", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    false,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test positive write-block-size-mb flag.",
			args:                          []string{"gcsfuse", "--enable-streaming-writes", "--write-block-size-mb=10", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      10,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test positive write-global-max-blocks flag.",
			args:                          []string{"gcsfuse", "--enable-streaming-writes", "--write-global-max-blocks=10", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  10,
			expectedWriteMaxBlocksPerFile: 1,
		},
		{
			name:                          "Test positive write-max-blocks-per-file flag.",
			args:                          []string{"gcsfuse", "--enable-streaming-writes", "--write-max-blocks-per-file=10", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  4,
			expectedWriteMaxBlocksPerFile: 10,
		},
		{
			name:                          "Test high performance config values.",
			args:                          []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "abc", "pqr"},
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  1600,
		},
		{
			name:                          "Test high performance config values with --write-global-max-blocks flag overriden.",
			args:                          []string{"gcsfuse", "--write-global-max-blocks=2000", "--disable-autoconfig=false", "abc", "pqr"},
			expectedCreateEmptyFile:       false,
			expectedEnableStreamingWrites: true,
			expectedEnableRapidAppends:    true,
			expectedWriteBlockSizeMB:      32,
			expectedWriteGlobalMaxBlocks:  2000,
			expectedWriteMaxBlocksPerFile: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var wc cfg.WriteConfig
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				wc = cfg.Write
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedCreateEmptyFile, wc.CreateEmptyFile)
				assert.Equal(t, tc.expectedEnableStreamingWrites, wc.EnableStreamingWrites)
				assert.Equal(t, tc.expectedWriteBlockSizeMB, wc.BlockSizeMb)
				assert.Equal(t, tc.expectedWriteGlobalMaxBlocks, wc.GlobalMaxBlocks)
				assert.Equal(t, tc.expectedEnableRapidAppends, wc.EnableRapidAppends)
			}
		})
	}
}

func TestArgsParsing_ReadConfigFlags(t *testing.T) {
	tests := []struct {
		name                             string
		args                             []string
		expectedReadBlockSizeMB          int64
		expectedReadGlobalMaxBlocks      int64
		expectedReadMaxBlocksPerHandle   int64
		expectedReadStartBlocksPerHandle int64
		expectedReadMinBlocksPerHandle   int64
	}{
		{
			name:                             "Test default flags.",
			args:                             []string{"gcsfuse", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test enable buffered read flag true.",
			args:                             []string{"gcsfuse", "--enable-buffered-read", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test enable buffered read flag false.",
			args:                             []string{"gcsfuse", "--enable-buffered-read=false", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test positive read-block-size-mb flag.",
			args:                             []string{"gcsfuse", "--read-block-size-mb=10", "abc", "pqr"},
			expectedReadBlockSizeMB:          10,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test positive read-global-max-blocks flag.",
			args:                             []string{"gcsfuse", "--read-global-max-blocks=10", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      10,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test positive read-max-blocks-per-handle flag.",
			args:                             []string{"gcsfuse", "--read-max-blocks-per-handle=10", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   10,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test positive read-start-blocks-per-handle flag.",
			args:                             []string{"gcsfuse", "--read-start-blocks-per-handle=10", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 10,
			expectedReadMinBlocksPerHandle:   4,
		},
		{
			name:                             "Test positive read-min-blocks-per-handle flag.",
			args:                             []string{"gcsfuse", "--read-min-blocks-per-handle=10", "abc", "pqr"},
			expectedReadBlockSizeMB:          16,
			expectedReadGlobalMaxBlocks:      40,
			expectedReadMaxBlocksPerHandle:   20,
			expectedReadStartBlocksPerHandle: 1,
			expectedReadMinBlocksPerHandle:   10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rc cfg.ReadConfig
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				rc = cfg.Read
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedReadBlockSizeMB, rc.BlockSizeMb)
				assert.Equal(t, tc.expectedReadGlobalMaxBlocks, rc.GlobalMaxBlocks)
				assert.Equal(t, tc.expectedReadMaxBlocksPerHandle, rc.MaxBlocksPerHandle)
				assert.Equal(t, tc.expectedReadStartBlocksPerHandle, rc.StartBlocksPerHandle)
				assert.Equal(t, tc.expectedReadMinBlocksPerHandle, rc.MinBlocksPerHandle)
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
			args: []string{"gcsfuse", "--file-cache-cache-file-for-range-read", "--file-cache-download-chunk-size-mb=20", "--file-cache-enable-crc", "--cache-dir=/some/valid/dir", "--file-cache-experimental-exclude-regex=.*", "--file-cache-enable-parallel-downloads", "--file-cache-max-parallel-downloads=40", "--file-cache-max-size-mb=100", "--file-cache-parallel-downloads-per-file=2", "--file-cache-enable-o-direct=false", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				CacheDir: "/some/valid/dir",
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:                  true,
					DownloadChunkSizeMb:                    20,
					EnableCrc:                              true,
					EnableParallelDownloads:                true,
					ExperimentalExcludeRegex:               ".*",
					ExperimentalParallelDownloadsDefaultOn: true,
					MaxParallelDownloads:                   40,
					MaxSizeMb:                              100,
					ParallelDownloadsPerFile:               2,
					WriteBufferSize:                        4 * 1024 * 1024,
					EnableODirect:                          false,
				},
			},
		},
		{
			name: "Test default file cache flags.",
			args: []string{"gcsfuse", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileCache: cfg.FileCacheConfig{
					CacheFileForRangeRead:                  false,
					DownloadChunkSizeMb:                    200,
					EnableCrc:                              false,
					EnableParallelDownloads:                false,
					ExperimentalExcludeRegex:               "",
					ExperimentalParallelDownloadsDefaultOn: true,
					MaxParallelDownloads:                   int64(max(16, 2*runtime.NumCPU())),
					MaxSizeMb:                              -1,
					ParallelDownloadsPerFile:               16,
					WriteBufferSize:                        4 * 1024 * 1024,
					EnableODirect:                          false,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				experimentalMetadataPrefetch = cfg.MetadataCache.ExperimentalMetadataPrefetchOnMount
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			args: []string{"gcsfuse", "--dir-mode=0777", "--disable-parallel-dirops", "--experimental-enable-dentry-cache", "--experimental-enable-readdirplus", "--file-mode=0666", "--o", "ro", "--gid=7", "--ignore-interrupts=false", "--kernel-list-cache-ttl-secs=300", "--rename-dir-limit=10", "--temp-dir=~/temp", "--uid=8", "--precondition-errors=false", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0777,
					DisableParallelDirops:         true,
					ExperimentalEnableDentryCache: true,
					ExperimentalEnableReaddirplus: true,
					FileMode:                      0666,
					FuseOptions:                   []string{"ro"},
					Gid:                           7,
					IgnoreInterrupts:              false,
					KernelListCacheTtlSecs:        300,
					RenameDirLimit:                10,
					TempDir:                       cfg.ResolvedPath(path.Join(hd, "temp")),
					PreconditionErrors:            false,
					Uid:                           8,
				},
			},
		},
		{
			name: "mode_flags_without_0_prefix",
			args: []string{"gcsfuse", "--dir-mode=777", "--file-mode=666", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0777,
					DisableParallelDirops:         false,
					ExperimentalEnableDentryCache: false,
					ExperimentalEnableReaddirplus: false,
					FileMode:                      0666,
					FuseOptions:                   []string{},
					Gid:                           -1,
					IgnoreInterrupts:              true,
					KernelListCacheTtlSecs:        0,
					RenameDirLimit:                0,
					TempDir:                       "",
					PreconditionErrors:            true,
					Uid:                           -1,
				},
			},
		},
		{
			name: "high performance defaults with rename dir options with autoconfig enabled",
			args: []string{"gcsfuse", "--dir-mode=777", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "--file-mode=666", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0777,
					DisableParallelDirops:         false,
					ExperimentalEnableDentryCache: false,
					ExperimentalEnableReaddirplus: false,
					FileMode:                      0666,
					FuseOptions:                   []string{},
					Gid:                           -1,
					IgnoreInterrupts:              true,
					KernelListCacheTtlSecs:        0,
					RenameDirLimit:                200000,
					TempDir:                       "",
					PreconditionErrors:            true,
					Uid:                           -1,
				},
			},
		},
		{
			name: "high performance defaults with rename dir options with autoconfig disabled",
			args: []string{"gcsfuse", "--dir-mode=777", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=true", "--file-mode=666", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0777,
					DisableParallelDirops:         false,
					ExperimentalEnableDentryCache: false,
					ExperimentalEnableReaddirplus: false,
					FileMode:                      0666,
					FuseOptions:                   []string{},
					Gid:                           -1,
					IgnoreInterrupts:              true,
					KernelListCacheTtlSecs:        0,
					RenameDirLimit:                0,
					TempDir:                       "",
					PreconditionErrors:            true,
					Uid:                           -1,
				},
			},
		},
		{
			name: "high performance defaults with overriden rename dir options",
			args: []string{"gcsfuse", "--dir-mode=777", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "--rename-dir-limit=15000", "--file-mode=666", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0777,
					DisableParallelDirops:         false,
					ExperimentalEnableDentryCache: false,
					ExperimentalEnableReaddirplus: false,
					FileMode:                      0666,
					FuseOptions:                   []string{},
					Gid:                           -1,
					IgnoreInterrupts:              true,
					KernelListCacheTtlSecs:        0,
					RenameDirLimit:                15000,
					TempDir:                       "",
					PreconditionErrors:            true,
					Uid:                           -1,
				},
			},
		},
		{
			name: "default",
			args: []string{"gcsfuse", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					DirMode:                       0755,
					DisableParallelDirops:         false,
					ExperimentalEnableDentryCache: false,
					ExperimentalEnableReaddirplus: false,
					FileMode:                      0644,
					FuseOptions:                   []string{},
					Gid:                           -1,
					IgnoreInterrupts:              true,
					KernelListCacheTtlSecs:        0,
					RenameDirLimit:                0,
					TempDir:                       "",
					PreconditionErrors:            true,
					Uid:                           -1,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
		{
			name: "invalid_experimental_enable_readdirplus",
			args: []string{"gcsfuse", "--experimental-enable-readdirplus=abc", "abc", "pqr"},
		},
		{
			name: "invalid_experimental_enable_dentry_cache",
			args: []string{"gcsfuse", "--experimental-enable-dentry-cache=abc", "abc", "pqr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

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
			expectedEnableHNS: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotEnableHNS bool
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotEnableHNS = cfg.EnableHns
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedEnableHNS, gotEnableHNS)
			}
		})
	}
}

func TestArgsParsing_EnableGoogleLibAuthFlag(t *testing.T) {
	tests := []struct {
		name                        string
		args                        []string
		expectedEnableGoogleLibAuth bool
	}{
		{
			name:                        "default",
			args:                        []string{"gcsfuse", "abc", "pqr"},
			expectedEnableGoogleLibAuth: false,
		},
		{
			name:                        "normal",
			args:                        []string{"gcsfuse", "--enable-google-lib-auth=false", "abc", "pqr"},
			expectedEnableGoogleLibAuth: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotEnableGoogleLibAuth bool
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotEnableGoogleLibAuth = cfg.EnableGoogleLibAuth
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedEnableGoogleLibAuth, gotEnableGoogleLibAuth)
			}
		})
	}
}

func TestArgsParsing_EnableAtomicRenameObjectFlag(t *testing.T) {
	tests := []struct {
		name                             string
		args                             []string
		expectedEnableAtomicRenameObject bool
	}{
		{
			name:                             "default",
			args:                             []string{"gcsfuse", "abc", "pqr"},
			expectedEnableAtomicRenameObject: true,
		},
		{
			name:                             "normal",
			args:                             []string{"gcsfuse", "--enable-atomic-rename-object=false", "abc", "pqr"},
			expectedEnableAtomicRenameObject: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotEnableAtomicRenameObject bool
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotEnableAtomicRenameObject = cfg.EnableAtomicRenameObject
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedEnableAtomicRenameObject, gotEnableAtomicRenameObject)
			}
		})
	}
}

func TestArgsParsing_EnableNewReaderFlag(t *testing.T) {
	tests := []struct {
		name                    string
		args                    []string
		expectedEnableNewReader bool
	}{
		{
			name:                    "default",
			args:                    []string{"gcsfuse", "abc", "pqr"},
			expectedEnableNewReader: true,
		},
		{
			name:                    "normal",
			args:                    []string{"gcsfuse", "--enable-new-reader=false", "abc", "pqr"},
			expectedEnableNewReader: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotEnableNewReader bool
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotEnableNewReader = cfg.EnableNewReader
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			require.NoError(t, err)
			assert.Equal(t, tc.expectedEnableNewReader, gotEnableNewReader)
		})
	}
}

func TestArgsParsing_MetricsFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *cfg.MetricsConfig
	}{
		{
			name: "cloud-metrics-export-interval-secs-positive",
			args: []string{"gcsfuse", "--cloud-metrics-export-interval-secs=10", "abc", "pqr"},
			expected: &cfg.MetricsConfig{
				CloudMetricsExportIntervalSecs: 10,
				Workers:                        3,
				BufferSize:                     256,
			},
		},
		{
			name: "stackdriver-export-interval-positive",
			args: []string{"gcsfuse", "--stackdriver-export-interval=10h", "abc", "pqr"},
			expected: &cfg.MetricsConfig{
				CloudMetricsExportIntervalSecs: 10 * 3600,
				StackdriverExportInterval:      time.Duration(10) * time.Hour,
				Workers:                        3,
				BufferSize:                     256,
			},
		},
		{
			name: "use_new_metric_names",
			args: []string{"gcsfuse", "--metrics-use-new-names=true", "abc", "pqr"},
			expected: &cfg.MetricsConfig{
				UseNewNames: true,
				Workers:     3,
				BufferSize:  256,
			},
		},
		{
			name: "metrics_workers_non_default",
			args: []string{"gcsfuse", "--metrics-workers=10", "abc", "pqr"},
			expected: &cfg.MetricsConfig{
				Workers:    10,
				BufferSize: 256,
			},
		},
		{
			name: "metrics_buffer_size_non_default",
			args: []string{"gcsfuse", "--metrics-buffer-size=1024", "abc", "pqr"},
			expected: &cfg.MetricsConfig{
				Workers:    3,
				BufferSize: 1024,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, &gotConfig.Metrics)
			}
		})
	}
}

func TestArgsParsing_MetricsViewConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfgFile  string
		expected *cfg.MetricsConfig
	}{
		{
			name:     "default",
			cfgFile:  "empty.yml",
			expected: &cfg.MetricsConfig{Workers: 3, BufferSize: 256},
		},
		{
			name:     "cloud-metrics-export-interval-secs-positive",
			cfgFile:  "metrics_export_interval_positive.yml",
			expected: &cfg.MetricsConfig{CloudMetricsExportIntervalSecs: 100, Workers: 3, BufferSize: 256},
		},
		{
			name:    "stackdriver-export-interval-positive",
			cfgFile: "stackdriver_export_interval_positive.yml",
			expected: &cfg.MetricsConfig{
				CloudMetricsExportIntervalSecs: 12 * 3600,
				StackdriverExportInterval:      12 * time.Hour,
				Workers:                        3,
				BufferSize:                     256,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs([]string{"gcsfuse", fmt.Sprintf("--config-file=testdata/metrics_config/%s", tc.cfgFile), "abc", "pqr"}, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, &gotConfig.Metrics)
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
			args: []string{"gcsfuse", "--stat-cache-capacity=2000", "--stat-cache-ttl=2m", "--type-cache-ttl=1m20s", "--enable-nonexistent-type-cache", "--experimental-metadata-prefetch-on-mount=async", "--stat-cache-max-size-mb=15", "--metadata-cache-ttl-secs=25", "--metadata-cache-negative-ttl-secs=20", "--type-cache-max-size-mb=30", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         2000,
					DeprecatedStatCacheTtl:              2 * time.Minute,
					DeprecatedTypeCacheTtl:              80 * time.Second,
					EnableNonexistentTypeCache:          true,
					ExperimentalMetadataPrefetchOnMount: "async",
					StatCacheMaxSizeMb:                  15,
					TtlSecs:                             25,
					NegativeTtlSecs:                     20,
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
					StatCacheMaxSizeMb:                  33,
					TtlSecs:                             60,
					NegativeTtlSecs:                     5,
					TypeCacheMaxSizeMb:                  4,
				},
			},
		},
		{
			name: "high performance default config values with autoconfig disabled",
			args: []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=true", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         20460,
					DeprecatedStatCacheTtl:              60 * time.Second,
					DeprecatedTypeCacheTtl:              60 * time.Second,
					EnableNonexistentTypeCache:          false,
					ExperimentalMetadataPrefetchOnMount: "disabled",
					StatCacheMaxSizeMb:                  33,
					TtlSecs:                             60,
					NegativeTtlSecs:                     5,
					TypeCacheMaxSizeMb:                  4,
				},
			},
		},
		{
			name: "high performance default config values with autoconfig enabled",
			args: []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         20460,
					DeprecatedStatCacheTtl:              60 * time.Second,
					DeprecatedTypeCacheTtl:              60 * time.Second,
					EnableNonexistentTypeCache:          false,
					ExperimentalMetadataPrefetchOnMount: "disabled",
					StatCacheMaxSizeMb:                  1024,
					TtlSecs:                             9223372036,
					NegativeTtlSecs:                     0,
					TypeCacheMaxSizeMb:                  128,
				},
			},
		},
		{
			name: "high performance default config values obey customer flags",
			args: []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "--stat-cache-capacity=2000", "--stat-cache-ttl=2m", "--type-cache-ttl=1m20s", "--enable-nonexistent-type-cache", "--experimental-metadata-prefetch-on-mount=async", "--stat-cache-max-size-mb=15", "--metadata-cache-ttl-secs=25", "--metadata-cache-negative-ttl-secs=20", "--type-cache-max-size-mb=30", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         2000,
					DeprecatedStatCacheTtl:              2 * time.Minute,
					DeprecatedTypeCacheTtl:              80 * time.Second,
					EnableNonexistentTypeCache:          true,
					ExperimentalMetadataPrefetchOnMount: "async",
					StatCacheMaxSizeMb:                  15,
					TtlSecs:                             25,
					NegativeTtlSecs:                     20,
					TypeCacheMaxSizeMb:                  30,
				},
			},
		},
		{
			name: "high performance default config values use deprecated flags",
			args: []string{"gcsfuse", "--machine-type=a3-highgpu-8g", "--disable-autoconfig=false", "--stat-cache-capacity=2000", "--stat-cache-ttl=2m", "--type-cache-ttl=4m", "--enable-nonexistent-type-cache", "--experimental-metadata-prefetch-on-mount=async", "--metadata-cache-negative-ttl-secs=20", "--type-cache-max-size-mb=30", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				MetadataCache: cfg.MetadataCacheConfig{
					DeprecatedStatCacheCapacity:         2000,
					DeprecatedStatCacheTtl:              2 * time.Minute,
					DeprecatedTypeCacheTtl:              4 * time.Minute,
					EnableNonexistentTypeCache:          true,
					ExperimentalMetadataPrefetchOnMount: "async",
					StatCacheMaxSizeMb:                  4,
					TtlSecs:                             120,
					NegativeTtlSecs:                     20,
					TypeCacheMaxSizeMb:                  30,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.MetadataCache, gotConfig.MetadataCache)
			}
		})
	}
}

func TestArgParsing_GCSRetries(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig *cfg.Config
	}{
		{
			name: "Test with non default chunkTransferTimeout",
			args: []string{"gcsfuse", "--chunk-transfer-timeout-secs=30", "abc", "pqr"},
			expectedConfig: &cfg.Config{
				GcsRetries: cfg.GcsRetriesConfig{
					ChunkTransferTimeoutSecs: 30,
					MaxRetryAttempts:         0,
					MaxRetrySleep:            30 * time.Second,
					Multiplier:               2,
					ReadStall: cfg.ReadStallGcsRetriesConfig{
						Enable:              true,
						InitialReqTimeout:   20 * time.Second,
						MinReqTimeout:       1500 * time.Millisecond,
						MaxReqTimeout:       1200 * time.Second,
						ReqIncreaseRate:     15,
						ReqTargetPercentile: 0.99,
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig.GcsRetries, gotConfig.GcsRetries)
			}
		})
	}
}

func TestArgsParsing_ProfilerFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig cfg.ProfilingConfig
	}{
		{
			name: "Default profiler config (disabled)",
			args: []string{"gcsfuse", "bucket", "mountpoint"},
			expectedConfig: cfg.ProfilingConfig{
				Enabled:       false, // Profiler is disabled by default
				Label:         "gcsfuse-0.0.0",
				Mutex:         false, // Default for --profiling-mutex
				Cpu:           true,  // Default for --profiling-cpu
				AllocatedHeap: true,  // Default for --profiling-allocated-heap
				Heap:          true,  // Default for --profiling-heap
				Goroutines:    false, // Default for --profiling-goroutines
			},
		},
		{
			name: "Profiler enabled, sub-profilers default",
			args: []string{"gcsfuse", "--enable-cloud-profiling", "bucket", "mountpoint"},
			expectedConfig: cfg.ProfilingConfig{
				Enabled:       true,
				Label:         "gcsfuse-0.0.0",
				Mutex:         false,
				Cpu:           true,
				AllocatedHeap: true,
				Heap:          true,
				Goroutines:    false,
			},
		},
		{
			name: "Profiler enabled, all sub-profilers explicitly true and label set",
			args: []string{"gcsfuse", "--enable-cloud-profiling", "--profiling-label=v1.0.0", "--profiling-mutex=true", "--profiling-cpu=true", "--profiling-allocated-heap=true", "--profiling-heap=true", "--profiling-goroutines=true", "bucket", "mountpoint"},
			expectedConfig: cfg.ProfilingConfig{
				Enabled:       true,
				Label:         "v1.0.0",
				Mutex:         true,
				Cpu:           true,
				AllocatedHeap: true,
				Heap:          true,
				Goroutines:    true,
			},
		},
		{
			name: "Profiler enabled, all sub-profilers explicitly false",
			args: []string{"gcsfuse", "--enable-cloud-profiling", "--profiling-mutex=false", "--profiling-cpu=false", "--profiling-allocated-heap=false", "--profiling-heap=false", "--profiling-goroutines=false", "bucket", "mountpoint"},
			expectedConfig: cfg.ProfilingConfig{
				Enabled:       true,
				Label:         "gcsfuse-0.0.0",
				Mutex:         false,
				Cpu:           false,
				AllocatedHeap: false,
				Heap:          false,
				Goroutines:    false,
			},
		},
		{
			name: "Profiler explicitly disabled, some sub-profiler flags set",
			args: []string{"gcsfuse", "--enable-cloud-profiling=false", "--profiling-mutex=true", "--profiling-cpu=false", "bucket", "mountpoint"},
			expectedConfig: cfg.ProfilingConfig{
				Enabled:       false, // Master switch is off
				Label:         "gcsfuse-0.0.0",
				Mutex:         true,  // Flag was parsed
				Cpu:           false, // Flag was parsed
				AllocatedHeap: true,  // Default for its flag
				Heap:          true,  // Default for its flag
				Goroutines:    false, // Default for its flag
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotProfilerConfig cfg.ProfilingConfig
			cmd, err := newRootCmd(func(c *cfg.Config, _, _ string) error {
				gotProfilerConfig = c.Profiling
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs(tc.args, cmd))

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedConfig, gotProfilerConfig)
			}
		})
	}
}

func TestArgsParsing_ReadInactiveTimeoutConfig(t *testing.T) {
	tests := []struct {
		name            string
		cfgFile         string
		expectedTimeout time.Duration
	}{
		{
			name:            "default",
			cfgFile:         "empty.yaml",
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "override_default",
			cfgFile:         "override.yaml",
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "override_with_grpc",
			cfgFile:         "override_with_grpc.yaml",
			expectedTimeout: 30 * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotConfig *cfg.Config
			cmd, err := newRootCmd(func(cfg *cfg.Config, _, _ string) error {
				gotConfig = cfg
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(convertToPosixArgs([]string{"gcsfuse", fmt.Sprintf("--config-file=testdata/read_config/%s", tc.cfgFile), "abc", "pqr"}, cmd))

			err = cmd.Execute()

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTimeout, gotConfig.Read.InactiveStreamTimeout)
		})
	}
}

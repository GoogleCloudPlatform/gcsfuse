// Copyright 2024 Google Inc. All Rights Reserved.
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
	"testing"

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
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _ string, _ string) error {
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
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _ string, _ string) error {
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
		name                             string
		args                             []string
		expectedCacheFileForRangeRead    bool
		expectedDownloadChunkSizeMb      int64
		expectedEnableCrc                bool
		expectedEnableParallelDownloads  bool
		expectedMaxParallelDownloads     int64
		expectedMaxSizeMb                int64
		expectedParallelDownloadsPerFile int64
	}{
		{
			name:                             "Test default file cache flags.",
			args:                             []string{"gcsfuse", "--cache-file-for-range-read", "--download-chunk-size-mb=20", "--enable-crc", "--enable-parallel-downloads", "--max-parallel-downloads=40", "--file-cache-max-size-mb=100", "--parallel-downloads-per-file=2", "abc", "pqr"},
			expectedCacheFileForRangeRead:    true,
			expectedDownloadChunkSizeMb:      20,
			expectedEnableCrc:                true,
			expectedEnableParallelDownloads:  true,
			expectedMaxParallelDownloads:     40,
			expectedMaxSizeMb:                100,
			expectedParallelDownloadsPerFile: 2,
		},
		{
			name:                             "Test default file cache flags.",
			args:                             []string{"gcsfuse", "abc", "pqr"},
			expectedCacheFileForRangeRead:    false,
			expectedDownloadChunkSizeMb:      50,
			expectedEnableCrc:                false,
			expectedEnableParallelDownloads:  false,
			expectedMaxParallelDownloads:     int64(cfg.DefaultMaxParallelDownloads()),
			expectedMaxSizeMb:                -1,
			expectedParallelDownloadsPerFile: 16,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var cacheFileForRangeRead, enableCrc, enableParallelDownloads bool
			var downloadChunkSizeMb, maxParallelDownloads, maxSizeMb, parallelDownloadsPerFile int64
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _ string, _ string) error {
				cacheFileForRangeRead = cfg.FileCache.CacheFileForRangeRead
				downloadChunkSizeMb = cfg.FileCache.DownloadChunkSizeMb
				enableCrc = cfg.FileCache.EnableCrc
				enableParallelDownloads = cfg.FileCache.EnableParallelDownloads
				maxParallelDownloads = cfg.FileCache.MaxParallelDownloads
				maxSizeMb = cfg.FileCache.MaxSizeMb
				parallelDownloadsPerFile = cfg.FileCache.ParallelDownloadsPerFile
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedCacheFileForRangeRead, cacheFileForRangeRead)
				assert.Equal(t, tc.expectedDownloadChunkSizeMb, downloadChunkSizeMb)
				assert.Equal(t, tc.expectedEnableCrc, enableCrc)
				assert.Equal(t, tc.expectedEnableParallelDownloads, enableParallelDownloads)
				assert.Equal(t, tc.expectedMaxParallelDownloads, maxParallelDownloads)
				assert.Equal(t, tc.expectedMaxSizeMb, maxSizeMb)
				assert.Equal(t, tc.expectedParallelDownloadsPerFile, parallelDownloadsPerFile)
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
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _ string, _ string) error {
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
			cmd, err := NewRootCmd(func(cfg *cfg.Config, _ string, _ string) error {
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			assert.Error(t, err)
		})
	}
}

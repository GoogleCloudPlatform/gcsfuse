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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cmd

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	var actual cfg.Config
	cmd, err := NewRootCmd(func(c cfg.Config) error {
		actual = c
		return nil
	})
	require.Nil(t, err)
	cmd.SetArgs([]string{"abc", "pqr"})

	if assert.Nil(t, cmd.Execute()) {

		assert.Equal(t, "", actual.AppName)

		assert.Equal(t, false, actual.Foreground)

		assert.Equal(t, []string{}, actual.FileSystem.FuseOptions)

		assert.Equal(t, cfg.Octal(0755), actual.FileSystem.DirMode)

		assert.Equal(t, cfg.Octal(0644), actual.FileSystem.FileMode)

		assert.Equal(t, -1, actual.FileSystem.Uid)

		assert.Equal(t, -1, actual.FileSystem.Gid)

		assert.Equal(t, false, actual.ImplicitDirs)

		assert.Equal(t, "", actual.OnlyDir)

		assert.Equal(t, 0, actual.FileSystem.RenameDirLimit)

		assert.Equal(t, true, actual.FileSystem.IgnoreInterrupts)

		assert.Equal(t, false, actual.FileSystem.DisableParallelDirops)

		if v, err := cfg.ParseURL(""); assert.Nil(t, err) {
			assert.Equal(t, v, actual.GcsConnection.CustomEndpoint)
		}

		assert.Equal(t, false, actual.GcsAuth.AnonymousAccess)

		assert.Equal(t, "", actual.GcsConnection.BillingProject)

		if v, err := cfg.GetNewResolvedPath(""); assert.Nil(t, err) {
			assert.Equal(t, v, actual.GcsAuth.KeyFile)
		}

		assert.Equal(t, "", actual.GcsAuth.TokenUrl)

		assert.Equal(t, true, actual.GcsAuth.ReuseTokenFromUrl)

		assert.Equal(t, float64(-1), actual.GcsConnection.LimitBytesPerSec)

		assert.Equal(t, float64(-1), actual.GcsConnection.LimitOpsPerSec)

		assert.Equal(t, 200, actual.GcsConnection.SequentialReadSizeMb)

		if v, err := time.ParseDuration("30s"); assert.Nil(t, err) {
			assert.Equal(t, v, actual.GcsRetries.MaxRetrySleep)
		}

		assert.Equal(t, 32, actual.MetadataCache.StatCacheMaxSizeMb)

		assert.Equal(t, 4, actual.MetadataCache.TypeCacheMaxSizeMb)

		assert.Equal(t, 60, actual.MetadataCache.TtlSecs)

		assert.Equal(t, 20460, actual.MetadataCache.DeprecatedStatCacheCapacity)

		if v, err := time.ParseDuration("60s"); assert.Nil(t, err) {
			assert.Equal(t, v, actual.MetadataCache.DeprecatedStatCacheTtl)
		}

		if v, err := time.ParseDuration("60s"); assert.Nil(t, err) {
			assert.Equal(t, v, actual.MetadataCache.DeprecatedTypeCacheTtl)
		}

		assert.Equal(t, 0, actual.List.KernelListCacheTtlSecs)

		if v, err := time.ParseDuration("0s"); assert.Nil(t, err) {
			assert.Equal(t, v, actual.GcsConnection.HttpClientTimeout)
		}

		assert.Equal(t, float64(2), actual.GcsRetries.Multiplier)

		if v, err := cfg.GetNewResolvedPath(""); assert.Nil(t, err) {
			assert.Equal(t, v, actual.FileSystem.TempDir)
		}

		assert.Equal(t, cfg.Protocol("http1"), actual.GcsConnection.ClientProtocol)

		assert.Equal(t, 0, actual.GcsConnection.MaxConnsPerHost)

		assert.Equal(t, 100, actual.GcsConnection.MaxIdleConnsPerHost)

		assert.Equal(t, false, actual.MetadataCache.EnableNonexistentTypeCache)

		if v, err := time.ParseDuration("0s"); assert.Nil(t, err) {
			assert.Equal(t, v, actual.Metrics.StackdriverExportInterval)
		}

		assert.Equal(t, "", actual.Monitoring.ExperimentalOpentelemetryCollectorAddress)

		if v, err := cfg.GetNewResolvedPath(""); assert.Nil(t, err) {
			assert.Equal(t, v, actual.Logging.FilePath)
		}

		assert.Equal(t, "json", actual.Logging.Format)

		assert.Equal(t, false, actual.GcsConnection.ExperimentalEnableJsonRead)

		assert.Equal(t, false, actual.Debug.Gcs)

		assert.Equal(t, false, actual.Debug.ExitOnInvariantViolation)

		assert.Equal(t, false, actual.Debug.LogMutex)

		assert.Equal(t, "disabled", actual.MetadataCache.ExperimentalMetadataPrefetchOnMount)

		assert.Equal(t, false, actual.Write.CreateEmptyFile)

		assert.Equal(t, cfg.LogSeverity("INFO"), actual.Logging.Severity)

		assert.Equal(t, -1, actual.FileCache.MaxSizeMb)

		assert.Equal(t, false, actual.FileCache.CacheFileForRangeRead)

		assert.Equal(t, false, actual.FileCache.EnableCrc)

		assert.Equal(t, false, actual.FileCache.EnableParallelDownloads)

		assert.Equal(t, 16, actual.FileCache.ParallelDownloadsPerFile)

		assert.Equal(t, config.DefaultMaxParallelDownloads(), actual.FileCache.MaxParallelDownloads)

		assert.Equal(t, 50, actual.FileCache.DownloadChunkSizeMb)

		if v, err := cfg.GetNewResolvedPath(""); assert.Nil(t, err) {
			assert.Equal(t, v, actual.CacheDir)
		}

		assert.Equal(t, false, actual.List.EnableEmptyManagedFolders)

		assert.Equal(t, 0, actual.GcsConnection.GrpcConnPoolSize)

		assert.Equal(t, false, actual.EnableHns)

		assert.Equal(t, 512, actual.Logging.LogRotate.MaxFileSizeMb)

		assert.Equal(t, 10, actual.Logging.LogRotate.BackupFileCount)

		assert.Equal(t, true, actual.Logging.LogRotate.Compress)

	}
}

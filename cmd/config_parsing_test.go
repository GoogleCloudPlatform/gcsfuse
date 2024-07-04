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
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConfigObject(t *testing.T, args []string) (*cfg.Config, error) {
	t.Helper()
	var c cfg.Config
	cmd, err := NewRootCmd(func(config cfg.Config) error {
		c = config
		return nil
	})
	require.Nil(t, err)
	cmdArgs := append([]string{"gcsfuse"}, args...)
	cmdArgs = append(cmdArgs, "a")
	cmd.SetArgs(cmdArgs)
	if err = cmd.Execute(); err != nil {
		return nil, err
	}

	return &c, nil
}

func getConfigObjectWithConfigFile(t *testing.T, configFilePath string) (*cfg.Config, error) {
	t.Helper()
	return getConfigObject(t, []string{fmt.Sprintf("--config-file=%s", configFilePath)})
}

func validateDefaultConfig(t *testing.T, defCfg *cfg.Config) {
	assert.False(t, defCfg.Write.CreateEmptyFile)
	assert.False(t, defCfg.List.EnableEmptyManagedFolders)
	assert.Equal(t, "INFO", string(defCfg.Logging.Severity))
	assert.Equal(t, "json", defCfg.Logging.Format)
	assert.Equal(t, cfg.ResolvedPath(""), defCfg.Logging.FilePath)
	assert.Equal(t, int64(512), defCfg.Logging.LogRotate.MaxFileSizeMb)
	assert.Equal(t, int64(10), defCfg.Logging.LogRotate.BackupFileCount)
	assert.True(t, defCfg.Logging.LogRotate.Compress)
	assert.Equal(t, "", string(defCfg.CacheDir))
	assert.Equal(t, int64(-1), defCfg.FileCache.MaxSizeMb)
	assert.False(t, defCfg.FileCache.CacheFileForRangeRead)
	assert.False(t, defCfg.FileCache.EnableParallelDownloads)
	assert.Equal(t, int64(16), defCfg.FileCache.ParallelDownloadsPerFile)
	assert.LessOrEqual(t, int64(16), defCfg.FileCache.MaxParallelDownloads)
	assert.Equal(t, int64(50), defCfg.FileCache.DownloadChunkSizeMb)
	assert.False(t, defCfg.FileCache.EnableCrc)
	assert.Equal(t, int64(1), defCfg.GcsConnection.GrpcConnPoolSize)
	assert.False(t, defCfg.GcsAuth.AnonymousAccess)
	assert.False(t, defCfg.EnableHns)
	assert.True(t, defCfg.FileSystem.IgnoreInterrupts)
	assert.False(t, defCfg.FileSystem.DisableParallelDirops)
	assert.Equal(t, int64(0), defCfg.FileSystem.KernelListCacheTtlSecs)
}
func TestReadConfigFile_DefaultConfig(t *testing.T) {
	defCfg, err := getConfigObject(t, nil)

	require.Nil(t, err)
	validateDefaultConfig(t, defCfg)
}

func TestReadConfigFile_EmptyFile(t *testing.T) {
	mountConfig, err := getConfigObjectWithConfigFile(t, "testdata/empty_file.yaml")

	require.Nil(t, err)
	validateDefaultConfig(t, mountConfig)
}

func TestReadConfigFile_NonExistingFile(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/nofile.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/invalid_config.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_ValidConfigWith0BackupFileCount(t *testing.T) {
	mountConfig, err := getConfigObjectWithConfigFile(t, "testdata/valid_config_with_0_backup-file-count.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.True(t, mountConfig.Write.CreateEmptyFile)
	assert.Equal(t, cfg.LogSeverity("ERROR"), mountConfig.Logging.Severity)
	assert.Equal(t, cfg.ResolvedPath("/tmp/logfile.json"), mountConfig.Logging.FilePath)
	assert.Equal(t, "text", mountConfig.Logging.Format)
	assert.Equal(t, int64(0), mountConfig.Logging.LogRotate.BackupFileCount)
	assert.False(t, mountConfig.Logging.LogRotate.Compress)
}

func TestReadConfigFile_Invalid_UnexpectedField_Config(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/invalid_unexpectedfield_config.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_ValidConfig(t *testing.T) {
	mountConfig, err := getConfigObjectWithConfigFile(t, "testdata/valid_config.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.True(t, mountConfig.Write.CreateEmptyFile)
	assert.Equal(t, cfg.LogSeverity("ERROR"), mountConfig.Logging.Severity)
	assert.Equal(t, cfg.ResolvedPath("/tmp/logfile.json"), mountConfig.Logging.FilePath)
	assert.Equal(t, "text", mountConfig.Logging.Format)

	// log-rotate config
	assert.Equal(t, int64(100), mountConfig.Logging.LogRotate.MaxFileSizeMb)
	assert.Equal(t, int64(5), mountConfig.Logging.LogRotate.BackupFileCount)
	assert.False(t, mountConfig.Logging.LogRotate.Compress)

	// metadata-cache config
	assert.Equal(t, int64(5), mountConfig.MetadataCache.TtlSecs)
	assert.Equal(t, int64(1), mountConfig.MetadataCache.TypeCacheMaxSizeMb)
	assert.Equal(t, int64(3), mountConfig.MetadataCache.StatCacheMaxSizeMb)

	// list config
	assert.True(t, mountConfig.List.EnableEmptyManagedFolders)

	// auth config
	assert.True(t, mountConfig.GcsAuth.AnonymousAccess)

	// enable-hns
	assert.True(t, bool(mountConfig.EnableHns))

	// file-system config
	assert.True(t, mountConfig.FileSystem.IgnoreInterrupts)
	assert.True(t, mountConfig.FileSystem.DisableParallelDirops)

	// file-cache config
	assert.Equal(t, int64(100), mountConfig.FileCache.MaxSizeMb)
	assert.True(t, mountConfig.FileCache.CacheFileForRangeRead)
	assert.True(t, mountConfig.FileCache.EnableParallelDownloads)
	assert.Equal(t, int64(10), mountConfig.FileCache.ParallelDownloadsPerFile)
	assert.Equal(t, int64(-1), mountConfig.FileCache.MaxParallelDownloads)
	assert.Equal(t, int64(100), mountConfig.FileCache.DownloadChunkSizeMb)
	assert.False(t, mountConfig.FileCache.EnableCrc)
}

func TestReadConfigFile_InvalidLogConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/invalid_log_config.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidLogRotateConfig1(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/invalid_log_rotate_config_1.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidLogRotateConfig2(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/invalid_log_rotate_config_2.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidFileCacheMaxSizeConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/file_cache_config/invalid_max_size_mb.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidMaxParallelDownloadsConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/file_cache_config/invalid_max_parallel_downloads.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidZeroMaxParallelDownloadsConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/file_cache_config/invalid_zero_max_parallel_downloads.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidParallelDownloadsPerFileConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/file_cache_config/invalid_parallel_downloads_per_file.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_InvalidDownloadChunkSizeMBConfig(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/file_cache_config/invalid_download_chunk_size_mb.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_MetatadaCacheConfig_InvalidTTL(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/metadata_cache_config_invalid_ttl.yaml")

	assert.Error(t, err)
}

func TestReadConfigFile_MetatadaCacheConfig_TtlNotSet(t *testing.T) {
	_, err := getConfigObjectWithConfigFile(t, "testdata/metadata_cache_config_ttl-unset.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, TtlInSecsUnsetSentinel, mountConfig.MetadataCache.TtlInSeconds)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TtlTooHigh() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_ttl_too_high.yaml")

	assert.ErrorContains(t, err, MetadataCacheTtlSecsTooHighError)
}

/*
func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidTypeCacheMaxSize() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_type-cache-max-size-mb.yaml")

	assert.ErrorContains(t, err, TypeCacheMaxSizeMBInvalidValueError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TypeCacheMaxSizeNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_type-cache-max-size-mb_unset.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, DefaultTypeCacheMaxSizeMB, mountConfig.MetadataCache.TypeCacheMaxSizeMB)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidStatCacheSize() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_stat-cache-max-size-mb.yaml")

	assert.ErrorContains(t, err, StatCacheMaxSizeMBInvalidValueError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_StatCacheSizeNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_stat-cache-max-size-mb_unset.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, StatCacheMaxSizeMBUnsetSentinel, mountConfig.MetadataCache.StatCacheMaxSizeMB)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_StatCacheSizeTooHigh() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_stat-cache-max-size-mb_too_high.yaml")

	assert.ErrorContains(t, err, StatCacheMaxSizeMBTooHighError)
}

func (t *YamlParserTest) TestReadConfigFile_GrpcClientConfig_invalidConnPoolSize() {
	_, err := ParseConfigFile("testdata/gcs_connection/invalid_conn_pool_size.yaml")

	assert.ErrorContains(t, err, "error parsing gcs-connection configs: the value of conn-pool-size can't be less than 1")
}

func (t *YamlParserTest) TestReadConfigFile_GrpcClientConfig_unsetConnPoolSize() {
	mountConfig, err := ParseConfigFile("testdata/gcs_connection/unset_conn_pool_size.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, DefaultGrpcConnPoolSize, mountConfig.GCSConnection.GRPCConnPoolSize)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidIgnoreInterruptsValue() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_ignore_interrupts.yaml")

	assert.ErrorContains(t, err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!str `abc` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetIgnoreInterruptsValue() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_ignore_interrupts.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.True(t, mountConfig.FileSystem.IgnoreInterrupts)
}

func (t *YamlParserTest) TestReadConfigFile_GCSAuth_InvalidAnonymousAccessValue() {
	_, err := ParseConfigFile("testdata/gcs_auth/invalid_anonymous_access.yaml")

	assert.ErrorContains(t, err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!str `abc` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_GCSAuth_UnsetAnonymousAccessValue() {
	mountConfig, err := ParseConfigFile("testdata/gcs_auth/unset_anonymous_access.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.False(t, mountConfig.GcsAuth.AnonymousAccess)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidDisableParallelDirops() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_disable_parallel_dirops.yaml")

	assert.ErrorContains(t, err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!int `-1` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetDisableParallelDirops() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_disable_parallel_dirops.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.False(t, mountConfig.FileSystem.DisableParallelDirops)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidKernelListCacheTtl() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_kernel_list_cache_ttl.yaml")

	assert.ErrorContains(t, err, fmt.Sprintf("invalid kernelListCacheTtlSecs: %s", TtlInSecsInvalidValueError))
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsupportedLargeKernelListCacheTtl() {
	_, err := ParseConfigFile("testdata/file_system_config/unsupported_large_kernel_list_cache_ttl.yaml")

	assert.ErrorContains(t, err, fmt.Sprintf("invalid kernelListCacheTtlSecs: %s", TtlInSecsTooHighError))
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetKernelListCacheTtl() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_kernel_list_cache_ttl.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, DefaultKernelListCacheTtlSeconds, mountConfig.FileSystem.KernelListCacheTtlSeconds)
}

func TestReadConfigFile_FileSystemConfig_ValidKernelListCacheTtl(t *testing.T) {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/valid_kernel_list_cache_ttl.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, mountConfig)
	assert.Equal(t, int64(10), mountConfig.FileSystem.KernelListCacheTtlSeconds)
}
*/

// Copyright 2021 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type YamlParserTest struct {
	suite.Suite
}

func TestYamlParserSuite(t *testing.T) {
	suite.Run(t, new(YamlParserTest))
}
func validateDefaultConfig(t *testing.T, mountConfig *MountConfig) {
	assert.NotNil(t, mountConfig)
	assert.False(t, mountConfig.CreateEmptyFile)
	assert.False(t, mountConfig.ListConfig.EnableEmptyManagedFolders)
	assert.Equal(t, "INFO", string(mountConfig.LogConfig.Severity))
	assert.Equal(t, "", mountConfig.LogConfig.Format)
	assert.Equal(t, "", mountConfig.LogConfig.FilePath)
	assert.Equal(t, 512, mountConfig.LogConfig.LogRotateConfig.MaxFileSizeMB)
	assert.Equal(t, 10, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	assert.True(t, bool(mountConfig.LogConfig.LogRotateConfig.Compress))
	assert.Equal(t, "", string(mountConfig.CacheDir))
	assert.Equal(t, int64(-1), mountConfig.FileCacheConfig.MaxSizeMB)
	assert.False(t, mountConfig.FileCacheConfig.CacheFileForRangeRead)
	assert.Equal(t, 1, mountConfig.GrpcClientConfig.ConnPoolSize)
	assert.False(t, mountConfig.AuthConfig.AnonymousAccess)
	assert.False(t, bool(mountConfig.EnableHNS))
	assert.False(t, mountConfig.FileSystemConfig.IgnoreInterrupts)
	assert.False(t, mountConfig.FileSystemConfig.DisableParallelDirops)
	assert.Equal(t, DefaultKernelDirCacheTtlInSeconds, mountConfig.KernelDirCacheTtlInSeconds)
}

func (t *YamlParserTest) TestReadConfigFile_EmptyFileName() {
	mountConfig, err := ParseConfigFile("")

	assert.NoError(t.T(), err)
	validateDefaultConfig(t.T(), mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_EmptyFile() {
	mountConfig, err := ParseConfigFile("testdata/empty_file.yaml")

	assert.NoError(t.T(), err)
	validateDefaultConfig(t.T(), mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_NonExistingFile() {
	_, err := ParseConfigFile("testdata/nofile.yaml")

	assert.ErrorContains(t.T(), err, "error reading config file: open testdata/nofile.yaml: no such file or directory")
}

func (t *YamlParserTest) TestReadConfigFile_InvalidConfig() {
	_, err := ParseConfigFile("testdata/invalid_config.yaml")

	assert.ErrorContains(t.T(), err, "error parsing config file: yaml: unmarshal errors:")
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfigWith0BackupFileCount() {
	mountConfig, err := ParseConfigFile("testdata/valid_config_with_0_backup-file-count.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.True(t.T(), mountConfig.WriteConfig.CreateEmptyFile)
	assert.Equal(t.T(), ERROR, mountConfig.LogConfig.Severity)
	assert.Equal(t.T(), "/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	assert.Equal(t.T(), "text", mountConfig.LogConfig.Format)
	assert.Equal(t.T(), 0, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	assert.False(t.T(), mountConfig.LogConfig.LogRotateConfig.Compress)
}

func (t *YamlParserTest) TestReadConfigFile_Invalid_UnexpectedField_Config() {
	_, err := ParseConfigFile("testdata/invalid_unexpectedfield_config.yaml")

	assert.ErrorContains(t.T(), err, "error parsing config file: yaml: unmarshal errors:")
	assert.ErrorContains(t.T(), err, "line 5: field formats not found in type config.LogConfig")
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfig() {
	mountConfig, err := ParseConfigFile("testdata/valid_config.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.True(t.T(), mountConfig.WriteConfig.CreateEmptyFile)
	assert.Equal(t.T(), ERROR, mountConfig.LogConfig.Severity)
	assert.Equal(t.T(), "/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	assert.Equal(t.T(), "text", mountConfig.LogConfig.Format)

	// log-rotate config
	assert.Equal(t.T(), 100, mountConfig.LogConfig.LogRotateConfig.MaxFileSizeMB)
	assert.Equal(t.T(), 5, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	assert.False(t.T(), mountConfig.LogConfig.LogRotateConfig.Compress)

	// metadata-cache config
	assert.Equal(t.T(), int64(5), mountConfig.MetadataCacheConfig.TtlInSeconds)
	assert.Equal(t.T(), 1, mountConfig.MetadataCacheConfig.TypeCacheMaxSizeMB)
	assert.Equal(t.T(), int64(3), mountConfig.MetadataCacheConfig.StatCacheMaxSizeMB)

	// list config
	assert.True(t.T(), mountConfig.ListConfig.EnableEmptyManagedFolders)

	// auth config
	assert.True(t.T(), mountConfig.AuthConfig.AnonymousAccess)

	// enable-hns
	assert.True(t.T(), bool(mountConfig.EnableHNS))

	// file-system config
	assert.True(t.T(), mountConfig.FileSystemConfig.IgnoreInterrupts)
	assert.True(t.T(), mountConfig.FileSystemConfig.DisableParallelDirops)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogConfig() {
	_, err := ParseConfigFile("testdata/invalid_log_config.yaml")

	assert.ErrorContains(t.T(), err, fmt.Sprintf(parseConfigFileErrMsgFormat, "log severity should be one of [trace, debug, info, warning, error, off]"))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogRotateConfig1() {
	_, err := ParseConfigFile("testdata/invalid_log_rotate_config_1.yaml")

	assert.ErrorContains(t.T(), err, fmt.Sprintf(parseConfigFileErrMsgFormat, "max-file-size-mb should be atleast 1"))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogRotateConfig2() {
	_, err := ParseConfigFile("testdata/invalid_log_rotate_config_2.yaml")

	assert.ErrorContains(t.T(), err, fmt.Sprintf(parseConfigFileErrMsgFormat, "backup-file-count should be 0 (to retain all backup files) or a positive value"))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidFileCacheMaxSizeConfig() {
	_, err := ParseConfigFile("testdata/invalid_filecachesize_config.yaml")

	assert.ErrorContains(t.T(), err, "error parsing file-cache configs: the value of max-size-mb for file-cache can't be less than -1")
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidTTL() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_ttl.yaml")

	assert.ErrorContains(t.T(), err, MetadataCacheTtlSecsInvalidValueError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TtlNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_ttl-unset.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.Equal(t.T(), TtlInSecsUnsetSentinel, mountConfig.MetadataCacheConfig.TtlInSeconds)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TtlTooHigh() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_ttl_too_high.yaml")

	assert.ErrorContains(t.T(), err, MetadataCacheTtlSecsTooHighError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidTypeCacheMaxSize() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_type-cache-max-size-mb.yaml")

	assert.ErrorContains(t.T(), err, TypeCacheMaxSizeMBInvalidValueError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TypeCacheMaxSizeNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_type-cache-max-size-mb_unset.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.Equal(t.T(), DefaultTypeCacheMaxSizeMB, mountConfig.MetadataCacheConfig.TypeCacheMaxSizeMB)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidStatCacheSize() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_stat-cache-max-size-mb.yaml")

	assert.ErrorContains(t.T(), err, StatCacheMaxSizeMBInvalidValueError)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_StatCacheSizeNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_stat-cache-max-size-mb_unset.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.Equal(t.T(), StatCacheMaxSizeMBUnsetSentinel, mountConfig.MetadataCacheConfig.StatCacheMaxSizeMB)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_StatCacheSizeTooHigh() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_stat-cache-max-size-mb_too_high.yaml")

	assert.ErrorContains(t.T(), err, StatCacheMaxSizeMBTooHighError)
}

func (t *YamlParserTest) TestReadConfigFile_GrpcClientConfig_invalidConnPoolSize() {
	_, err := ParseConfigFile("testdata/grpc_client_config/invalid_conn_pool_size.yaml")

	assert.ErrorContains(t.T(), err, "error parsing grpc-config: the value of conn-pool-size can't be less than 1")
}

func (t *YamlParserTest) TestReadConfigFile_GrpcClientConfig_unsetConnPoolSize() {
	mountConfig, err := ParseConfigFile("testdata/grpc_client_config/unset_conn_pool_size.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.Equal(t.T(), DefaultGrpcConnPoolSize, mountConfig.GrpcClientConfig.ConnPoolSize)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidIgnoreInterruptsValue() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_ignore_interrupts.yaml")

	assert.ErrorContains(t.T(), err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!str `abc` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetIgnoreInterruptsValue() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_ignore_interrupts.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.Equal(t.T(), false, mountConfig.FileSystemConfig.IgnoreInterrupts)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidAnonymousAccessValue() {
	_, err := ParseConfigFile("testdata/auth_config/invalid_anonymous_access.yaml")

	assert.ErrorContains(t.T(), err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!str `abc` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetAnonymousAccessValue() {
	mountConfig, err := ParseConfigFile("testdata/auth_config/unset_anonymous_access.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.False(t.T(), mountConfig.AuthConfig.AnonymousAccess)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidDisableParallelDirops() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_disable_parallel_dirops.yaml")

	assert.ErrorContains(t.T(), err, "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!int `-1` into bool")
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetDisableParallelDirops() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_disable_parallel_dirops.yaml")

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), mountConfig)
	assert.False(t.T(), mountConfig.FileSystemConfig.DisableParallelDirops)
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_InvalidKernelDirCacheTtl() {
	_, err := ParseConfigFile("testdata/file_system_config/invalid_kernel_dir_cache_ttl.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing config file: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!str `invalid` into int64"))
}

func (t *YamlParserTest) TestReadConfigFile_FileSystemConfig_UnsetKernelDirCacheTtl() {
	mountConfig, err := ParseConfigFile("testdata/file_system_config/unset_kernel_dir_cache_ttl.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(DefaultKernelDirCacheTtlInSeconds, mountConfig.FileSystemConfig.KernelDirCacheTtlInSeconds)
}

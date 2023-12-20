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
	"strings"
	"testing"

	"github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestYamlParser(t *testing.T) { RunTests(t) }

type YamlParserTest struct {
}

func init() { RegisterTestSuite(&YamlParserTest{}) }

func validateDefaultConfig(mountConfig *MountConfig) {
	AssertNe(nil, mountConfig)
	ExpectEq(false, mountConfig.CreateEmptyFile)
	ExpectEq("INFO", mountConfig.LogConfig.Severity)
	ExpectEq("", mountConfig.LogConfig.Format)
	ExpectEq("", mountConfig.LogConfig.FilePath)
	ExpectEq(512, mountConfig.LogConfig.LogRotateConfig.MaxFileSizeMB)
	ExpectEq(10, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	ExpectEq(true, mountConfig.LogConfig.LogRotateConfig.Compress)
	ExpectEq("", mountConfig.CacheLocation)
	ExpectEq(0, mountConfig.FileCacheConfig.MaxSizeInMB)
	ExpectEq(false, mountConfig.FileCacheConfig.CacheFileForRangeRead)
}

func (t *YamlParserTest) TestReadConfigFile_EmptyFileName() {
	mountConfig, err := ParseConfigFile("")

	AssertEq(nil, err)
	validateDefaultConfig(mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_EmptyFile() {
	mountConfig, err := ParseConfigFile("testdata/empty_file.yaml")

	AssertEq(nil, err)
	validateDefaultConfig(mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_NonExistingFile() {
	_, err := ParseConfigFile("testdata/nofile.yaml")

	AssertNe(nil, err)
	AssertEq("error reading config file: open testdata/nofile.yaml: no such file or directory", err.Error())
}

func (t *YamlParserTest) TestReadConfigFile_InvalidConfig() {
	_, err := ParseConfigFile("testdata/invalid_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing config file: yaml: unmarshal errors:"))
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfigWith0BackupFileCount() {
	mountConfig, err := ParseConfigFile("testdata/valid_config_with_0_backup-file-count.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	ExpectEq(true, mountConfig.WriteConfig.CreateEmptyFile)
	ExpectEq(ERROR, mountConfig.LogConfig.Severity)
	ExpectEq("/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	ExpectEq("text", mountConfig.LogConfig.Format)
	ExpectEq(100, mountConfig.LogConfig.LogRotateConfig.MaxFileSizeMB)
	ExpectEq(0, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	ExpectEq(false, mountConfig.LogConfig.LogRotateConfig.Compress)
}

func (t *YamlParserTest) TestReadConfigFile_Invalid_UnexpectedField_Config() {
	_, err := ParseConfigFile("testdata/invalid_unexpectedfield_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing config file: yaml: unmarshal errors:"))
	AssertTrue(strings.Contains(err.Error(), "line 5: field formats not found in type config.LogConfig"))
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfig() {
	mountConfig, err := ParseConfigFile("testdata/valid_config.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	ExpectEq(true, mountConfig.WriteConfig.CreateEmptyFile)
	ExpectEq(ERROR, mountConfig.LogConfig.Severity)
	ExpectEq("/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	ExpectEq("text", mountConfig.LogConfig.Format)

	// log-rotate config
	ExpectEq(100, mountConfig.LogConfig.LogRotateConfig.MaxFileSizeMB)
	ExpectEq(5, mountConfig.LogConfig.LogRotateConfig.BackupFileCount)
	ExpectEq(false, mountConfig.LogConfig.LogRotateConfig.Compress)

	// metadata-cache config
	ExpectEq(5, mountConfig.MetadataCacheConfig.TtlInSeconds)
	ExpectEq(1, mountConfig.MetadataCacheConfig.TypeCacheMaxSizeMbPerDirectory)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogConfig() {
	_, err := ParseConfigFile("testdata/invalid_log_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(),
		fmt.Sprintf(parseConfigFileErrMsgFormat, "log severity should be one of [trace, debug, info, warning, error, off]")))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogRotateConfig1() {
	_, err := ParseConfigFile("testdata/invalid_log_rotate_config_1.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(),
		fmt.Sprintf(parseConfigFileErrMsgFormat, "max-file-size-mb should be atleast 1")))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogRotateConfig2() {
	_, err := ParseConfigFile("testdata/invalid_log_rotate_config_2.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(),
		fmt.Sprintf(parseConfigFileErrMsgFormat, "backup-file-count should be 0 (to retain all backup files) or a positive value")))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidFileCacheMaxSizeConfig() {
	_, err := ParseConfigFile("testdata/invalid_filecachesize_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing file-cache configs: the value of max-size-in-mb for file-cache can't be less than -1"))
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidTTL() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_ttl.yaml")

	AssertNe(nil, err)
	AssertThat(err, oglematchers.Error(oglematchers.HasSubstr(MetadataCacheTtlSecsInvalidValueError)))
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TtlNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_ttl-unset.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(TtlInSecsUnsetSentinel, mountConfig.MetadataCacheConfig.TtlInSeconds)
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_InvalidTypeCacheSize() {
	_, err := ParseConfigFile("testdata/metadata_cache_config_invalid_type-cache-size.yaml")

	AssertNe(nil, err)
	AssertThat(err, oglematchers.Error(oglematchers.HasSubstr(MetadataCacheTtlSecsInvalidValueError)))
}

func (t *YamlParserTest) TestReadConfigFile_MetatadaCacheConfig_TypeCacheSizeNotSet() {
	mountConfig, err := ParseConfigFile("testdata/metadata_cache_config_type-cache-size_unset.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(TypeCacheMaxSizeInMbPerDirectoryUnsetSentinel, mountConfig.MetadataCacheConfig.TypeCacheMaxSizeMbPerDirectory)
}

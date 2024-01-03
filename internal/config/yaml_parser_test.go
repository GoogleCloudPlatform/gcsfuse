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
	AssertEq(false, mountConfig.CreateEmptyFile)
	AssertEq("INFO", mountConfig.LogConfig.Severity)
	AssertEq("", mountConfig.LogConfig.Format)
	AssertEq("", mountConfig.LogConfig.FilePath)
	AssertEq("", mountConfig.CacheLocation)
	AssertEq(0, mountConfig.FileCacheConfig.MaxSizeInMB)
	AssertEq(false, mountConfig.FileCacheConfig.CacheFileForRangeRead)
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
	AssertEq(true, mountConfig.WriteConfig.CreateEmptyFile)
	AssertEq(ERROR, mountConfig.LogConfig.Severity)
	AssertEq("/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	AssertEq("text", mountConfig.LogConfig.Format)

	// metadata-cache config
	AssertEq(5, mountConfig.MetadataCacheConfig.TtlInSeconds)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidValidLogConfig() {
	_, err := ParseConfigFile("testdata/invalid_log_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing config file: log severity should be one of [trace, debug, info, warning, error, off]"))
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

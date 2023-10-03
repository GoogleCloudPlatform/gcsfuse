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

	. "github.com/jacobsa/ogletest"
)

func TestYamlParser(t *testing.T) { RunTests(t) }

type YamlParserTest struct {
}

func init() { RegisterTestSuite(&YamlParserTest{}) }

func validateDefaultConfig(mountConfig *MountConfig) {
	AssertNe(nil, mountConfig)
	AssertEq(false, mountConfig.WriteConfig.CreateEmptyFile)
	AssertEq(false, mountConfig.WriteConfig.EnableStreamingWrites)
	AssertEq(0, mountConfig.WriteConfig.BufferSizeMB)
	AssertEq("INFO", mountConfig.LogConfig.Severity)
	AssertEq("", mountConfig.LogConfig.Format)
	AssertEq("", mountConfig.LogConfig.FilePath)
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

func (t *YamlParserTest) TestReadConfigFile_ValidConfig() {
	mountConfig, err := ParseConfigFile("testdata/valid_config.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(true, mountConfig.WriteConfig.CreateEmptyFile)
	AssertEq(true, mountConfig.WriteConfig.EnableStreamingWrites)
	AssertEq(150, mountConfig.WriteConfig.BufferSizeMB)
	AssertEq(ERROR, mountConfig.LogConfig.Severity)
	AssertEq("/tmp/logfile.json", mountConfig.LogConfig.FilePath)
	AssertEq("text", mountConfig.LogConfig.Format)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidLogConfig() {
	_, err := ParseConfigFile("testdata/invalid_log_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing logging configs: log severity should be one of [trace, debug, info, warning, error, off]"))
}

func (t *YamlParserTest) TestReadConfigFile_InvalidBufferSizeConfig() {
	_, err := ParseConfigFile("testdata/invalid_write_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing write configs: buffer-size-mb should be greater than 0 if enable-streaming-writes is enabled"))
}

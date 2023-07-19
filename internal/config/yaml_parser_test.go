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
	AssertEq(true, mountConfig.CreateEmptyFile)
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
}

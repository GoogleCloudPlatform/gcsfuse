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

func (t *YamlParserTest) TestReadConfigFile_EmptyFileName() {
	mountConfig, err := ReadConfigFile("")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_EmptyFile() {
	mountConfig, err := ReadConfigFile("testdata/empty_file.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidFile() {
	_, err := ReadConfigFile("testdata/invalid_file.yaml")

	AssertNe(nil, err)
	AssertEq("error reading config file: open testdata/invalid_file.yaml: no such file or directory", err.Error())
}

func (t *YamlParserTest) TestReadConfigFile_InvalidConfig() {
	_, err := ReadConfigFile("testdata/invalid_config.yaml")

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error parsing config file: yaml: unmarshal errors:"))
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfig() {
	mountConfig, err := ReadConfigFile("testdata/valid_config.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(true, mountConfig.WriteConfig.CreateEmptyFile)
}

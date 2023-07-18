package config

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestYamlParser(t *testing.T) { RunTests(t) }

type YamlParserTest struct {
}

func init() { RegisterTestSuite(&YamlParserTest{}) }

func (t *YamlParserTest) TestReadConfigFile_EmptyFile() {
	mountConfig, err := ReadConfigFile("testdata/empty_file.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidFile() {
	_, err := ReadConfigFile("testdata/invalid_file.yaml")

	AssertNe(nil, err)
}

func (t *YamlParserTest) TestReadConfigFile_InvalidConfig() {
	_, err := ReadConfigFile("testdata/invalid_config.yaml")

	AssertNe(nil, err)
}

func (t *YamlParserTest) TestReadConfigFile_ValidConfig() {
	mountConfig, err := ReadConfigFile("testdata/valid_config.yaml")

	AssertEq(nil, err)
	AssertNe(nil, mountConfig)
	AssertEq(true, mountConfig.WriteConfig.CreateEmptyFile)
}

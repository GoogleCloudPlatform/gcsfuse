package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	. "github.com/jacobsa/ogletest"
)

func TestConfig(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ConfigTest struct {
}

func init() { RegisterTestSuite(&ConfigTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConfigTest) TestOverrideLoggingFlags_WithNonEmptyLogConfigs() {
	flags := &flagStorage{
		LogFile:   "a.txt",
		LogFormat: "json",
		DebugFuse: true,
		DebugGCS:  false,
	}
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		Severity: config.ERROR,
		File:     "/tmp/hello.txt",
		Format:   "text",
	}
	mountConfig.WriteConfig = config.WriteConfig{
		CreateEmptyFile: true,
	}

	overrideWithLoggingFlags(mountConfig, flags)

	AssertEq(mountConfig.LogConfig.Format, "text")
	AssertEq(mountConfig.LogConfig.File, "/tmp/hello.txt")
	AssertEq(mountConfig.LogConfig.Severity, config.TRACE)
}

func (t *ConfigTest) TestOverrideLoggingFlags_WithEmptyLogConfigs() {
	flags := &flagStorage{
		LogFile:   "a.txt",
		LogFormat: "json",
	}
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		Severity: config.INFO,
		File:     "",
		Format:   "",
	}
	mountConfig.WriteConfig = config.WriteConfig{
		CreateEmptyFile: true,
	}

	overrideWithLoggingFlags(mountConfig, flags)

	AssertEq(mountConfig.LogConfig.Format, "json")
	AssertEq(mountConfig.LogConfig.File, "a.txt")
	AssertEq(mountConfig.LogConfig.Severity, config.INFO)
}

func (t *ConfigTest) TestResolveConfigFilePaths() {
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		File: "~/test.txt",
	}

	err := resolveConfigFilePaths(mountConfig)

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), mountConfig.LogConfig.File)
}

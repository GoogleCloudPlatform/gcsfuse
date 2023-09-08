package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	. "github.com/jacobsa/ogletest"
)

func Test_Main(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
}

func init() { RegisterTestSuite(&MainTest{}) }

func (t *MainTest) TestCreateStorageHandle() {
	flags := &flagStorage{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 100,
		HttpClientTimeout:   5,
		MaxRetryDuration:    7,
		RetryMultiplier:     2,
		AppName:             "app",
		KeyFile:             "testdata/test_creds.json",
	}

	storageHandle, err := createStorageHandle(flags)

	AssertEq(nil, err)
	AssertNe(nil, storageHandle)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	userAgent := getUserAgent("AppName")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s AppName (GPN:gcsfuse-DLVM)", getVersion()))

	ExpectEq(expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	userAgent := getUserAgent("AppName")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-AppName)", getVersion()))

	ExpectEq(expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarAndAppNameAreNotSet() {
	userAgent := getUserAgent("")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse)", getVersion()))

	ExpectEq(expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarSetAndAppNameNotSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	userAgent := getUserAgent("")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s (GPN:gcsfuse-DLVM)", getVersion()))

	ExpectEq(expectedUserAgent, userAgent)
}

func (t *MainTest) TestOverrideLoggingFlags_WithNonEmptyLogConfigs() {
	flags := &flagStorage{
		LogFile:   "a.txt",
		LogFormat: "json",
		DebugFuse: true,
		DebugGCS:  false,
	}
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		Severity:  config.ERROR,
		LogFile:   "/tmp/hello.txt",
		LogFormat: "text",
	}
	mountConfig.WriteConfig = config.WriteConfig{
		CreateEmptyFile: true,
	}

	overrideWithLoggingFlags(mountConfig, flags)

	AssertEq(mountConfig.LogFormat, "text")
	AssertEq(mountConfig.LogFile, "/tmp/hello.txt")
	AssertEq(mountConfig.LogConfig.Severity, config.TRACE)
}

func (t *MainTest) TestOverrideLoggingFlags_WithEmptyLogConfigs() {
	flags := &flagStorage{
		LogFile:   "a.txt",
		LogFormat: "json",
	}
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		Severity:  config.INFO,
		LogFile:   "",
		LogFormat: "",
	}
	mountConfig.WriteConfig = config.WriteConfig{
		CreateEmptyFile: true,
	}

	overrideWithLoggingFlags(mountConfig, flags)

	AssertEq(mountConfig.LogFormat, "json")
	AssertEq(mountConfig.LogFile, "a.txt")
	AssertEq(mountConfig.Severity, config.INFO)
}

func (t *MainTest) TestResolveConfigFilePaths() {
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		LogFile: "~/test.txt",
	}

	err := resolveConfigFilePaths(mountConfig)

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), mountConfig.LogFile)
}

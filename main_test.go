package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/util"

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
		MaxRetrySleep:       7,
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

func (t *MainTest) TestStringifyShouldReturnAllFlagsPassedInMountConfigAsMarshalledString() {
	mountConfig := &config.MountConfig{
		WriteConfig: config.WriteConfig{
			CreateEmptyFile: false,
		},
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
			FilePath: "\"path\"to\"file\"",
			LogRotateConfig: config.LogRotateConfig{
				MaxFileSizeMB:   2,
				BackupFileCount: 2,
				Compress:        true,
			},
		},
	}

	actual := util.Stringify(mountConfig)

	expected := "{\"CreateEmptyFile\":false,\"Severity\":\"TRACE\",\"Format\":\"\",\"FilePath\":\"\\\"path\\\"to\\\"file\\\"\",\"LogRotateConfig\":{\"MaxFileSizeMB\":2,\"BackupFileCount\":2,\"Compress\":true},\"MaxSizeInMB\":0,\"CacheFileForRangeRead\":false,\"CacheLocation\":\"\",\"TtlInSeconds\":0,\"TypeCacheMaxSizeMb\":0}"
	AssertEq(expected, actual)
}

func (t *MainTest) TestStringifyShouldReturnAllFlagsPassedInFlagStorageAsMarshalledString() {
	mountOptions := map[string]string{
		"1": "one",
		"2": "two",
		"3": "three",
	}
	flags := &flagStorage{
		SequentialReadSizeMb: 10,
		ClientProtocol:       mountpkg.ClientProtocol("http4"),
		MountOptions:         mountOptions,
	}

	actual := util.Stringify(flags)

	expected := "{\"AppName\":\"\",\"Foreground\":false,\"ConfigFile\":\"\",\"MountOptions\":{\"1\":\"one\",\"2\":\"two\",\"3\":\"three\"},\"DirMode\":0,\"FileMode\":0,\"Uid\":0,\"Gid\":0,\"ImplicitDirs\":false,\"OnlyDir\":\"\",\"RenameDirLimit\":0,\"CustomEndpoint\":null,\"BillingProject\":\"\",\"KeyFile\":\"\",\"TokenUrl\":\"\",\"ReuseTokenFromUrl\":false,\"EgressBandwidthLimitBytesPerSecond\":0,\"OpRateLimitHz\":0,\"SequentialReadSizeMb\":10,\"MaxRetrySleep\":0,\"StatCacheCapacity\":0,\"StatOrTypeCacheTTL\":0,\"HttpClientTimeout\":0,\"MaxRetryDuration\":0,\"RetryMultiplier\":0,\"LocalFileCache\":false,\"TempDir\":\"\",\"ClientProtocol\":\"http4\",\"MaxConnsPerHost\":0,\"MaxIdleConnsPerHost\":0,\"EnableNonexistentTypeCache\":false,\"StackdriverExportInterval\":0,\"OtelCollectorAddress\":\"\",\"LogFile\":\"\",\"LogFormat\":\"\",\"ExperimentalEnableJsonRead\":false,\"DebugFuseErrors\":false,\"DebugFuse\":false,\"DebugFS\":false,\"DebugGCS\":false,\"DebugHTTP\":false,\"DebugInvariants\":false,\"DebugMutex\":false}"
	AssertEq(expected, actual)
}

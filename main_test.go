package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

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

func (t *MainTest) TestCreateStorageHandleEnableStorageClientLibraryIsTrue() {
	storageHandle, err := createStorageHandle(&flagStorage{
		EnableStorageClientLibrary: true,
		KeyFile:                    "testdata/test_creds.json",
	})

	ExpectNe(nil, storageHandle)
	ExpectEq(nil, err)
}

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

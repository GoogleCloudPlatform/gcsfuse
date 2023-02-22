package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

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
	// Creating fake credential to pass as key file
	CreateFakeCreds("creds.json")

	storageHandle, err := createStorageHandle(&flagStorage{
		EnableStorageClientLibrary: true,
		KeyFile:                    "creds.json",
	})

	ExpectNe(nil, storageHandle)
	ExpectEq(nil, err)

	// Removing creds.json file
	RemoveFakeCreds("creds.json")
}

func (t *MainTest) TestCreateStorageHandle() {
	// Creating fake credential to pass as key file
	CreateFakeCreds("creds.json")

	flags := &flagStorage{
		DisableHTTP2:        false,
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 100,
		MaxRetryDuration:    7,
		RetryMultiplier:     2,
		AppName:             "app",
		KeyFile:             "creds.json",
	}

	storageHandle, err := createStorageHandle(flags)

	AssertEq(nil, err)
	AssertNe(nil, storageHandle)

	// Removing creds.json file
	RemoveFakeCreds("creds.json")
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	userAgent := getUserAgent("AppName")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s %s %s", getVersion(), "AppName", os.Getenv("GCSFUSE_METADATA_IMAGE_TYPE")))

	ExpectEq(expectedUserAgent, userAgent)
}

func (t *MainTest) TestGetUserAgentWhenMetadataImageTypeEnvVarIsNotSet() {
	userAgent := getUserAgent("AppName")
	expectedUserAgent := strings.TrimSpace(fmt.Sprintf("gcsfuse/%s %s", getVersion(), "AppName"))

	ExpectEq(expectedUserAgent, userAgent)
}

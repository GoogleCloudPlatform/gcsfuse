package main

import (
	"os"
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
	storageHandle, err := createStorageHandle(&flagStorage{
		EnableStorageClientLibrary: true,
	})

	ExpectNe(nil, storageHandle)
	ExpectEq(nil, err)
}

func (t *MainTest) TestCreateStorageHandle() {
	flags := &flagStorage{
		DisableHTTP2:        false,
		MaxConnsPerHost:     5,
		MaxIdleConnsPerHost: 100,
		HttpClientTimeout:   5,
		MaxRetryDuration:    7,
		RetryMultiplier:     2,
		AppName:             "app",
	}

	storageHandle, err := createStorageHandle(flags)

	AssertEq(nil, err)
	AssertNe(nil, storageHandle)
}

func (t *MainTest) TestGetUserAgentWhereEnvironmentVariableIsSet() {
	os.Setenv("GCSFUSE_METADATA_IMAGE_TYPE", "DLVM")
	defer os.Unsetenv("GCSFUSE_METADATA_IMAGE_TYPE")

	userAgent := getUserAgent("AppName")

	ExpectEq("gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) AppName DLVM", userAgent)
}

func (t *MainTest) TestGetUserAgentWhereEnvironmentVariableIsNotSet() {
	userAgent := getUserAgent("AppName")

	ExpectEq("gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) AppName", userAgent)
}

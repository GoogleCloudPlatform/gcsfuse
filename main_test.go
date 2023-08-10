package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	. "github.com/jacobsa/ogletest"
)

func Test_Main(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
}

func init() { RegisterTestSuite(&MainTest{}) }

func (t *MainTest) TestHandleCustomEndpointWithProdGCSEndpoint() {
	url, err := url.Parse("https://storage.googleapis.com:443")
	AssertEq(nil, err)
	fs := flagStorage{
		KeyFile:  "testdata/test_creds.json",
		Endpoint: url,
	}
	storageClientConfig := storage.StorageClientConfig{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 100,
	}

	err = handleCustomEndpoint(&fs, &storageClientConfig)
	AssertEq(nil, err)

	ExpectNe(nil, &storageClientConfig.TokenSrc)
	ExpectEq(0, len(storageClientConfig.ClientOptions))
}

func (t *MainTest) TestHandleCustomEndpointWithNonProdGCSEndpoint() {
	url, err := url.Parse("http://localhost:443")
	AssertEq(nil, err)
	fs := flagStorage{
		KeyFile:  "testdata/test_creds.json",
		Endpoint: url,
	}
	storageClientConfig := storage.StorageClientConfig{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 100,
	}

	err = handleCustomEndpoint(&fs, &storageClientConfig)
	AssertEq(nil, err)

	ExpectNe(nil, &storageClientConfig.TokenSrc)
	ExpectEq(1, len(storageClientConfig.ClientOptions))
}

func (t *MainTest) TestHandleCustomEndpointWithNoEndpoint() {
	fs := flagStorage{
		KeyFile:  "testdata/test_creds.json",
		Endpoint: nil,
	}
	storageClientConfig := storage.StorageClientConfig{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 100,
	}

	err := handleCustomEndpoint(&fs, &storageClientConfig)
	AssertEq(nil, err)

	ExpectNe(nil, &storageClientConfig.TokenSrc)
	ExpectEq(0, len(storageClientConfig.ClientOptions))
}

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

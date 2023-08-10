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

const DummyKeyFile = "testdata/test_creds.json"

func Test_Main(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
}

func init() { RegisterTestSuite(&MainTest{}) }

func (t *MainTest) TestIsGCSProdHostnameWithProdHostName() {
	url, err := url.Parse(storage.ProdEndpoint)
	AssertEq(nil, err)

	res := isProdEndpoint(url)

	ExpectTrue(res)
}

func (t *MainTest) TestIsGCSProdHostnameWithCustomName() {
	url, err := url.Parse(storage.CustomEndpoint)
	AssertEq(nil, err)

	res := isProdEndpoint(url)

	ExpectFalse(res)
}

func (t *MainTest) TestCreateTokenSrcWithCustomEndpoint() {
	url, err := url.Parse(storage.CustomEndpoint)
	AssertEq(nil, err)

	fs := flagStorage{
		KeyFile:           DummyKeyFile,
		ClientProtocol:    mountpkg.HTTP1,
		HttpClientTimeout: 20,
		Endpoint:          url,
	}

	tokenSrc, err := createTokenSource(&fs)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *MainTest) TestCreateTokenSrcWithProdEndpoint() {
	url, err := url.Parse(storage.ProdEndpoint)
	AssertEq(nil, err)
	fs := flagStorage{
		KeyFile:           DummyKeyFile,
		ClientProtocol:    mountpkg.HTTP1,
		HttpClientTimeout: 20,
		Endpoint:          url,
	}

	tokenSrc, err := createTokenSource(&fs)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *MainTest) TestIsGCSProdHostnameWithNoEndpoint() {
	// if no url provided, it automatically start using prod GCS endpoint.
	var url *url.URL

	res := isProdEndpoint(url)

	ExpectTrue(res)
}

func (t *MainTest) TestCreateHttpClientObjWithHttp1() {
	fs := flagStorage{
		KeyFile:           DummyKeyFile,
		ClientProtocol:    mountpkg.HTTP1,
		HttpClientTimeout: 20,
	}

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := createHttpClientObj(&fs)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(20, httpClient.Timeout)
}

func (t *MainTest) TestCreateHttpClientObjWithHttp2() {
	fs := flagStorage{
		KeyFile:           DummyKeyFile,
		ClientProtocol:    mountpkg.HTTP2,
		HttpClientTimeout: 20,
	}

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := createHttpClientObj(&fs)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(20, httpClient.Timeout)
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

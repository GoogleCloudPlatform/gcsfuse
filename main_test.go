package main

import (
	"fmt"
	"log"
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
	c := "{\n  \"type\": \"service_account\",\n  \"project_id\":  \"test\",\n  \"private_key_id\":  \"test\",\n  \"private_key\":  \"test\",\n  \"client_email\":  \"test\",\n  \"client_id\":  \"test\",\n  \"auth_uri\":  \"test\",\n  \"token_uri\":  \"test\",\n  \"auth_provider_x509_cert_url\":  \"test\",\n  \"client_x509_cert_url\":  \"test\"\n}"
	f, err := os.Create("creds.json")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(c)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	storageHandle, err := createStorageHandle(&flagStorage{
		EnableStorageClientLibrary: true,
		KeyFile:                    "creds.json",
	})

	ExpectNe(nil, storageHandle)
	ExpectEq(nil, err)

	e := os.Remove("creds.json")
	if e != nil {
		log.Fatal(e)
	}
}

func (t *MainTest) TestCreateStorageHandle() {
	c := "{\n  \"type\": \"service_account\",\n  \"project_id\":  \"test\",\n  \"private_key_id\":  \"test\",\n  \"private_key\":  \"test\",\n  \"client_email\":  \"test\",\n  \"client_id\":  \"test\",\n  \"auth_uri\":  \"test\",\n  \"token_uri\":  \"test\",\n  \"auth_provider_x509_cert_url\":  \"test\",\n  \"client_x509_cert_url\":  \"test\"\n}"
	f, err := os.Create("creds.json")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(c)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

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

	e := os.Remove("creds.json")
	if e != nil {
		log.Fatal(e)
	}
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

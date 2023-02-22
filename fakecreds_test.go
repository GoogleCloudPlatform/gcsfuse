package main

import (
	"os"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func Test_FakeCreds(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FakeCredsTest struct {
}

func init() { RegisterTestSuite(&FakeCredsTest{}) }

func (t *FakeCredsTest) TestCreateFakeCredsAndRemoveFakeCreds() {
	expectedContent := "{\n  \"type\": \"service_account\"," +
			"\n  \"project_id\":  \"project_id\"," +
			"\n  \"private_key_id\":  \"private_key_id\"," +
			"\n  \"private_key\":  \"private_key\"," +
			"\n  \"client_email\":  \"client_email\"," +
			"\n  \"client_id\":  \"client_id\"," +
			"\n  \"auth_uri\":  \"auth_uri\"," +
			"\n  \"token_uri\":  \"token_uri\"," +
			"\n  \"auth_provider_x509_cert_url\":  \"auth_provider_x509_cert_url\"," +
			"\n  \"client_x509_cert_url\":  \"client_x509_cert_url\"\n}"

	err := CreateFakeCreds("creds.json")

	AssertEq(err, nil)
	f, err := os.Open("creds.json")
	AssertEq(nil, err)
	content := make([]byte, len(expectedContent))
	_, err = f.Read(content)
	AssertEq(err, nil)
	AssertEq(expectedContent, string(content[:]))

	err = RemoveFakeCreds("creds.json")

	AssertEq(err, nil)
}

func (t *FakeCredsTest) TestRemoveFakeCredsWhenFileNotExist() {
	// creds.json does not exist
	err := RemoveFakeCreds("creds.json")

	ExpectNe(nil, err)
}

package main

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

const TestBucketName string = "gcsfuse-default-bucket"

func TestMains(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MainTest struct {
}

func init() { RegisterTestSuite(&MainTest{}) }

func (t *MainTest) TestCreateStorageHandleEnableStorageClientLibraryIsTrue() {
	storageHandle, err := CreateStorageHandle(&flagStorage{
		EnableStorageClientLibrary: true,
	})
	ExpectNe(nil, storageHandle)
	ExpectEq(nil, err)
}

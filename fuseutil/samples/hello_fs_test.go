// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples_test

import (
	"io/ioutil"
	"testing"

	"github.com/jacobsa/gcsfuse/fuseutil"
	"github.com/jacobsa/gcsfuse/fuseutil/samples"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestHelloFS(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type HelloFSTest struct {
	mfs *fuseutil.MountedFileSystem
}

var _ SetUpInterface = &HelloFSTest{}

func init() { RegisterTestSuite(&HelloFSTest{}) }

func (t *HelloFSTest) SetUp(ti *TestInfo) {
	var err error
	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "hello_fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Mount a file system.
	fs := &samples.HelloFS{}
	if t.mfs, err = fuseutil.Mount(mountPoint, fs); err != nil {
		panic("Mount: " + err.Error())
	}

	if err = t.mfs.WaitForReady(context.Background()); err != nil {
		panic("MountedFileSystem.WaitForReady: " + err.Error())
	}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *HelloFSTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

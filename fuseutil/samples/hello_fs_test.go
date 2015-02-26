// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package samples_test

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jacobsa/gcsfuse/fuseutil"
	"github.com/jacobsa/gcsfuse/fuseutil/samples"
	"github.com/jacobsa/gcsfuse/timeutil"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestHelloFS(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type HelloFSTest struct {
	clock timeutil.SimulatedClock
	mfs   *fuseutil.MountedFileSystem
}

var _ SetUpInterface = &HelloFSTest{}
var _ TearDownInterface = &HelloFSTest{}

func init() { RegisterTestSuite(&HelloFSTest{}) }

func (t *HelloFSTest) SetUp(ti *TestInfo) {
	var err error

	// Set up a fixed, non-zero time.
	t.clock.AdvanceTime(time.Now().Sub(t.clock.Now()))

	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "hello_fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Mount a file system.
	fs := &samples.HelloFS{
		Clock: &t.clock,
	}

	if t.mfs, err = fuseutil.Mount(mountPoint, fs); err != nil {
		panic("Mount: " + err.Error())
	}

	if err = t.mfs.WaitForReady(context.Background()); err != nil {
		panic("MountedFileSystem.WaitForReady: " + err.Error())
	}
}

func (t *HelloFSTest) TearDown() {
	// Unmount the file system. Try again on "resource busy" errors.
	delay := 10 * time.Millisecond
	for {
		err := t.mfs.Unmount()
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		panic("MountedFileSystem.Unmount: " + err.Error())
	}

	if err := t.mfs.Join(context.Background()); err != nil {
		panic("MountedFileSystem.Join: " + err.Error())
	}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *HelloFSTest) ReadDir_Root() {
	entries, err := ioutil.ReadDir(t.mfs.Dir())

	AssertEq(nil, err)
	AssertEq(2, len(entries))
	var fi os.FileInfo

	// dir
	fi = entries[0]
	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.ModeDir|0500, fi.Mode())
	ExpectEq(t.clock.Now(), fi.ModTime())
	ExpectTrue(fi.IsDir())

	// hello
	fi = entries[1]
	ExpectEq("hello", fi.Name())
	ExpectEq(len("Hello, world!"), fi.Size())
	ExpectEq(0400, fi.Mode())
	ExpectEq(t.clock.Now(), fi.ModTime())
	ExpectFalse(fi.IsDir())
}

func (t *HelloFSTest) ReadDir_Subdir() {
	AssertTrue(false, "TODO")
}

func (t *HelloFSTest) StatHello() {
	AssertTrue(false, "TODO")
}

func (t *HelloFSTest) StatWorld() {
	AssertTrue(false, "TODO")
}

func (t *HelloFSTest) ReadHello() {
	AssertTrue(false, "TODO")
}

func (t *HelloFSTest) ReadWorld() {
	AssertTrue(false, "TODO")
}

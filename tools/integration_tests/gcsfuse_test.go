// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jacobsa/fuse"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestGcsfuse(t *testing.T) { RunTests(t) }

// Cf. bucket.go.
const fakeBucketName = "fake@bucket"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type GcsfuseTest struct {
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

var _ SetUpInterface = &GcsfuseTest{}
var _ TearDownInterface = &GcsfuseTest{}

func init() { RegisterTestSuite(&GcsfuseTest{}) }

func (t *GcsfuseTest) SetUp(_ *TestInfo) {
	var err error
	t.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")

	// Set up the temporary directory.
	t.dir, err = ioutil.TempDir("", "gcsfuse_test")
	AssertEq(nil, err)
}

func (t *GcsfuseTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

// Call gcsfuse with the supplied args, waiting for it to mount. Return nil
// only if it mounts successfully.
func (t *GcsfuseTest) mount(args []string) (err error) {
	// Set up a pipe that gcsfuse can write to to tell us when it has
	// successfully mounted.
	statusR, statusW, err := os.Pipe()
	if err != nil {
		err = fmt.Errorf("Pipe: %v", err)
		return
	}

	// Run gcsfuse, writing the result of waiting for it to a channel.
	gcsfuseErr := make(chan error, 1)
	go func() {
		gcsfuseErr <- t.runGcsfuse(args, statusW)
	}()

	// In the background, wait for something to be written to the pipe.
	pipeErr := make(chan error, 1)
	go func() {
		defer statusR.Close()
		n, err := statusR.Read(make([]byte, 1))
		if n == 1 {
			pipeErr <- nil
			return
		}

		pipeErr <- fmt.Errorf("statusR.Read: %v", err)
	}()

	// Watch for a result from one of them.
	select {
	case err = <-gcsfuseErr:
		err = fmt.Errorf("gcsfuse: %v", err)
		return

	case err = <-pipeErr:
		if err == nil {
			// All is good.
			return
		}

		err = <-gcsfuseErr
		err = fmt.Errorf("gcsfuse after pipe error: %v", err)
		return
	}
}

// Run gcsfuse and wait for it to return. Hand it the supplied pipe to write
// into when it successfully mounts. This function takes responsibility for
// closing the write end of the pipe locally.
func (t *GcsfuseTest) runGcsfuse(args []string, statusW *os.File) (err error) {
	defer statusW.Close()

	cmd := exec.Command(t.gcsfusePath)
	cmd.Args = append(cmd.Args, args...)
	cmd.ExtraFiles = []*os.File{statusW}
	cmd.Env = []string{"STATUS_PIPE=3"}

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v\nOutput:\n%s", err, output)
		return
	}

	return
}

// Unmount the file system mounted at the supplied directory. Try again on
// "resource busy" errors, which happen from time to time on OS X (due to weird
// requests from the Finder).
func unmount(dir string) (err error) {
	delay := 10 * time.Millisecond
	for {
		err = fuse.Unmount(dir)
		if err == nil {
			return
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		err = fmt.Errorf("Unmount: %v", err)
		return
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseTest) BadUsage() {
	testCases := []struct {
		args           []string
		expectedOutput string
	}{
		// Too few args
		0: {
			[]string{fakeBucketName},
			"exactly two arguments",
		},

		// Too many args
		1: {
			[]string{fakeBucketName, "a", "b"},
			"exactly two arguments",
		},

		// Unknown flag
		2: {
			[]string{"--tweak_frobnicator", fakeBucketName, "a"},
			"not defined.*tweak_frobnicator",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := exec.Command(t.gcsfusePath)
		cmd.Args = append(cmd.Args, tc.args...)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), MatchesRegexp(tc.expectedOutput), "case %d", i)
	}
}

func (t *GcsfuseTest) CannedContents() {
	var err error
	var fi os.FileInfo

	// Mount.
	args := []string{fakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Check the expected contents of the file system (cf. bucket.go).
	fi, err = os.Lstat(path.Join(t.dir, "foo"))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())

	contents, err := ioutil.ReadFile(path.Join(t.dir, "foo"))
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	fi, err = os.Lstat(path.Join(t.dir, "bar"))
	AssertEq(nil, err)
	ExpectEq(0755|os.ModeDir, fi.Mode())

	// The implicit directory shouldn't be visible, since we don't have implicit
	// directories enabled.
	_, err = os.Lstat(path.Join(t.dir, "baz"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *GcsfuseTest) ReadOnlyMode() {
	var err error

	// Mount.
	args := []string{"-o", "ro", fakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Writing to the file system should fail.
	err = ioutil.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *GcsfuseTest) ReadWriteMode() {
	var err error

	// Mount.
	args := []string{fakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Overwrite the canned file.
	p := path.Join(t.dir, "foo")

	err = ioutil.WriteFile(p, []byte("enchilada"), 0400)
	AssertEq(nil, err)

	contents, err := ioutil.ReadFile(p)
	AssertEq(nil, err)
	ExpectEq("enchilada", string(contents))
}

func (t *GcsfuseTest) FileAndDirModeFlags() {
	var err error
	var fi os.FileInfo

	// Mount with non-standard modes.
	args := []string{
		"--file-mode", "461",
		"--dir-mode", "511",
		fakeBucketName,
		t.dir,
	}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Stat contents.
	fi, err = os.Lstat(path.Join(t.dir, "foo"))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0461), fi.Mode())

	fi, err = os.Lstat(path.Join(t.dir, "bar"))
	AssertEq(nil, err)
	ExpectEq(0511|os.ModeDir, fi.Mode())
}

func (t *GcsfuseTest) UidAndGidFlags() {
	var err error
	var fi os.FileInfo

	// Mount, setting the flags. Make sure to set the directory mode such that we
	// can actually see the contents.
	args := []string{
		"--uid", "1719",
		"--gid", "2329",
		"--dir-mode", "555",
		fakeBucketName,
		t.dir,
	}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Stat contents.
	fi, err = os.Lstat(path.Join(t.dir, "foo"))
	AssertEq(nil, err)
	ExpectEq(1719, fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(2329, fi.Sys().(*syscall.Stat_t).Gid)

	fi, err = os.Lstat(path.Join(t.dir, "bar"))
	AssertEq(nil, err)
	ExpectEq(1719, fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(2329, fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *GcsfuseTest) ImplicitDirs() {
	var err error
	var fi os.FileInfo

	// Mount with implicit directories enabled.
	args := []string{
		"--implicit-dirs",
		fakeBucketName,
		t.dir,
	}

	err = t.mount(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// The implicit directory should be visible, as should its child.
	fi, err = os.Lstat(path.Join(t.dir, "baz"))
	AssertEq(nil, err)
	ExpectEq(0755|os.ModeDir, fi.Mode())

	fi, err = os.Lstat(path.Join(t.dir, "baz/qux"))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
}

func (t *GcsfuseTest) VersionFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) HelpFlags() {
	AssertTrue(false, "TODO")
}

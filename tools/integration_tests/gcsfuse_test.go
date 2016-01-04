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
	"bytes"
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

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestGcsfuse(t *testing.T) { RunTests(t) }

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

// Create an appropriate exec.Cmd for running gcsfuse, setting the required
// environment.
func (t *GcsfuseTest) gcsfuseCommand(args []string) (cmd *exec.Cmd) {
	cmd = exec.Command(t.gcsfusePath, args...)

	// Teach gcsfuse where fusermount lives.
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", path.Dir(gFusermountPath)),
	}

	return
}

// Call gcsfuse with the supplied args, waiting for it to exit. Return nil only
// if it exits successfully.
func (t *GcsfuseTest) runGcsfuse(args []string) (err error) {
	cmd := t.gcsfuseCommand(args)

	// Run.
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error %q running gcsfuse; output:\n%s", err.Error(), output)
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
			[]string{canned.FakeBucketName},
			"exactly two arguments",
		},

		// Too many args
		1: {
			[]string{canned.FakeBucketName, "a", "b"},
			"exactly two arguments",
		},

		// Unknown flag
		2: {
			[]string{"--tweak_frobnicator", canned.FakeBucketName, "a"},
			"not defined.*tweak_frobnicator",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := t.gcsfuseCommand(tc.args)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), MatchesRegexp(tc.expectedOutput), "case %d", i)
	}
}

func (t *GcsfuseTest) NonExistentMountPoint() {
	var err error

	// Mount.
	args := []string{canned.FakeBucketName, path.Join(t.dir, "blahblah")}

	err = t.runGcsfuse(args)
	ExpectThat(err, Error(HasSubstr("no such")))
	ExpectThat(err, Error(HasSubstr("blahblah")))
}

func (t *GcsfuseTest) MountPointIsAFile() {
	var err error

	// Write a file.
	p := path.Join(t.dir, "foo")

	err = ioutil.WriteFile(p, []byte{}, 0500)
	AssertEq(nil, err)
	defer os.Remove(p)

	// Mount.
	args := []string{canned.FakeBucketName, p}

	err = t.runGcsfuse(args)
	ExpectThat(err, Error(HasSubstr(p)))
	ExpectThat(err, Error(HasSubstr("not a directory")))
}

func (t *GcsfuseTest) CannedContents() {
	var err error
	var fi os.FileInfo

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Check the expected contents of the file system.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())

	contents, err := ioutil.ReadFile(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(canned.TopLevelFile_Contents, string(contents))

	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelDir))
	AssertEq(nil, err)
	ExpectEq(0755|os.ModeDir, fi.Mode())

	// The implicit directory shouldn't be visible, since we don't have implicit
	// directories enabled.
	_, err = os.Lstat(path.Join(t.dir, path.Dir(canned.ImplicitDirFile)))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *GcsfuseTest) ReadOnlyMode() {
	var err error

	// Mount.
	args := []string{"-o", "ro", canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Writing to the file system should fail.
	err = ioutil.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *GcsfuseTest) ReadWriteMode() {
	var err error

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Overwrite the canned file.
	p := path.Join(t.dir, canned.TopLevelFile)

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
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Stat contents.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0461), fi.Mode())

	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelDir))
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
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// Stat contents.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(1719, fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(2329, fi.Sys().(*syscall.Stat_t).Gid)

	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelDir))
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
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// The implicit directory should be visible, as should its child.
	fi, err = os.Lstat(path.Join(t.dir, path.Dir(canned.ImplicitDirFile)))
	AssertEq(nil, err)
	ExpectEq(0755|os.ModeDir, fi.Mode())

	fi, err = os.Lstat(path.Join(t.dir, canned.ImplicitDirFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
}

func (t *GcsfuseTest) OnlyDir() {
	var err error
	var fi os.FileInfo

	// Mount only a single directory from the bucket.
	args := []string{
		"--only-dir",
		path.Dir(canned.ExplicitDirFile),
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// It should be as if t.dir points into the bucket's first-level directory.
	entries, err := fusetesting.ReadDirPicky(t.dir)
	AssertEq(nil, err)

	AssertEq(1, len(entries))
	fi = entries[0]
	ExpectEq(path.Base(canned.ExplicitDirFile), fi.Name())
	ExpectEq(len(canned.ExplicitDirFile_Contents), fi.Size())
}

func (t *GcsfuseTest) OnlyDir_TrailingSlash() {
	var err error
	var fi os.FileInfo

	// Mount only a single directory from the bucket, including a trailing slash.
	args := []string{
		"--only-dir",
		path.Dir(canned.ExplicitDirFile) + "/",
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// It should be as if t.dir points into the bucket's first-level directory.
	entries, err := fusetesting.ReadDirPicky(t.dir)
	AssertEq(nil, err)

	AssertEq(1, len(entries))
	fi = entries[0]
	ExpectEq(path.Base(canned.ExplicitDirFile), fi.Name())
	ExpectEq(len(canned.ExplicitDirFile_Contents), fi.Size())
}

func (t *GcsfuseTest) OnlyDir_WithImplicitDir() {
	var err error
	var fi os.FileInfo

	// Mount only a single implicit directory from the bucket.
	args := []string{
		"--only-dir",
		path.Dir(canned.ImplicitDirFile),
		canned.FakeBucketName,
		t.dir,
	}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer unmount(t.dir)

	// It should be as if t.dir points into the implicit directory
	entries, err := fusetesting.ReadDirPicky(t.dir)
	AssertEq(nil, err)

	AssertEq(1, len(entries))
	fi = entries[0]
	ExpectEq(path.Base(canned.ImplicitDirFile), fi.Name())
	ExpectEq(len(canned.ImplicitDirFile_Contents), fi.Size())
}

func (t *GcsfuseTest) RelativeMountPoint() {
	// Start gcsfuse with a relative mount point.
	cmd := t.gcsfuseCommand([]string{
		canned.FakeBucketName,
		path.Base(t.dir),
	})

	cmd.Dir = path.Dir(t.dir)

	output, err := cmd.CombinedOutput()
	AssertEq(nil, err, "output:\n%s", output)

	defer unmount(t.dir)

	// The file system should be available.
	fi, err := os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *GcsfuseTest) ForegroundMode() {
	// Start gcsfuse, writing stderr to a pipe.
	cmd := t.gcsfuseCommand([]string{
		"--foreground",
		canned.FakeBucketName,
		t.dir,
	})

	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", path.Dir(gFusermountPath)),
	}

	stderr, err := cmd.StderrPipe()
	AssertEq(nil, err)

	err = cmd.Start()
	AssertEq(nil, err)
	defer cmd.Wait()
	defer cmd.Process.Kill()

	// Accumulate output from stderr until we see a successful mount message,
	// hackily synchronizing. Yes, this is an O(n^2) loop.
	var output []byte
	{
		buf := make([]byte, 4096)
		for !bytes.Contains(output, []byte("successfully mounted")) {
			n, err := stderr.Read(buf)
			output = append(output, buf[:n]...)
			AssertEq(nil, err, "Output so far:\n%s", output)
		}
	}

	defer unmount(t.dir)

	// The gcsfuse process should still be running, even after waiting a moment.
	time.Sleep(50 * time.Millisecond)
	err = cmd.Process.Signal(syscall.Signal(0))
	AssertEq(nil, err)

	// The file system should be available.
	fi, err := os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())

	// Unmounting should work fine.
	err = unmount(t.dir)
	AssertEq(nil, err)

	// Now the process should exit successfully.
	err = cmd.Wait()
	AssertEq(nil, err)
}

func (t *GcsfuseTest) VersionFlags() {
	testCases := []struct {
		args []string
	}{
		0: {[]string{"-v"}},
		1: {[]string{"--version"}},
	}

	// For each argument, gcsfuse should exist successfully.
	for i, tc := range testCases {
		cmd := t.gcsfuseCommand(tc.args)
		output, err := cmd.CombinedOutput()
		ExpectEq(nil, err, "case %d\nOutput:\n%s", i, output)
	}
}

func (t *GcsfuseTest) HelpFlags() {
	testCases := []struct {
		args []string
	}{
		0: {[]string{"-h"}},
		1: {[]string{"--help"}},
	}

	// For each argument, gcsfuse should exist successfully.
	for i, tc := range testCases {
		cmd := t.gcsfuseCommand(tc.args)
		output, err := cmd.CombinedOutput()
		ExpectEq(nil, err, "case %d\nOutput:\n%s", i, output)
	}
}

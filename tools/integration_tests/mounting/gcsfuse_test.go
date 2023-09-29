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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/tools/util"
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
	t.dir, err = os.MkdirTemp("", "gcsfuse_test")
	AssertEq(nil, err)
}

func (t *GcsfuseTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

// Create an appropriate exec.Cmd for running gcsfuse, setting the required
// environment.
func (t *GcsfuseTest) gcsfuseCommand(args []string, env []string) (cmd *exec.Cmd) {
	cmd = exec.Command(t.gcsfusePath, args...)
	cmd.Env = make([]string, len(env))
	copy(cmd.Env, env)

	// Teach gcsfuse where fusermount lives.
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path.Dir(gFusermountPath)))

	return
}

// Call gcsfuse with the supplied args and environment variable,
// waiting for it to exit. Return nil only if it exits successfully.
func (t *GcsfuseTest) runGcsfuseWithEnv(args []string, env []string) (err error) {
	cmd := t.gcsfuseCommand(args, env)

	// Run.
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error %w running gcsfuse; output:\n%s", err, output)
		return
	}

	return
}

// Call gcsfuse with the supplied args, waiting for it to exit. Return nil only
// if it exits successfully.
func (t *GcsfuseTest) runGcsfuse(args []string) (err error) {
	return t.runGcsfuseWithEnv(args, nil)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseTest) BadUsage() {
	testCases := []struct {
		args           []string
		expectedOutput string
	}{
		// Too many args
		0: {
			[]string{canned.FakeBucketName, "a", "b"},
			"gcsfuse takes one or two arguments.",
		},

		// Unknown flag
		1: {
			[]string{"--tweak_frobnicator", canned.FakeBucketName, "a"},
			"not defined.*tweak_frobnicator",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := t.gcsfuseCommand(tc.args, nil)

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

// TODO: fails
/*
func (t *GcsfuseTest) NonEmptyMountPoint() {
	var err error

	// osxfuse apparently doesn't care about this.
	if runtime.GOOS == "darwin" {
		return
	}

	// Write a file into the mount point.
	p := path.Join(t.dir, "foo")
	err = os.WriteFile(p, nil, 0600)
	AssertEq(nil, err)

	defer os.Remove(p)

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	ExpectThat(err, Error(HasSubstr("exit status 1")))
}
*/

func (t *GcsfuseTest) MountPointIsAFile() {
	var err error

	// Write a file.
	p := path.Join(t.dir, "foo")

	err = os.WriteFile(p, []byte{}, 0500)
	AssertEq(nil, err)
	defer os.Remove(p)

	// Mount.
	args := []string{canned.FakeBucketName, p}

	err = t.runGcsfuse(args)
	ExpectThat(err, Error(HasSubstr(p)))
	ExpectThat(err, Error(HasSubstr("is not a directory")))
}

func (t *GcsfuseTest) KeyFile() {
	const nonexistent = "/tmp/foobarbazdoesntexist"

	// Specify a non-existent key file in two different ways.
	// We pass --max-retry-sleep 0, just to limit the number of retry to 0 with no wait time
	testCases := []struct {
		extraArgs []string
		env       []string
	}{
		// Via flag
		0: {
			extraArgs: []string{fmt.Sprintf("--key-file=%s", nonexistent), "--max-retry-sleep=0"},
		},

		// Via the environment
		1: {
			env:       []string{fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", nonexistent)},
			extraArgs: []string{"--max-retry-sleep=0"},
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		args := tc.extraArgs
		args = append(args, "some-non-canned-bucket-name", t.dir)

		cmd := t.gcsfuseCommand(args, tc.env)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), HasSubstr(nonexistent), "case %d", i)
		ExpectThat(string(output), HasSubstr("no such file"), "case %d", i)

		AssertEq(nil, util.Unmount(t.dir))
	}
}

func (t *GcsfuseTest) CannedContents() {
	var err error
	var fi os.FileInfo

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Check the expected contents of the file system.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())

	contents, err := os.ReadFile(path.Join(t.dir, canned.TopLevelFile))
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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Writing to the file system should fail.
	err = os.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *GcsfuseTest) ReadWriteMode() {
	var err error

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.runGcsfuse(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Overwrite the canned file.
	p := path.Join(t.dir, canned.TopLevelFile)

	err = os.WriteFile(p, []byte("enchilada"), 0400)
	AssertEq(nil, err)

	contents, err := os.ReadFile(p)
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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	},
		nil)

	cmd.Dir = path.Dir(t.dir)

	output, err := cmd.CombinedOutput()
	AssertEq(nil, err, "output:\n%s", output)

	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// The file system should be available.
	fi, err := os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *GcsfuseTest) ForegroundMode() {
	// Start gcsfuse, writing stderr to a pipe.
	// Here --log-file=/proc/self/fd/2 represents stderr
	cmd := t.gcsfuseCommand([]string{
		"--foreground",
		"--log-file=/proc/self/fd/2",
		canned.FakeBucketName,
		t.dir,
	},
		nil)

	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", path.Dir(gFusermountPath)),
	}

	stderr, err := cmd.StderrPipe()
	AssertEq(nil, err)

	err = cmd.Start()
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, cmd.Wait())
	}()
	defer func() {
		ExpectEq(nil, cmd.Process.Kill())
	}()

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

	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

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
	AssertEq(nil, util.Unmount(t.dir))

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
		cmd := t.gcsfuseCommand(tc.args, nil)
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
		cmd := t.gcsfuseCommand(tc.args, nil)
		output, err := cmd.CombinedOutput()
		ExpectEq(nil, err, "case %d\nOutput:\n%s", i, output)
	}
}

const TEST_RELATIVE_FILE_NAME = "test.txt"
const TEST_HOME_RELATIVE_FILE_NAME = "test_home.json"

func createTestFilesForRelativePathTesting() (
	curDirTestFile string, homeDirTestFile string) {

	curWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	curDirTestFile = filepath.Join(curWorkingDir, TEST_RELATIVE_FILE_NAME)
	_, err = os.Create(curDirTestFile)
	AssertEq(nil, err)

	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)

	homeDirTestFile = filepath.Join(homeDir, TEST_HOME_RELATIVE_FILE_NAME)
	_, err = os.Create(homeDirTestFile)
	AssertEq(nil, err)

	return
}

func (t *GcsfuseTest) LogFilePath() {
	curDirTestFile, homeDirTestFile := createTestFilesForRelativePathTesting()
	defer os.Remove(curDirTestFile)
	defer os.Remove(homeDirTestFile)

	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)

	// Specify log-file and key-file in different way with --foreground flag.
	testCases := []struct {
		extraArgs []string
		env       []string
	}{
		// Relative path
		0: {
			extraArgs: []string{"--log-file", TEST_RELATIVE_FILE_NAME},
		},

		// Relative with ./
		1: {
			extraArgs: []string{"--log-file",
				fmt.Sprintf("./%s", TEST_RELATIVE_FILE_NAME)},
		},

		// Path with tilda
		2: {
			extraArgs: []string{"--log-file",
				fmt.Sprintf("~/%s", TEST_HOME_RELATIVE_FILE_NAME)},
			env: []string{fmt.Sprintf("HOME=%s", homeDir)},
		},

		// Absolute path
		3: {
			extraArgs: []string{"--log-file", curDirTestFile},
		},
	}

	for _, tc := range testCases {
		args := tc.extraArgs
		args = append(args, canned.FakeBucketName, t.dir)

		AssertEq(nil, t.runGcsfuseWithEnv(args, tc.env))
		AssertEq(nil, util.Unmount(t.dir))
	}
}

func (t *GcsfuseTest) KeyFilePath() {
	curDirTestFile, homeDirTestFile := createTestFilesForRelativePathTesting()
	defer os.Remove(curDirTestFile)
	defer os.Remove(homeDirTestFile)

	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)

	// Specify key-file in different way with --foreground flag.
	testCases := []struct {
		extraArgs []string
		env       []string
	}{
		// relative path
		0: {
			extraArgs: []string{"--key-file", TEST_RELATIVE_FILE_NAME},
		},

		// relative with ./
		1: {
			extraArgs: []string{"--key-file",
				fmt.Sprintf("./%s", TEST_RELATIVE_FILE_NAME)},
		},

		// path with tilda
		2: {
			extraArgs: []string{"--key-file",
				fmt.Sprintf("~/%s", TEST_HOME_RELATIVE_FILE_NAME)},
			env: []string{fmt.Sprintf("HOME=%s", homeDir)},
		},

		// Absolute path
		3: {
			extraArgs: []string{"--key-file", curDirTestFile},
		},
	}

	for _, tc := range testCases {
		args := tc.extraArgs
		args = append(args, canned.FakeBucketName, t.dir)

		AssertEq(nil, t.runGcsfuseWithEnv(args, tc.env))
		AssertEq(nil, util.Unmount(t.dir))
	}
}

func (t *GcsfuseTest) BothLogAndKeyFilePath() {
	curDirTestFile, homeDirTestFile := createTestFilesForRelativePathTesting()
	defer os.Remove(curDirTestFile)
	defer os.Remove(homeDirTestFile)

	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)

	// Specify log-file and key-file in different way with --foreground flag.
	testCases := []struct {
		extraArgs []string
		env       []string
	}{
		// relative path
		0: {
			extraArgs: []string{"--key-file", TEST_RELATIVE_FILE_NAME,
				"--log-file", TEST_RELATIVE_FILE_NAME},
		},

		// relative with ./
		1: {
			extraArgs: []string{"--key-file",
				fmt.Sprintf("./%s", TEST_RELATIVE_FILE_NAME),
				"--log-file",
				fmt.Sprintf("./%s", TEST_RELATIVE_FILE_NAME)},
		},

		// path with tilda
		2: {
			extraArgs: []string{"--key-file",
				fmt.Sprintf("~/%s", TEST_HOME_RELATIVE_FILE_NAME),
				"--log-file",
				fmt.Sprintf("~/%s", TEST_HOME_RELATIVE_FILE_NAME)},
			env: []string{fmt.Sprintf("HOME=%s", homeDir)},
		},

		// Absolute path
		3: {
			extraArgs: []string{"--log-file", curDirTestFile,
				"--key-file", curDirTestFile},
		},
	}

	for _, tc := range testCases {
		args := tc.extraArgs
		args = append(args, canned.FakeBucketName, t.dir)

		AssertEq(nil, t.runGcsfuseWithEnv(args, tc.env))
		AssertEq(nil, util.Unmount(t.dir))
	}
}

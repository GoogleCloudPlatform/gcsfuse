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
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/canned"
	"github.com/googlecloudplatform/gcsfuse/tools/util"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestMountHelper(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MountHelperTest struct {
	// Path to the mount(8) helper binary.
	helperPath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

var _ SetUpInterface = &MountHelperTest{}
var _ TearDownInterface = &MountHelperTest{}

func init() { RegisterTestSuite(&MountHelperTest{}) }

func (t *MountHelperTest) SetUp(_ *TestInfo) {
	var err error

	// Set up the appropriate helper path.
	switch runtime.GOOS {
	case "darwin":
		t.helperPath = path.Join(gBuildDir, "sbin/mount_gcsfuse")

	case "linux":
		t.helperPath = path.Join(gBuildDir, "sbin/mount.gcsfuse")

	default:
		AddFailure("Don't know how to deal with OS: %q", runtime.GOOS)
		AbortTest()
	}

	// Set up the temporary directory.
	t.dir, err = os.MkdirTemp("", "mount_helper_test")
	AssertEq(nil, err)
}

func (t *MountHelperTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

func (t *MountHelperTest) mountHelperCommand(args []string) (cmd *exec.Cmd) {
	cmd = exec.Command(t.helperPath)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", path.Join(gBuildDir, "bin")),
	}

	return
}

func (t *MountHelperTest) mount(args []string) (err error) {
	cmd := t.mountHelperCommand(args)

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("CombinedOutput: %w\nOutput:\n%s", err, output)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MountHelperTest) BadUsage() {
	testCases := []struct {
		args           []string
		expectedOutput string
	}{
		// Too few args
		0: {
			[]string{canned.FakeBucketName},
			"two positional arguments",
		},

		// Too many args
		1: {
			[]string{canned.FakeBucketName, "a", "b"},
			"Unexpected arg 3",
		},

		// Trailing -o
		2: {
			[]string{canned.FakeBucketName, "a", "-o"},
			"Unexpected -o",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := t.mountHelperCommand(tc.args)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), MatchesRegexp(tc.expectedOutput), "case %d", i)
	}
}

func (t *MountHelperTest) NoMtabFlag() {
	var err error

	// Mount. The "-n" argument should be ignored.
	args := []string{canned.FakeBucketName, t.dir, "-n"}

	err = t.mount(args)
	AssertEq(nil, err)
	AssertEq(nil, util.Unmount(t.dir))
}

func (t *MountHelperTest) SuccessfulMount() {
	var err error
	var fi os.FileInfo

	// Mount.
	args := []string{canned.FakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Check that the file system is available.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *MountHelperTest) RelativeMountPoint() {
	var err error
	var fi os.FileInfo

	// Mount with a relative mount point.
	cmd := t.mountHelperCommand([]string{
		canned.FakeBucketName,
		path.Base(t.dir),
	})

	cmd.Dir = path.Dir(t.dir)

	output, err := cmd.CombinedOutput()
	AssertEq(nil, err, "output:\n%s", output)

	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// The file system should be available.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *MountHelperTest) ReadOnlyMode() {
	var err error

	// Mount.
	args := []string{"-o", "ro", canned.FakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Writing to the file system should fail.
	err = os.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *MountHelperTest) ExtraneousOptions() {
	var err error
	var fi os.FileInfo

	// Mount with extra junk that shouldn't be passed on.
	args := []string{
		"-o", "noauto,nouser,no_netdev,auto,user,_netdev",
		canned.FakeBucketName,
		t.dir,
	}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Check that the file system is available.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *MountHelperTest) LinuxArgumentOrder() {
	var err error

	// Linux places the options at the end.
	args := []string{canned.FakeBucketName, t.dir, "-o", "ro"}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Writing to the file system should fail.
	err = os.WriteFile(path.Join(t.dir, "blah"), []byte{}, 0400)
	ExpectThat(err, Error(HasSubstr("read-only")))
}

func (t *MountHelperTest) FuseSubtype() {
	var err error
	var fi os.FileInfo

	// This test isn't relevant except on Linux.
	if runtime.GOOS != "linux" {
		return
	}

	// Mount using the tool that would be invoked by ~mount -t fuse.gcsfuse`.
	t.helperPath = path.Join(gBuildDir, "sbin/mount.fuse.gcsfuse")
	args := []string{canned.FakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Check that the file system is available.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0644), fi.Mode())
	ExpectEq(len(canned.TopLevelFile_Contents), fi.Size())
}

func (t *MountHelperTest) ModeOptions() {
	var err error
	var fi os.FileInfo

	// Mount.
	args := []string{
		"-o", "dir_mode=754",
		"-o", "file_mode=612",
		canned.FakeBucketName, t.dir,
	}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// Stat the directory.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelDir))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0754)|os.ModeDir, fi.Mode())

	// Stat the file.
	fi, err = os.Lstat(path.Join(t.dir, canned.TopLevelFile))
	AssertEq(nil, err)
	ExpectEq(os.FileMode(0612), fi.Mode())
}

func (t *MountHelperTest) ImplicitDirs() {
	var err error

	// Mount.
	args := []string{"-o", "implicit_dirs", canned.FakeBucketName, t.dir}

	err = t.mount(args)
	AssertEq(nil, err)
	defer func() {
		AssertEq(nil, util.Unmount(t.dir))
	}()

	// The implicit directory should be visible.
	fi, err := os.Lstat(path.Join(t.dir, path.Dir(canned.ImplicitDirFile)))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())
}

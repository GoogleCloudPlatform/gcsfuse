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
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

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
	t.dir, err = ioutil.TempDir("", "mount_helper_test")
	AssertEq(nil, err)
}

func (t *MountHelperTest) TearDown() {
	err := os.Remove(t.dir)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MountHelperTest) BadUsage() {
	AssertTrue(false, "TODO")
}

func (t *MountHelperTest) SuccessfulMount() {
	AssertTrue(false, "TODO")
}

func (t *MountHelperTest) ReadOnlyMode() {
	AssertTrue(false, "TODO")
}

func (t *MountHelperTest) ExtraneousOptions() {
	AssertTrue(false, "TODO")
}

func (t *MountHelperTest) FuseSubtype() {
	AssertTrue(false, "TODO")
}

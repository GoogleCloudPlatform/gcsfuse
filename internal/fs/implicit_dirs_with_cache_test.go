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

// A collection of tests for a file system where we do not attempt to write to
// the file system at all. Rather we set up contents in a GCS bucket out of
// band, wait for them to be available, and then read them via the file system.

package fs_test

import (
	"os"
	"os/exec"
	"path"
	"time"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ImplicitDirsWithCacheTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&ImplicitDirsWithCacheTest{})
}

func (t *ImplicitDirsWithCacheTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.DirTypeCacheTTL = time.Minute * 3
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ImplicitDirsWithCacheTest) TestRemoveAll() {
	// Create an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo/bar": "",
			}))

	// Attempt to recursively remove implicit dir.
	cmd := exec.Command(
		"rm",
		"-r",
		path.Join(mntDir, "foo/"),
	)
	output, err := cmd.CombinedOutput()

	AssertEq("", string(output))
	AssertEq(nil, err)
}

func (t *ImplicitDirsWithCacheTest) TestRenameImplicitDir() {
	// Create an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo/bar1": "",
				"foo/bar2": "",
			}))

	// Attempt to rename implicit dir.
	err := os.Rename(
		path.Join(mntDir, "foo/"),
		path.Join(mntDir, "fooNew/"),
	)

	//verify os.Rename successful
	AssertEq(nil, err)
	// verify the renamed files and directories
	fi1, err := os.Stat(path.Join(mntDir, "fooNew"))
	AssertEq(nil, err)
	AssertTrue(fi1.IsDir())
	fi2, err := os.Stat(path.Join(mntDir, "fooNew/bar1"))
	AssertEq(nil, err)
	AssertEq(fi2.Name(), "bar1")
	fi3, err := os.Stat(path.Join(mntDir, "fooNew/bar2"))
	AssertEq(nil, err)
	AssertEq(fi3.Name(), "bar2")
	fi4, err := os.Stat(path.Join(mntDir, "foo"))
	ExpectThat(err, Error(HasSubstr("no such file or directory")))
	AssertEq(nil, fi4)
}

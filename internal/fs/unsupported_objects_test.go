// Copyright 2024 Google LLC
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
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UnsupportedObjectsTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&UnsupportedObjectsTest{})
}

func (t *UnsupportedObjectsTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func verifyInvalidPath(t *UnsupportedObjectsTest, path string) {
	_, err := os.Stat(path)

	AssertNe(nil, err, "Failed to get error in stat of %q", path)
}

// Create objects with unsupported object names and
// verify the behavior of mount using os.Stat and WalkDir.
func (t *UnsupportedObjectsTest) UnsupportedGcsObjectNames() {
	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo//0":   "", // unsupported
				"foo/1":    "", // supported
				"foo/2//4": "", // unsupported
				"foo//2/5": "", // unsupported
				"foo/2/6":  "", // supported
				"6":        "", // supported
				"/7":       "", // unsupported
				"/8/10":    "", // unsupported
				"/":        "", // unsupported
				"10/12":    "", // supported
			}))

	// Verify that the unsupported objects fail os.Stat.
	verifyInvalidPath(t, path.Join(mntDir, "foo//0"))
	verifyInvalidPath(t, path.Join(mntDir, "foo/2//4"))
	verifyInvalidPath(t, path.Join(mntDir, "foo//2/5"))
	verifyInvalidPath(t, path.Join(mntDir, "/7"))
	verifyInvalidPath(t, path.Join(mntDir, "/8/10"))

	// Verify that the supported objects appear in WalkDir.
	expectedWalkedEntries := []struct {
		path  string
		name  string
		isDir bool
		found bool
	}{{
		path:  mntDir,
		name:  mntDir[strings.LastIndex(mntDir, "/")+1:],
		isDir: true,
	}, {
		path:  path.Join(mntDir, "foo"),
		name:  "foo",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/1"),
		name: "1",
	}, {
		path:  path.Join(mntDir, "foo/2"),
		name:  "2",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/2/6"),
		name: "6",
	}, {
		path: path.Join(mntDir, "6"),
		name: "6",
	}, {
		path:  path.Join(mntDir, "10"),
		name:  "10",
		isDir: true,
	}, {
		path: path.Join(mntDir, "10/12"),
		name: "12",
	},
	}

	AssertEq(nil, filepath.WalkDir(mntDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		foundMatchingExpectedWalkingEntry := false
		for i := range expectedWalkedEntries {
			expectedWalkedEntry := &expectedWalkedEntries[i]
			if expectedWalkedEntry.path == path && expectedWalkedEntry.name == d.Name() && d.IsDir() == expectedWalkedEntry.isDir {
				if expectedWalkedEntry.found {
					return fmt.Errorf("found duplicate walked entry: path=%s, name=%s, isDir=%v", path, d.Name(), d.IsDir())
				}

				foundMatchingExpectedWalkingEntry = true
				expectedWalkedEntry.found = true
			}
		}

		if !foundMatchingExpectedWalkingEntry {
			return fmt.Errorf("got unexpected walk entry: path=%s, name=%s, isDir=%v", path, d.Name(), d.IsDir())
		}

		return nil
	}))

	for _, expectedWalkedEntry := range expectedWalkedEntries {
		if !expectedWalkedEntry.found {
			AddFailure("Missing walked entry: path=%s, name=%s, isDir=%v", expectedWalkedEntry.path, expectedWalkedEntry.name, expectedWalkedEntry.isDir)
		}
	}

	err = os.RemoveAll(mntDir + "/*")

	AssertEq(nil, err)
}

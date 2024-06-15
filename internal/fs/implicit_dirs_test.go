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
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ImplicitDirsTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&ImplicitDirsTest{})
}

func (t *ImplicitDirsTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true

	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// Create objects in implicit directories with
// unsupported object names and
// test that stat and ReadDirPicky on the different directories.
func (t *ImplicitDirsTest) UnsupportedDirNames() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo//bar": "", // unsupported
				"foo/1":    "", // supported
				"a/2":      "", // supported
				"a//2/6":   "", //unsupported
				"a//3":     "", // unsupported
				"4":        "", // supported
				"/4/7":     "", //unsupported
				"/bar":     "", // unsupported
				"bar//5":   "", // unsupported
				"/":        "", //unsupported
			}))
	defer func() {
		storageutil.DeleteAllObjects(context.Background(), bucket)
	}()

	// Statting the mount directory should return a directory entry.
	fi, err = os.Stat(mntDir)
	AssertEq(nil, err)
	AssertNe(nil, fi)
	ExpectTrue(fi.IsDir())

	// Statting the mount-directory/foo should return a directory entry named "foo".
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	AssertNe(nil, fi)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting the mount-directory/a should return a directory entry named "a".
	fi, err = os.Stat(path.Join(mntDir, "a"))
	AssertEq(nil, err)
	AssertNe(nil, fi)
	ExpectEq("a", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting the mount-directory/a//3 should fail as it should be ignored.
	_, err = os.Stat(path.Join(mntDir, "a/3"))
	AssertNe(nil, err)

	// Statting the mount-directory/4 should return a file entry named "4".
	fi, err = os.Stat(path.Join(mntDir, "4"))
	AssertEq(nil, err)
	AssertNe(nil, fi)
	ExpectEq("4", fi.Name())
	ExpectFalse(fi.IsDir())

	// Statting the mount-directory/bar should fail as it should be ignored.
	fi, err = os.Stat(path.Join(mntDir, "bar"))
	//AssertNe(nil, err)
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	// ReadDirPicky on mountdir should not fail as the unsupported sub-directories should be ignored.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	//AssertNe(nil, err)
	///*
	AssertEq(nil, err)
	AssertNe(nil, entries)
	AssertEq(4, len(entries))
	AssertNe(nil, entries[0])
	ExpectEq("4", entries[0].Name())
	ExpectFalse(entries[0].IsDir())
	AssertNe(nil, entries[1])
	ExpectEq("a", entries[1].Name())
	ExpectTrue(entries[1].IsDir())
	AssertNe(nil, entries[2])
	ExpectEq("bar", entries[2].Name())
	ExpectTrue(entries[2].IsDir())
	AssertNe(nil, entries[3])
	ExpectEq("foo", entries[3].Name())
	ExpectTrue(entries[3].IsDir())
	//*/

	// ReadDirPicky on mountdir/foo should work as the unsupported sub-directories should be ignored.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	AssertNe(nil, entries)
	AssertEq(1, len(entries))
	AssertNe(nil, entries[0])
	ExpectEq("1", entries[0].Name())
	ExpectFalse(entries[0].IsDir())

	// ReadDirPicky on mntdir/a should work.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "a"))
	AssertEq(nil, err)
	AssertEq(1, len(entries))
	AssertNe(nil, entries[0])
	ExpectEq("2", entries[0].Name())
	ExpectFalse(entries[0].IsDir())
}

// Create objects in implicit directories with
// unsupported names such as those having // in them
// and test that stat and ReadDirPicky on the different directories.
func (t *ImplicitDirsTest) UnsupportedDirNames_WalkDir() {
	// Set up contents.
	ExpectEq(
		nil,
		t.createObjects(
			map[string]string{
				"a/b":     "", // supported
				"a//b/i":  "", // unsupported
				"foo/c/d": "", // supported
				"foo//e":  "", // unsupported
				"f":       "", // supported
				"/h":      "", // unsupported
				"/":       "", // unsupported
			}))

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
		path: path.Join(mntDir, "f"),
		name: "f",
	}, {
		path:  path.Join(mntDir, "a"),
		name:  "a",
		isDir: true,
	}, {
		path: path.Join(mntDir, "a/b"),
		name: "b",
	}, {
		path:  path.Join(mntDir, "foo"),
		name:  "foo",
		isDir: true,
	}, {
		path:  path.Join(mntDir, "foo/c"),
		name:  "c",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/c/d"),
		name: "d",
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
}

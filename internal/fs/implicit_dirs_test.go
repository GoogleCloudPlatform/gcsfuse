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
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
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

func (t *ImplicitDirsTest) NothingPresent() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *ImplicitDirsTest) FileObjectPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo": "taco",
			}))

	// Statting the name should return an entry for the file.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())

	// ReadDir should show the file.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) DirectoryObjectPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// Directory
				"foo/": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) ImplicitDirectory_DefinedByFile() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/bar": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) ImplicitDirectory_DefinedByDirectory() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/bar/": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) ConflictingNames_PlaceholderPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo": "taco",

				// Directory
				"foo/": "",
			}))

	// A listing of the parent should contain a directory named "foo" and a
	// file named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) ConflictingNames_PlaceholderNotPresent() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo": "taco",

				// Implicit directory
				"foo/bar": "",
			}))

	// A listing of the parent should contain a directory named "foo" and a
	// file named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())

	fi = entries[1]
	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) ConflictingNames_OneIsSymlink() {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// Symlink
				"foo": "",

				// Directory
				"foo/": "",
			}))

	// Cause "foo" to look like a symlink.
	err = setSymlinkTarget(ctx, bucket, "foo", "")
	AssertEq(nil, err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	ExpectEq("foo\n", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Lstat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(mntDir, "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

func (t *ImplicitDirsTest) StatUnknownName_NoOtherContents() {
	var err error

	// Stat an unknown name.
	_, err = os.Stat(path.Join(mntDir, "unknown"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ImplicitDirsTest) StatUnknownName_UnrelatedContents() {
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"bar": "",
				"baz": "",
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(mntDir, "foo"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ImplicitDirsTest) StatUnknownName_PrefixOfActualNames() {
	var err error

	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foop":  "",
				"fooq/": "",
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(mntDir, "foo"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ImplicitDirsTest) ImplicitBecomesExplicit() {
	var fi os.FileInfo
	var err error

	// Set up an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/bar": "",
			}))

	// Stat it.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Set up an explicit placeholder.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/": "",
			}))

	// Stat the directory again.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) ExplicitBecomesImplicit() {
	var fi os.FileInfo
	var err error

	// Set up an explicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/":    "",
				"foo/bar": "",
			}))

	// Stat it.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Remove the explicit placeholder.
	AssertEq(
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "foo/"}))

	// Stat the directory again.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) Rmdir_NotEmpty_OnlyImplicit() {
	var err error

	// Set up an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/bar": "",
			}))

	// Attempt to remove it.
	err = os.Remove(path.Join(mntDir, "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))

	AssertEq(nil, err)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) Rmdir_NotEmpty_ImplicitAndExplicit() {
	var err error

	// Set up an implicit directory that also has a backing object.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/":    "",
				"foo/bar": "",
			}))

	// Attempt to remove it.
	err = os.Remove(path.Join(mntDir, "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))

	AssertEq(nil, err)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) Rmdir_Empty() {
	var err error
	var entries []os.FileInfo

	// Create two levels of directories. We can't make an empty implicit dir, so
	// there must be a backing object for each.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				"foo/":     "",
				"foo/bar/": "",
			}))

	// Remove the leaf.
	err = os.Remove(path.Join(mntDir, "foo/bar"))
	AssertEq(nil, err)

	// There should be nothing left in the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	// Now the root directory should be empty, too.
	entries, err = fusetesting.ReadDirPicky(mntDir)

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *ImplicitDirsTest) AtimeCtimeAndMtime() {
	var err error
	mountTime := mtimeClock.Now()

	// Create an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo/bar": "",
			}))

	// Stat it.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	// We require only that the times be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(mtime, timeutil.TimeNear(mountTime, delta))
}

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
	_, err = os.Stat(path.Join(mntDir, "bar"))
	AssertNe(nil, err)

	// ReadDirPicky on mountdir should not fail as the unsupported sub-directories should be ignored.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertNe(nil, entries)
	AssertEq(3, len(entries))
	AssertNe(nil, entries[0])
	ExpectEq("4", entries[0].Name())
	ExpectFalse(entries[0].IsDir())
	AssertNe(nil, entries[1])
	ExpectEq("a", entries[1].Name())
	ExpectTrue(entries[1].IsDir())
	AssertNe(nil, entries[2])
	ExpectEq("foo", entries[2].Name())
	ExpectTrue(entries[2].IsDir())

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

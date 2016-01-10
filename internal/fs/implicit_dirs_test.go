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
	"path"
	"syscall"
	"time"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
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

func init() { RegisterTestSuite(&ImplicitDirsTest{}) }

func (t *ImplicitDirsTest) SetUp(ti *TestInfo) {
	t.serverCfg.ImplicitDirectories = true
	t.fsTest.SetUp(ti)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ImplicitDirsTest) NothingPresent() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())

	// ReadDir should show the file.
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo\n"))
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
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo\n"))
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
	err = setSymlinkTarget(t.ctx, t.bucket, "foo", "")
	AssertEq(nil, err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(t.mfs.Dir())
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
	fi, err = os.Lstat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(t.mfs.Dir(), "foo\n"))
	AssertEq(nil, err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

func (t *ImplicitDirsTest) StatUnknownName_NoOtherContents() {
	var err error

	// Stat an unknown name.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Remove the explicit placeholder.
	AssertEq(
		nil,
		t.bucket.DeleteObject(
			t.ctx,
			&gcs.DeleteObjectRequest{Name: "foo/"}))

	// Stat the directory again.
	fi, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))
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
	err = os.Remove(path.Join(t.Dir, "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(t.Dir, "foo"))

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
	err = os.Remove(path.Join(t.Dir, "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(t.Dir, "foo"))

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
	err = os.Remove(path.Join(t.Dir, "foo/bar"))
	AssertEq(nil, err)

	// There should be nothing left in the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(t.Dir, "foo"))

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(t.Dir, "foo"))
	AssertEq(nil, err)

	// Now the root directory should be empty, too.
	entries, err = fusetesting.ReadDirPicky(t.Dir)

	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *ImplicitDirsTest) AtimeCtimeAndMtime() {
	var err error
	mountTime := t.mtimeClock.Now()

	// Create an implicit directory.
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"foo/bar": "",
			}))

	// Stat it.
	fi, err := os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// We require only that the times be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(mtime, timeutil.TimeNear(mountTime, delta))
}

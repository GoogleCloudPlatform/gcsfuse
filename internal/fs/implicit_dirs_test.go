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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImplicitDirsTestSuite(t *testing.T) {
	suite.Run(t, new(ImplicitDirsTest))
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ImplicitDirsTest struct {
	suite.Suite
	fsTest
}

func (t *ImplicitDirsTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.fsTest.SetupSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ImplicitDirsTest) TestNothingPresent() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	ExpectThat(entries, ElementsAre())
}

func (t *ImplicitDirsTest) TestFileObjectPresent() {
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())

	// ReadDir should show the file.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(4, fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) TestDirectoryObjectPresent() {
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestImplicitDirectory_DefinedByFile() {
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestImplicitDirectory_DefinedByDirectory() {
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestConflictingNames_PlaceholderPresent() {
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
	assert.Nil(t.T(), err)
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	assert.Nil(t.T(), err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) TestConflictingNames_PlaceholderNotPresent() {
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
	assert.Nil(t.T(), err)
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(mntDir, "foo\n"))
	assert.Nil(t.T(), err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
}

func (t *ImplicitDirsTest) TestConflictingNames_OneIsSymlink() {
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
	assert.Nil(t.T(), err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(mntDir, "foo\n"))
	assert.Nil(t.T(), err)

	ExpectEq("foo\n", fi.Name())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

func (t *ImplicitDirsTest) TestStatUnknownName_NoOtherContents() {
	var err error

	// Stat an unknown name.
	_, err = os.Stat(path.Join(mntDir, "unknown"))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *ImplicitDirsTest) TestStatUnknownName_UnrelatedContents() {
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

func (t *ImplicitDirsTest) TestStatUnknownName_PrefixOfActualNames() {
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

func (t *ImplicitDirsTest) TestImplicitBecomesExplicit() {
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
	assert.Nil(t.T(), err)

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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestExplicitBecomesImplicit() {
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
	assert.Nil(t.T(), err)

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
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestRmdir_NotEmpty_OnlyImplicit() {
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

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestRmdir_NotEmpty_ImplicitAndExplicit() {
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

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *ImplicitDirsTest) TestRmdir_Empty() {
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
	assert.Nil(t.T(), err)

	// There should be nothing left in the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Now the root directory should be empty, too.
	entries, err = fusetesting.ReadDirPicky(mntDir)

	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())
}

func (t *ImplicitDirsTest) TestAtimeCtimeAndMtime() {
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
	assert.Nil(t.T(), err)

	// We require only that the times be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(mtime, timeutil.TimeNear(mountTime, delta))
}

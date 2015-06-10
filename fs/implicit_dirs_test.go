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

	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ImplicitDirsTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&ImplicitDirsTest{}) }

func (s *ImplicitDirsTest) SetUp(t *ogletest.T) {
	s.serverCfg.ImplicitDirectories = true
	s.fsTest.SetUp(t)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (s *ImplicitDirsTest) NothingPresent(t *ogletest.T) {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.ExpectThat(entries, ElementsAre())
}

func (s *ImplicitDirsTest) FileObjectPresent(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				// File
				"foo": "taco",
			}))

	// Statting the name should return an entry for the file.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(4, fi.Size())
	t.ExpectFalse(fi.IsDir())

	// ReadDir should show the file.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(4, fi.Size())
	t.ExpectFalse(fi.IsDir())
}

func (s *ImplicitDirsTest) DirectoryObjectPresent(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				// Directory
				"foo/": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) ImplicitDirectory_DefinedByFile(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/bar": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) ImplicitDirectory_DefinedByDirectory(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/bar/": "",
			}))

	// Statting the name should return an entry for the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// ReadDir should show the directory.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) ConflictingNames_PlaceholderPresent(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				// File
				"foo": "taco",

				// Directory
				"foo/": "",
			}))

	// A listing of the parent should contain a directory named "foo" and a
	// file named "foo\n".
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo\n"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectFalse(fi.IsDir())
}

func (s *ImplicitDirsTest) ConflictingNames_PlaceholderNotPresent(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				// File
				"foo": "taco",

				// Implicit directory
				"foo/bar": "",
			}))

	// A listing of the parent should contain a directory named "foo" and a
	// file named "foo\n".
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())

	fi = entries[1]
	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the file.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo\n"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectFalse(fi.IsDir())
}

func (s *ImplicitDirsTest) ConflictingNames_OneIsSymlink(t *ogletest.T) {
	var fi os.FileInfo
	var entries []os.FileInfo
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				// Symlink
				"foo": "",

				// Directory
				"foo/": "",
			}))

	// Cause "foo" to look like a symlink.
	err = setSymlinkTarget(t.Ctx, s.bucket, "foo", "")
	t.AssertEq(nil, err)

	// A listing of the parent should contain a directory named "foo" and a
	// symlink named "foo\n".
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	fi = entries[1]
	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)

	// Statting "foo" should yield the directory.
	fi, err = os.Lstat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Statting "foo\n" should yield the symlink.
	fi, err = os.Lstat(path.Join(s.mfs.Dir(), "foo\n"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo\n", fi.Name())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

func (s *ImplicitDirsTest) StatUnknownName_NoOtherContents(t *ogletest.T) {
	var err error

	// Stat an unknown name.
	_, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ImplicitDirsTest) StatUnknownName_UnrelatedContents(t *ogletest.T) {
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"bar": "",
				"baz": "",
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ImplicitDirsTest) StatUnknownName_PrefixOfActualNames(t *ogletest.T) {
	var err error

	// Set up contents.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foop":  "",
				"fooq/": "",
			}))

	// Stat an unknown name.
	_, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *ImplicitDirsTest) ImplicitBecomesExplicit(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Set up an implicit directory.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/bar": "",
			}))

	// Stat it.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Set up an explicit placeholder.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/": "",
			}))

	// Stat the directory again.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) ExplicitBecomesImplicit(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Set up an explicit directory.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/":    "",
				"foo/bar": "",
			}))

	// Stat it.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())

	// Remove the explicit placeholder.
	t.AssertEq(nil, s.bucket.DeleteObject(t.Ctx, "foo/"))

	// Stat the directory again.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) Rmdir_NotEmpty_OnlyImplicit(t *ogletest.T) {
	var err error

	// Set up an implicit directory.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/bar": "",
			}))

	// Attempt to remove it.
	err = os.Remove(path.Join(s.Dir, "foo"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) Rmdir_NotEmpty_ImplicitAndExplicit(t *ogletest.T) {
	var err error

	// Set up an implicit directory that also has a backing object.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/":    "",
				"foo/bar": "",
			}))

	// Attempt to remove it.
	err = os.Remove(path.Join(s.Dir, "foo"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not empty")))

	// It should still be there.
	fi, err := os.Lstat(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *ImplicitDirsTest) Rmdir_Empty(t *ogletest.T) {
	var err error
	var entries []os.FileInfo

	// Create two levels of directories. We can's make an empty implicit dir, so
	// there must be a backing object for each.
	t.AssertEq(
		nil,
		s.createObjects(
			t,
			map[string]string{
				"foo/":     "",
				"foo/bar/": "",
			}))

	// Remove the leaf.
	err = os.Remove(path.Join(s.Dir, "foo/bar"))
	t.AssertEq(nil, err)

	// There should be nothing left in the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)

	// Now the root directory should be empty, too.
	entries, err = fusetesting.ReadDirPicky(s.Dir)

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())
}

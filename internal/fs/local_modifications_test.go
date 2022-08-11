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

// A collection of tests for a file system backed by a GCS bucket, where we
// interact with the file system directly for creating and modifying files
// (rather than through the side channel of the GCS bucket itself).

package fs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

// The radius we use for "expect mtime is within"-style assertions. We can't
// share a synchronized clock with the ultimate source of mtimes because with
// writeback caching enabled the kernel manufactures them based on wall time.
const timeSlop = 25 * time.Millisecond

var fuseMaxNameLen int

func init() {
	switch runtime.GOOS {
	case "darwin":
		// FUSE_MAXNAMELEN is used on OS X in the kernel to limit the max length of
		// a name that readdir needs to process (cf. https://goo.gl/eega7V).
		//
		// NOTE(jacobsa): I can't find where this is defined, but this appears to
		// be its value.
		fuseMaxNameLen = 255

	case "linux":
		// On Linux, we're looking at FUSE_NAME_MAX (https://goo.gl/qd8G0f), used
		// in e.g. fuse_lookup_name (https://goo.gl/FHSAhy).
		fuseMaxNameLen = 1024

	default:
		panic(fmt.Sprintf("Unknown runtime.GOOS: %s", runtime.GOOS))
	}
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getFileOffset(f *os.File) (offset int64, err error) {
	const relativeToCurrent = 1
	offset, err = f.Seek(0, relativeToCurrent)
	return
}

// Return a collection of interesting names that should be legal to use.
func interestingLegalNames() (names []string) {
	names = []string{
		// Non-Roman scripts
		"타코",
		"世界",

		// Characters special to the shell
		"*![]&&||;",

		// Longest legal name
		strings.Repeat("a", fuseMaxNameLen),

		// Angstrom symbol singleton and normalized forms.
		// Cf. http://unicode.org/reports/tr15/
		"foo \u212b bar",
		"foo \u0041\u030a bar",
		"foo \u00c5 bar",

		// Hangul separating jamo
		// Cf. http://www.unicode.org/versions/Unicode7.0.0/ch18.pdf (Table 18-10)
		"foo \u3131\u314f bar",
		"foo \u1100\u1161 bar",
		"foo \uac00 bar",

		// Unicode specials
		// Cf. http://en.wikipedia.org/wiki/Specials_%28Unicode_block%29
		"foo \ufff9 bar",
		"foo \ufffa bar",
		"foo \ufffb bar",
		"foo \ufffc bar",
		"foo \ufffd bar",
	}

	// Most single-byte UTF-8 strings.
	for b := byte(0); b < utf8.RuneSelf; b++ {
		switch b {
		// NULL and '/' are not legal in file names.
		case 0, '/':
			continue

		// U+000A and U+000D are not legal in GCS.
		case '\u000a', '\u000d':
			continue
		}

		names = append(names, fmt.Sprintf("foo %c bar", b))
	}

	// All codepoints in Unicode general categories C* (control and special) and
	// Z* (space), except for:
	//
	//  *  Cn (non-character and reserved), which is not included in unicode.C.
	//  *  Co (private usage), which is large.
	//  *  Cs (surrages), which is large.
	//  *  U+0000, which is forbidden in paths by Go (cf. https://goo.gl/BHoO7N).
	//  *  U+000A and U+000D, which are forbidden by the docs.
	//
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !unicode.In(r, unicode.C) && !unicode.In(r, unicode.Z) {
			continue
		}

		if unicode.In(r, unicode.Co) {
			continue
		}

		if unicode.In(r, unicode.Cs) {
			continue
		}

		if r == 0x00 {
			continue
		}

		if r == 0x0a || r == 0x0d {
			continue
		}

		names = append(names, fmt.Sprintf("baz %s qux", string(r)))
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Open
////////////////////////////////////////////////////////////////////////

type OpenTest struct {
	fsTest
}

func init() { RegisterTestSuite(&OpenTest{}) }


////////////////////////////////////////////////////////////////////////
// File interaction
////////////////////////////////////////////////////////////////////////

type FileTest struct {
	fsTest
}

func init() { RegisterTestSuite(&FileTest{}) }

func (t *FileTest) WriteOverlapsEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 4 bytes long.
	err = t.f1.Truncate(4)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) WriteStartsAtEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 2 bytes long.
	err = t.f1.Truncate(2)
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) WriteStartsPastEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	AssertEq(nil, err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) WriteAtDoesntChangeOffset_NotAppendMode() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *FileTest) WriteAtDoesntChangeOffset_AppendMode() {
	var err error
	var n int

	// Create a file in append mode.
	t.f1, err = os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_CREATE,
		0600)

	AssertEq(nil, err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	AssertEq(nil, err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	AssertEq(nil, err)
	ExpectEq(4, offset)
}

func (t *FileTest) ReadsPastEndOfFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Read a range overlapping EOF.
	n, err = t.f1.ReadAt(buf[:4], 2)
	AssertEq(io.EOF, err)
	ExpectEq(2, n)
	ExpectEq("co", string(buf[:n]))

	// Read a range starting at EOF.
	n, err = t.f1.ReadAt(buf[:4], 4)
	AssertEq(io.EOF, err)
	ExpectEq(0, n)
	ExpectEq("", string(buf[:n]))

	// Read a range starting past EOF.
	n, err = t.f1.ReadAt(buf[:4], 100)
	AssertEq(io.EOF, err)
	ExpectEq(0, n)
	ExpectEq("", string(buf[:n]))
}

func (t *FileTest) Truncate_Smaller() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(2)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(2, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("ta", string(contents))
}

func (t *FileTest) Truncate_SameSize() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(4)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(4, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) Truncate_Larger() {
	var err error
	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	AssertEq(nil, err)

	// Truncate it.
	err = t.f1.Truncate(6)
	AssertEq(nil, err)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)
	ExpectEq(6, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	AssertEq(nil, err)
	ExpectEq("taco\x00\x00", string(contents))
}

func (t *FileTest) Seek() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Seek and overwrite.
	off, err := t.f1.Seek(1, 0)
	AssertEq(nil, err)
	AssertEq(1, off)

	n, err = t.f1.Write([]byte("xx"))
	AssertEq(nil, err)
	AssertEq(2, n)

	// Read full the contents of the file.
	n, err = t.f1.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	ExpectEq("txxo", string(buf[:n]))
}

func (t *FileTest) Stat() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	time.Sleep(timeSlop + timeSlop/2)
	writeTime := t.mtimeClock.Now()

	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	time.Sleep(timeSlop + timeSlop/2)

	// Stat it.
	fi, err := t.f1.Stat()
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(writeTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) StatUnopenedFile() {
	var err error

	// Create and close a file.
	time.Sleep(timeSlop + timeSlop/2)
	createTime := t.mtimeClock.Now()

	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte("taco"), 0700)
	AssertEq(nil, err)

	time.Sleep(timeSlop + timeSlop/2)

	// Stat it.
	fi, err := os.Stat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) LstatUnopenedFile() {
	var err error

	// Create and close a file.
	time.Sleep(timeSlop + timeSlop/2)
	createTime := t.mtimeClock.Now()

	err = ioutil.WriteFile(path.Join(t.mfs.Dir(), "foo"), []byte("taco"), 0700)
	AssertEq(nil, err)

	time.Sleep(timeSlop + timeSlop/2)

	// Lstat it.
	fi, err := os.Lstat(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) UnlinkFile_Exists() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	_, err = fusetesting.ReadDirPicky(t.mfs.Dir())
	AssertEq(nil, err)
//	ExpectThat(entries, ElementsAre())
}

func (t *FileTest) UnlinkFile_NonExistent() {
	err := os.Remove(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *FileTest) UnlinkFile_StillOpen() {
	var err error

	fileName := path.Join(t.mfs.Dir(), "foo")

	// Create and open a file.
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	AssertEq(nil, err)
	defer f.Close()

	// Write some data into it.
	n, err := f.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// The directory should no longer contain it.
	_, err = fusetesting.ReadDirPicky(t.mfs.Dir())
	AssertEq(nil, err)
//	ExpectThat(entries, ElementsAre())

	// We should be able to stat the file. It should still show as having
	// contents, but with no links.
	fi, err := f.Stat()

	AssertEq(nil, err)
	ExpectEq(4, fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// The contents should still be available.
	buf := make([]byte, 1024)
	n, err = f.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:4]))

	// Writing should still work, too.
	n, err = f.Write([]byte("burrito"))
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)
}

func (t *FileTest) UnlinkFile_NoLongerInBucket() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Delete it from the bucket through the back door.
	AssertEq(
		nil,
		t.bucket.DeleteObject(
			t.ctx,
			&gcs.DeleteObjectRequest{Name: "foo"}))

	AssertEq(nil, err)

	// Attempt to unlink it.
	err = os.Remove(fileName)

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *FileTest) UnlinkFile_FromSubDirectory() {
	var err error

	// Create a sub-directory.
	dirName := path.Join(t.mfs.Dir(), "dir")
	err = os.Mkdir(dirName, 0700)
	AssertEq(nil, err)

	// Write a file to that directory.
	fileName := path.Join(dirName, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := fusetesting.ReadDirPicky(dirName)
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *FileTest) UnlinkFile_ThenRecreateWithSameName() {
	var err error

	// Write a file.
	fileName := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	AssertEq(nil, err)

	// Re-create a file with the same name.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	AssertEq(nil, err)

	// Statting should result in a record for the new contents.
	fi, err := os.Stat(fileName)
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *FileTest) Chmod() {
	var err error

	// Write a file.
	p := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(p, []byte(""), 0700)
	AssertEq(nil, err)

	// Attempt to chmod it. Chmod should succeed even though we don't do anything
	// useful. The OS X Finder otherwise complains to the user when copying in a
	// file.
	err = os.Chmod(p, 0777)
	ExpectEq(nil, err)
}

func (t *FileTest) Chtimes_InactiveFile() {
	var err error

	// Create a file.
	p := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(p, []byte{}, 0600)
	AssertEq(nil, err)

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	AssertEq(nil, err)

	// Stat it and confirm that it worked.
	fi, err := os.Stat(p)
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) Chtimes_OpenFile_Clean() {
	var err error

	// Create a file.
	p := path.Join(t.mfs.Dir(), "foo")
	err = ioutil.WriteFile(p, []byte{}, 0600)
	AssertEq(nil, err)

	// Open it for reading.
	f, err := os.Open(p)
	AssertEq(nil, err)
	defer f.Close()

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	AssertEq(nil, err)

	// Stat it by path.
	fi, err := os.Stat(p)
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Stat it by fd.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Close the file, then stat it by path again.
	err = f.Close()
	AssertEq(nil, err)

	fi, err = os.Stat(p)
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) Chtimes_OpenFile_Dirty() {
	var err error

	// Create a file.
	p := path.Join(t.mfs.Dir(), "foo")
	f, err := os.Create(p)
	AssertEq(nil, err)
	defer f.Close()

	// Dirty the file.
	_, err = f.Write([]byte("taco"))
	AssertEq(nil, err)

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	AssertEq(nil, err)

	// Stat it by path.
	fi, err := os.Stat(p)
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Stat it by fd.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Close the file, then stat it by path again.
	err = f.Close()
	AssertEq(nil, err)

	fi, err = os.Stat(p)
	AssertEq(nil, err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) Sync_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Sync it.
	err = t.f1.Sync()
	AssertEq(nil, err)

	// The contents should now be in the bucket, even though we haven't closed
	// the file.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) Sync_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	// Sync the file.
	err = t.f1.Sync()
	AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	ExpectEq(o1.Generation, o2.Generation)
}

func (t *FileTest) Sync_Clobbered() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Dirty the file by giving it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Replace the underlying object with a new generation.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("foobar"))

	// Attempt to sync the file. This may result in an error if the OS has
	// decided to hold back the writes from above until now (in which case the
	// inode will fail to load the source object), or it may fail silently.
	// Either way, this should not result in a new generation being created.
	err = t.f1.Sync()
	if err != nil {
		ExpectThat(err, Error(HasSubstr("input/output error")))
	}

	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func (t *FileTest) Close_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Close it.
	err = t.f1.Close()
	t.f1 = nil
	AssertEq(nil, err)

	// The contents should now be in the bucket.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) Close_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	// Close the file.
	err = t.f1.Close()
	t.f1 = nil
	AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)

	ExpectEq(o1.Generation, o2.Generation)
}

func (t *FileTest) Close_Clobbered() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer f.Close()

	// Dirty the file by giving it some contents.
	n, err = f.Write([]byte("taco"))
	AssertEq(nil, err)
	AssertEq(4, n)

	// Replace the underlying object with a new generation.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		"foo",
		[]byte("foobar"))

	// Close the file. This may result in a "generation not found" error when
	// faulting in the object's contents on Linux where close may cause cached
	// writes to be delivered to the file system. But in any case the new
	// generation should not be replaced.
	f.Close()

	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("foobar", string(contents))
}

func (t *FileTest) AtimeAndCtime() {
	var err error

	// Create a file.
	p := path.Join(t.mfs.Dir(), "foo")
	createTime := t.mtimeClock.Now()
	err = ioutil.WriteFile(p, []byte{}, 0400)
	AssertEq(nil, err)

	// Stat it.
	fi, err := os.Stat(p)
	AssertEq(nil, err)

	// We require only that atime and ctime be "reasonable".
	atime, ctime, _ := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(createTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(createTime, delta))
}

func (t *FileTest) ContentTypes() {
	testCases := map[string]string{
		"foo.jpg": "image/jpeg",
		"bar.gif": "image/gif",
		"baz":     "",
	}

	runOne := func(name string, expected string) {
		p := path.Join(t.mfs.Dir(), name)

		// Create a file.
		f, err := os.Create(p)
		AssertEq(nil, err)
		defer f.Close()

		// Check the GCS content type.
		o, err := t.bucket.StatObject(t.ctx, &gcs.StatObjectRequest{Name: name})
		AssertEq(nil, err)
		ExpectEq(expected, o.ContentType, "name: %q", name)

		// Modify the file and cause a new generation to be written out.
		_, err = f.Write([]byte("taco"))
		AssertEq(nil, err)

		err = f.Sync()
		AssertEq(nil, err)

		// The GCS content type should still be correct.
		o, err = t.bucket.StatObject(t.ctx, &gcs.StatObjectRequest{Name: name})
		AssertEq(nil, err)
		ExpectEq(expected, o.ContentType, "name: %q", name)
	}

	for name, expected := range testCases {
		runOne(name, expected)
	}
}

////////////////////////////////////////////////////////////////////////
// Symlinks
////////////////////////////////////////////////////////////////////////


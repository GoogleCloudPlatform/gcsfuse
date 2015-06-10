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
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

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

func init() { ogletest.RegisterTestSuite(&OpenTest{}) }

func (s *OpenTest) NonExistent_CreateFlagNotSet(t *ogletest.T) {
	var err error
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR, 0700)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("no such file")))

	// No object should have been created.
	_, err = gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (s *OpenTest) NonExistent_CreateFlagSet(t *ogletest.T) {
	var err error

	// Open the file.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_CREATE,
		0700)

	t.AssertEq(nil, err)

	// The object should now be present in the bucket, with empty contents.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.AssertEq(nil, err)
	t.ExpectEq("", string(contents))

	// Write some contents.
	_, err = s.f1.Write([]byte("012"))
	t.AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = s.f1.Seek(1, 0)
	t.AssertEq(nil, err)

	buf := make([]byte, 2)
	_, err = io.ReadFull(s.f1, buf)

	t.AssertEq(nil, err)
	t.ExpectEq("12", string(buf))

	// Close the file.
	t.AssertEq(nil, s.f1.Close())
	s.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("012", string(fileContents))
}

func (s *OpenTest) ExistingFile(t *ogletest.T) {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// Write to the start of the file using File.Write.
	_, err = s.f1.Write([]byte("012"))
	t.AssertEq(nil, err)

	// Read some contents with Seek and Read.
	_, err = s.f1.Seek(2, 0)
	t.AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(s.f1, buf)

	t.AssertEq(nil, err)
	t.ExpectEq("2obu", string(buf))

	// Close the file.
	t.AssertEq(nil, s.f1.Close())
	s.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("012oburritoenchilada", string(fileContents))
}

func (s *OpenTest) ExistingFile_Truncate(t *ogletest.T) {
	var err error

	// Create a file.
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte("blahblahblah"),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_TRUNC,
		0)

	t.AssertEq(nil, err)

	// The file should be empty.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(0, fi.Size())

	// Write to the start of the file using File.Write.
	_, err = s.f1.Write([]byte("012"))
	t.AssertEq(nil, err)

	// Read the contents.
	_, err = s.f1.Seek(0, 0)
	t.AssertEq(nil, err)

	contentsSlice, err := ioutil.ReadAll(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq("012", string(contentsSlice))

	// Close the file.
	t.AssertEq(nil, s.f1.Close())
	s.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("012", string(fileContents))
}

func (s *OpenTest) AlreadyOpenedFile(t *ogletest.T) {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create and open a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Write some data into it.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Open another handle for reading and writing.
	s.f2, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// The contents written through the first handle should be available to the
	// second handle..
	n, err = s.f2.Read(buf[:2])
	t.AssertEq(nil, err)
	t.AssertEq(2, n)
	t.ExpectEq("ta", string(buf[:n]))

	// Write some contents with the second handle, which should now be at offset
	// 2.
	n, err = s.f2.Write([]byte("nk"))
	t.AssertEq(nil, err)
	t.AssertEq(2, n)

	// Check the overall contents now.
	contents, err := ioutil.ReadFile(s.f2.Name())
	t.AssertEq(nil, err)
	t.ExpectEq("tank", string(contents))
}

func (s *OpenTest) LegalNames(t *ogletest.T) {
	var err error

	names := interestingLegalNames()
	sort.Strings(names)

	// We should be able to create each name.
	for _, n := range names {
		err = ioutil.WriteFile(path.Join(s.Dir, n), []byte(n), 0400)
		t.AssertEq(nil, err, "Name: %q", n)
	}

	// A listing should contain them all.
	entries, err := fusetesting.ReadDirPicky(s.Dir)
	t.AssertEq(nil, err)

	t.AssertEq(len(names), len(entries))
	for i, n := range names {
		t.ExpectEq(n, entries[i].Name(), "Name: %q", n)
		t.ExpectEq(len(n), entries[i].Size(), "Name: %q", n)
	}

	// We should be able to read them all.
	for _, n := range names {
		contents, err := ioutil.ReadFile(path.Join(s.Dir, n))
		t.AssertEq(nil, err, "Name: %q", n)
		t.ExpectEq(n, string(contents), "Name: %q", n)
	}

	// And delete each.
	for _, n := range names {
		err = os.Remove(path.Join(s.Dir, n))
		t.AssertEq(nil, err, "Name: %q", n)
	}
}

func (s *OpenTest) IllegalNames(t *ogletest.T) {
	var err error

	// A collection of interesting names that are illegal to use, and a string we
	// expect to see in the associated error.
	testCases := []struct {
		name string
		err  string
	}{
		// Too long
		{strings.Repeat("a", fuseMaxNameLen+1), "name too long"},

		// Invalid UTF-8, rejected by GCS
		{"\x80", "input/output"},
	}

	// We should not be able to create any of these names.
	for _, tc := range testCases {
		err = ioutil.WriteFile(path.Join(s.Dir, tc.name), []byte{}, 0400)
		t.ExpectThat(err, Error(HasSubstr(tc.err)), "Name: %q", tc.name)
	}
}

////////////////////////////////////////////////////////////////////////
// Modes
////////////////////////////////////////////////////////////////////////

type ModesTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&ModesTest{}) }

func (s *ModesTest) ReadOnlyMode(t *ogletest.T) {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	t.AssertEq(nil, err)

	// Read its contents.
	fileContents, err := ioutil.ReadAll(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := s.f1.Write([]byte("taco"))

	t.AssertEq(0, n)
	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (s *ModesTest) WriteOnlyMode(t *ogletest.T) {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_WRONLY, 0)
	t.AssertEq(nil, err)

	// Reading should fail.
	_, err = ioutil.ReadAll(s.f1)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = s.f1.Write([]byte("000"))
	t.AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = s.f1.WriteAt([]byte("111"), 4)
	t.AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = s.f1.Seek(int64(len(contents)), 0)
	t.AssertEq(nil, err)

	_, err = s.f1.Write([]byte("222"))
	t.AssertEq(nil, err)

	// Check the size now.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	t.AssertEq(nil, s.f1.Close())
	s.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (s *ModesTest) ReadWriteMode(t *ogletest.T) {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// Write to the start of the file using File.Write.
	_, err = s.f1.Write([]byte("000"))
	t.AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = s.f1.WriteAt([]byte("111"), 4)
	t.AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = s.f1.Seek(int64(len(contents)), 0)
	t.AssertEq(nil, err)

	_, err = s.f1.Write([]byte("222"))
	t.AssertEq(nil, err)

	// Check the size now.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = s.f1.Seek(4, 0)
	t.AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(s.f1, buf)

	t.AssertEq(nil, err)
	t.ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = s.f1.ReadAt(buf, 0)

	t.AssertEq(nil, err)
	t.ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	t.AssertEq(nil, s.f1.Close())
	s.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (s *ModesTest) AppendMode_SeekAndWrite(t *ogletest.T) {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	t.AssertEq(nil, err)

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = s.f1.Seek(1, 0)
	t.AssertEq(nil, err)

	_, err = s.f1.Write([]byte("222"))
	t.AssertEq(nil, err)

	// The seek position should have been updated.
	off, err := getFileOffset(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(len(contents)+len("222"), off)

	// Check the size now.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(len(contents)+len("222"), fi.Size())

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := s.f1.ReadAt(buf, 0)

	t.AssertEq(io.EOF, err)
	t.ExpectEq(contents+"222", string(buf[:n]))

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq(contents+"222", string(fileContents))
}

func (s *ModesTest) AppendMode_WriteAt(t *ogletest.T) {
	var err error

	// Linux's support for pwrite is buggy; the pwrite(2) man page says this:
	//
	//     POSIX requires that opening a file with the O_APPEND flag should have
	//     no affect on the location at which pwrite() writes data.  However, on
	//     Linux,  if  a  file  is opened with O_APPEND, pwrite() appends data to
	//     the end of the file, regardless of the value of offset.
	//
	isLinux := (runtime.GOOS == "linux")

	// Create a file.
	const contents = "tacoburritoenchilada"
	t.AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(s.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	s.f1, err = os.OpenFile(path.Join(s.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	t.AssertEq(nil, err)

	// Seek somewhere in the file.
	_, err = s.f1.Seek(1, 0)
	t.AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = s.f1.WriteAt([]byte("111"), 4)
	t.AssertEq(nil, err)

	// The seek position should have been unaffected.
	off, err := getFileOffset(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(1, off)

	// Check the size now.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)

	if isLinux {
		t.ExpectEq(len(contents+"111"), fi.Size())
	} else {
		t.ExpectEq(len(contents), fi.Size())
	}

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := s.f1.ReadAt(buf, 0)

	t.AssertEq(io.EOF, err)
	if isLinux {
		t.ExpectEq("tacoburritoenchilada111", string(buf[:n]))
	} else {
		t.ExpectEq("taco111ritoenchilada", string(buf[:n]))
	}

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	if isLinux {
		t.ExpectEq("tacoburritoenchilada111", string(fileContents))
	} else {
		t.ExpectEq("taco111ritoenchilada", string(fileContents))
	}
}

func (s *ModesTest) AppendMode_WriteAt_PastEOF(t *ogletest.T) {
	var err error

	// Linux's support for pwrite is buggy; the pwrite(2) man page says this:
	//
	//     POSIX requires that opening a file with the O_APPEND flag should have
	//     no affect on the location at which pwrite() writes data.  However, on
	//     Linux,  if  a  file  is opened with O_APPEND, pwrite() appends data to
	//     the end of the file, regardless of the value of offset.
	//
	isLinux := (runtime.GOOS == "linux")

	// Open a file.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	t.AssertEq(nil, err)

	// Write three bytes.
	n, err := s.f1.Write([]byte("111"))
	t.AssertEq(nil, err)
	t.AssertEq(3, n)

	// Write at offset six.
	n, err = s.f1.WriteAt([]byte("222"), 6)
	t.AssertEq(nil, err)
	t.AssertEq(3, n)

	// The seek position should have been unaffected.
	off, err := getFileOffset(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(3, off)

	// Read the full contents of the file.
	contents, err := ioutil.ReadFile(s.f1.Name())
	t.AssertEq(nil, err)

	if isLinux {
		t.ExpectEq("111222", string(contents))
	} else {
		t.ExpectEq("111\x00\x00\x00222", string(contents))
	}
}

func (s *ModesTest) ReadFromWriteOnlyFile(t *ogletest.T) {
	var err error

	// Create and open a file for writing.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_WRONLY|os.O_CREATE,
		0700)

	t.AssertEq(nil, err)

	// Attempt to read from it.
	_, err = s.f1.Read(make([]byte, 1024))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (s *ModesTest) WriteToReadOnlyFile(t *ogletest.T) {
	var err error

	// Create and open a file for reading.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_RDONLY|os.O_CREATE,
		0700)

	t.AssertEq(nil, err)

	// Attempt to write s it.
	_, err = s.f1.Write([]byte("taco"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

////////////////////////////////////////////////////////////////////////
// Directory interaction
////////////////////////////////////////////////////////////////////////

type DirectoryTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&DirectoryTest{}) }

func (s *DirectoryTest) Mkdir_OneLevel(t *ogletest.T) {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	dirName := path.Join(s.mfs.Dir(), "dir")

	// Create a directory within the root.
	err = os.Mkdir(dirName, 0754)
	t.AssertEq(nil, err)

	// Stat the directory.
	fi, err = os.Stat(dirName)

	t.AssertEq(nil, err)
	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = fusetesting.ReadDirPicky(dirName)

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())

	// Read the root.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())

	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
}

func (s *DirectoryTest) Mkdir_TwoLevels(t *ogletest.T) {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	// Create a directory within the root.
	err = os.Mkdir(path.Join(s.mfs.Dir(), "parent"), 0700)
	t.AssertEq(nil, err)

	// Create a child of that directory.
	err = os.Mkdir(path.Join(s.mfs.Dir(), "parent/dir"), 0754)
	t.AssertEq(nil, err)

	// Stat the directory.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "parent/dir"))

	t.AssertEq(nil, err)
	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.mfs.Dir(), "parent/dir"))

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())

	// Read the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.mfs.Dir(), "parent"))

	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
}

func (s *DirectoryTest) Mkdir_AlreadyExists(t *ogletest.T) {
	var err error
	dirName := path.Join(s.mfs.Dir(), "dir")

	// Create the directory once.
	err = os.Mkdir(dirName, 0754)
	t.AssertEq(nil, err)

	// Attempt to create it again.
	err = os.Mkdir(dirName, 0754)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("exists")))
}

func (s *DirectoryTest) Mkdir_IntermediateIsFile(t *ogletest.T) {
	var err error

	// Create a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte{}, 0700)
	t.AssertEq(nil, err)

	// Attempt to create a directory within the file.
	dirName := path.Join(fileName, "dir")
	err = os.Mkdir(dirName, 0754)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not a directory")))
}

func (s *DirectoryTest) Mkdir_IntermediateIsNonExistent(t *ogletest.T) {
	var err error

	// Attempt to create a sub-directory of a non-existent sub-directory.
	dirName := path.Join(s.mfs.Dir(), "foo/dir")
	err = os.Mkdir(dirName, 0754)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("no such file or directory")))
}

func (s *DirectoryTest) Stat_Root(t *ogletest.T) {
	fi, err := os.Stat(s.mfs.Dir())
	t.AssertEq(nil, err)

	t.ExpectEq(path.Base(s.mfs.Dir()), fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *DirectoryTest) Stat_FirstLevelDirectory(t *ogletest.T) {
	var err error

	// Create a sub-directory.
	err = os.Mkdir(path.Join(s.mfs.Dir(), "dir"), 0700)
	t.AssertEq(nil, err)

	// Stat it.
	fi, err := os.Stat(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *DirectoryTest) Stat_SecondLevelDirectory(t *ogletest.T) {
	var err error

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(s.mfs.Dir(), "parent/dir"), 0700)
	t.AssertEq(nil, err)

	// Stat it.
	fi, err := os.Stat(path.Join(s.mfs.Dir(), "parent/dir"))
	t.AssertEq(nil, err)

	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *DirectoryTest) ReadDir_Root(t *ogletest.T) {
	var err error
	var fi os.FileInfo

	// Create a file and a directory.
	s.clock.AdvanceTime(time.Second)
	createTime := s.clock.Now()

	err = ioutil.WriteFile(path.Join(s.mfs.Dir(), "bar"), []byte("taco"), 0700)
	t.AssertEq(nil, err)

	err = os.Mkdir(path.Join(s.mfs.Dir(), "foo"), 0700)
	t.AssertEq(nil, err)

	s.clock.AdvanceTime(time.Second)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectThat(fi.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *DirectoryTest) ReadDir_SubDirectory(t *ogletest.T) {
	var err error
	var fi os.FileInfo

	// Create a directory.
	parent := path.Join(s.mfs.Dir(), "parent")
	err = os.Mkdir(parent, 0700)
	t.AssertEq(nil, err)

	// Create a file and a directory within it.
	s.clock.AdvanceTime(time.Second)
	createTime := s.clock.Now()

	err = ioutil.WriteFile(path.Join(parent, "bar"), []byte("taco"), 0700)
	t.AssertEq(nil, err)

	err = os.Mkdir(path.Join(parent, "foo"), 0700)
	t.AssertEq(nil, err)

	s.clock.AdvanceTime(time.Second)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(parent)
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectThat(fi.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	t.ExpectTrue(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *DirectoryTest) Rmdir_NotEmpty(t *ogletest.T) {
	var err error

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(s.Dir, "foo/bar"), 0754)
	t.AssertEq(nil, err)

	// Attempt to remove the parent.
	err = os.Remove(path.Join(s.Dir, "foo"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not empty")))

	// The parent should still be there.
	fi, err := os.Lstat(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *DirectoryTest) Rmdir_Empty(t *ogletest.T) {
	var err error
	var entries []os.FileInfo

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(s.mfs.Dir(), "foo/bar"), 0754)
	t.AssertEq(nil, err)

	// Remove the leaf.
	err = os.Remove(path.Join(s.mfs.Dir(), "foo/bar"))
	t.AssertEq(nil, err)

	// There should be nothing left in the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(s.mfs.Dir(), "foo"))

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())

	// Remove the parent.
	err = os.Remove(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Now the root directory should be empty, too.
	entries, err = fusetesting.ReadDirPicky(s.mfs.Dir())

	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())
}

func (s *DirectoryTest) Rmdir_OpenedForReading(t *ogletest.T) {
	var err error

	// Create a directory.
	err = os.Mkdir(path.Join(s.mfs.Dir(), "dir"), 0700)
	t.AssertEq(nil, err)

	// Open the directory for reading.
	s.f1, err = os.Open(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	// Remove the directory.
	err = os.Remove(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	// Create a new directory, with the same name even, and add some contents
	// within it.
	err = os.MkdirAll(path.Join(s.mfs.Dir(), "dir/foo"), 0700)
	t.AssertEq(nil, err)

	err = os.MkdirAll(path.Join(s.mfs.Dir(), "dir/bar"), 0700)
	t.AssertEq(nil, err)

	err = os.MkdirAll(path.Join(s.mfs.Dir(), "dir/baz"), 0700)
	t.AssertEq(nil, err)

	// We should still be able to stat the open file handle.
	fi, err := s.f1.Stat()
	t.ExpectEq("dir", fi.Name())

	// Attempt to read from the directory. Unfortunately we can's implement the
	// guarantee that no new entries are returned, but nothing crazy should
	// happen.
	_, err = s.f1.Readdirnames(0)
	if err != nil {
		t.ExpectThat(err, Error(HasSubstr("no such file")))
	}
}

func (s *DirectoryTest) Rmdir_ThenRecreateWithSameName(t *ogletest.T) {
	var err error

	// Create a directory.
	err = os.Mkdir(path.Join(s.mfs.Dir(), "dir"), 0700)
	t.AssertEq(nil, err)

	// Unlink the directory.
	err = os.Remove(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	// Re-create the directory with the same name. Nothing crazy should happen.
	// In the past, this used to crash (cf.
	// https://github.com/GoogleCloudPlatform/gcsfuse/issues/8).
	err = os.Mkdir(path.Join(s.mfs.Dir(), "dir"), 0700)
	t.AssertEq(nil, err)

	// Statting should reveal nothing surprising.
	fi, err := os.Stat(path.Join(s.mfs.Dir(), "dir"))
	t.AssertEq(nil, err)

	t.ExpectEq("dir", fi.Name())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectTrue(fi.IsDir())
}

func (s *DirectoryTest) CreateHardLink(t *ogletest.T) {
	var err error

	// Write a file.
	err = ioutil.WriteFile(path.Join(s.mfs.Dir(), "foo"), []byte(""), 0700)
	t.AssertEq(nil, err)

	// Attempt to hard link it. We don's support doing so.
	err = os.Link(
		path.Join(s.mfs.Dir(), "foo"),
		path.Join(s.mfs.Dir(), "bar"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not implemented")))
}

////////////////////////////////////////////////////////////////////////
// File interaction
////////////////////////////////////////////////////////////////////////

type FileTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&FileTest{}) }

func (s *FileTest) WriteOverlapsEndOfFile(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Make it 4 bytes long.
	err = s.f1.Truncate(4)
	t.AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = s.f1.WriteAt([]byte("taco"), 2)
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq("\x00\x00taco", string(contents))
}

func (s *FileTest) WriteStartsAtEndOfFile(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Make it 2 bytes long.
	err = s.f1.Truncate(2)
	t.AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = s.f1.WriteAt([]byte("taco"), 2)
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq("\x00\x00taco", string(contents))
}

func (s *FileTest) WriteStartsPastEndOfFile(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Write the range [2, 6).
	n, err = s.f1.WriteAt([]byte("taco"), 2)
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq("\x00\x00taco", string(contents))
}

func (s *FileTest) WriteAtDoesntChangeOffset_NotAppendMode(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Make it 16 bytes long.
	err = s.f1.Truncate(16)
	t.AssertEq(nil, err)

	// Seek to offset 4.
	_, err = s.f1.Seek(4, 0)
	t.AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = s.f1.WriteAt([]byte("taco"), 2)
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(4, offset)
}

func (s *FileTest) WriteAtDoesntChangeOffset_AppendMode(t *ogletest.T) {
	var err error
	var n int

	// Create a file in append mode.
	s.f1, err = os.OpenFile(
		path.Join(s.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	t.AssertEq(nil, err)

	// Make it 16 bytes long.
	err = s.f1.Truncate(16)
	t.AssertEq(nil, err)

	// Seek to offset 4.
	_, err = s.f1.Seek(4, 0)
	t.AssertEq(nil, err)

	// Write the range [10, 14).
	n, err = s.f1.WriteAt([]byte("taco"), 2)
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(s.f1)
	t.AssertEq(nil, err)
	t.ExpectEq(4, offset)
}

func (s *FileTest) ReadsPastEndOfFile(t *ogletest.T) {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Give it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Read a range overlapping EOF.
	n, err = s.f1.ReadAt(buf[:4], 2)
	t.AssertEq(io.EOF, err)
	t.ExpectEq(2, n)
	t.ExpectEq("co", string(buf[:n]))

	// Read a range starting at EOF.
	n, err = s.f1.ReadAt(buf[:4], 4)
	t.AssertEq(io.EOF, err)
	t.ExpectEq(0, n)
	t.ExpectEq("", string(buf[:n]))

	// Read a range starting past EOF.
	n, err = s.f1.ReadAt(buf[:4], 100)
	t.AssertEq(io.EOF, err)
	t.ExpectEq(0, n)
	t.ExpectEq("", string(buf[:n]))
}

func (s *FileTest) Truncate_Smaller(t *ogletest.T) {
	var err error
	fileName := path.Join(s.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	t.AssertEq(nil, err)

	// Open it for modification.
	s.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// Truncate it.
	err = s.f1.Truncate(2)
	t.AssertEq(nil, err)

	// Stat it.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(2, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	t.AssertEq(nil, err)
	t.ExpectEq("ta", string(contents))
}

func (s *FileTest) Truncate_SameSize(t *ogletest.T) {
	var err error
	fileName := path.Join(s.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	t.AssertEq(nil, err)

	// Open it for modification.
	s.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// Truncate it.
	err = s.f1.Truncate(4)
	t.AssertEq(nil, err)

	// Stat it.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(4, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	t.AssertEq(nil, err)
	t.ExpectEq("taco", string(contents))
}

func (s *FileTest) Truncate_Larger(t *ogletest.T) {
	var err error
	fileName := path.Join(s.mfs.Dir(), "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	t.AssertEq(nil, err)

	// Open it for modification.
	s.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	t.AssertEq(nil, err)

	// Truncate it.
	err = s.f1.Truncate(6)
	t.AssertEq(nil, err)

	// Stat it.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)
	t.ExpectEq(6, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	t.AssertEq(nil, err)
	t.ExpectEq("taco\x00\x00", string(contents))
}

func (s *FileTest) Seek(t *ogletest.T) {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Give it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Seek and overwrite.
	off, err := s.f1.Seek(1, 0)
	t.AssertEq(nil, err)
	t.AssertEq(1, off)

	n, err = s.f1.Write([]byte("xx"))
	t.AssertEq(nil, err)
	t.AssertEq(2, n)

	// Read full the contents of the file.
	n, err = s.f1.ReadAt(buf, 0)
	t.AssertEq(io.EOF, err)
	t.ExpectEq("txxo", string(buf[:n]))
}

func (s *FileTest) Stat(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Give it some contents.
	s.clock.AdvanceTime(time.Second)
	writeTime := s.clock.Now()

	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	s.clock.AdvanceTime(time.Second)

	// Stat it.
	fi, err := s.f1.Stat()
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectThat(fi.ModTime(), timeutil.TimeEq(writeTime))
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *FileTest) StatUnopenedFile(t *ogletest.T) {
	var err error

	// Create and close a file.
	s.clock.AdvanceTime(time.Second)
	createTime := s.clock.Now()

	err = ioutil.WriteFile(path.Join(s.mfs.Dir(), "foo"), []byte("taco"), 0700)
	t.AssertEq(nil, err)

	s.clock.AdvanceTime(time.Second)

	// Stat it.
	fi, err := os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectThat(fi.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *FileTest) LstatUnopenedFile(t *ogletest.T) {
	var err error

	// Create and close a file.
	s.clock.AdvanceTime(time.Second)
	createTime := s.clock.Now()

	err = ioutil.WriteFile(path.Join(s.mfs.Dir(), "foo"), []byte("taco"), 0700)
	t.AssertEq(nil, err)

	s.clock.AdvanceTime(time.Second)

	// Lstat it.
	fi, err := os.Lstat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
	t.ExpectThat(fi.ModTime(), timeutil.TimeEq(createTime))
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	t.ExpectEq(currentUid(t), fi.Sys().(*syscall.Stat_t).Uid)
	t.ExpectEq(currentGid(t), fi.Sys().(*syscall.Stat_t).Gid)
}

func (s *FileTest) UnlinkFile_Exists(t *ogletest.T) {
	var err error

	// Write a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	t.AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	t.AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())
}

func (s *FileTest) UnlinkFile_NonExistent(t *ogletest.T) {
	err := os.Remove(path.Join(s.mfs.Dir(), "foo"))

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("no such file")))
}

func (s *FileTest) UnlinkFile_StillOpen(t *ogletest.T) {
	var err error

	fileName := path.Join(s.mfs.Dir(), "foo")

	// Create and open a file.
	s.f1, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	t.AssertEq(nil, err)

	// Write some data into it.
	n, err := s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Unlink it.
	err = os.Remove(fileName)
	t.AssertEq(nil, err)

	// The directory should no longer contain it.
	entries, err := fusetesting.ReadDirPicky(s.mfs.Dir())
	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())

	// We should be able to stat the file. It should still show as having
	// contents, but with no links.
	fi, err := s.f1.Stat()

	t.AssertEq(nil, err)
	t.ExpectEq(4, fi.Size())
	t.ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// The contents should still be available.
	buf := make([]byte, 1024)
	n, err = s.f1.ReadAt(buf, 0)

	t.AssertEq(io.EOF, err)
	t.AssertEq(4, n)
	t.ExpectEq("taco", string(buf[:4]))

	// Writing should still work, too.
	n, err = s.f1.Write([]byte("burrito"))
	t.AssertEq(nil, err)
	t.AssertEq(len("burrito"), n)
}

func (s *FileTest) UnlinkFile_NoLongerInBucket(t *ogletest.T) {
	var err error

	// Write a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	t.AssertEq(nil, err)

	// Delete it from the bucket through the back door.
	err = s.bucket.DeleteObject(t.Ctx, "foo")
	t.AssertEq(nil, err)

	// Attempt to unlink it.
	err = os.Remove(fileName)

	t.AssertNe(nil, err)
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *FileTest) UnlinkFile_FromSubDirectory(t *ogletest.T) {
	var err error

	// Create a sub-directory.
	dirName := path.Join(s.mfs.Dir(), "dir")
	err = os.Mkdir(dirName, 0700)
	t.AssertEq(nil, err)

	// Write a file to that directory.
	fileName := path.Join(dirName, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	t.AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	t.AssertEq(nil, err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := fusetesting.ReadDirPicky(dirName)
	t.AssertEq(nil, err)
	t.ExpectThat(entries, ElementsAre())
}

func (s *FileTest) UnlinkFile_ThenRecreateWithSameName(t *ogletest.T) {
	var err error

	// Write a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	t.AssertEq(nil, err)

	// Unlink it.
	err = os.Remove(fileName)
	t.AssertEq(nil, err)

	// Re-create a file with the same name.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	t.AssertEq(nil, err)

	// Statting should result in a record for the new contents.
	fi, err := os.Stat(fileName)
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectEq(len("taco"), fi.Size())
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (s *FileTest) Chmod(t *ogletest.T) {
	var err error

	// Write a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte(""), 0700)
	t.AssertEq(nil, err)

	// Attempt to chmod it. We don's support doing so.
	err = os.Chmod(fileName, 0777)

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (s *FileTest) Chtimes(t *ogletest.T) {
	var err error

	// Write a file.
	fileName := path.Join(s.mfs.Dir(), "foo")
	err = ioutil.WriteFile(fileName, []byte(""), 0700)
	t.AssertEq(nil, err)

	// Attempt to change its atime and mtime. We don's support doing so.
	err = os.Chtimes(fileName, time.Now(), time.Now())

	t.AssertNe(nil, err)
	t.ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (s *FileTest) Sync_Dirty(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Give it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Sync it.
	err = s.f1.Sync()
	t.AssertEq(nil, err)

	// The contents should now be in the bucket, even though we haven's closed
	// the file.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.AssertEq(nil, err)
	t.ExpectEq("taco", string(contents))
}

func (s *FileTest) Sync_NotDirty(t *ogletest.T) {
	var err error

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := s.bucket.StatObject(t.Ctx, statReq)
	t.AssertEq(nil, err)

	// Sync the file.
	err = s.f1.Sync()
	t.AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := s.bucket.StatObject(t.Ctx, statReq)
	t.AssertEq(nil, err)

	t.ExpectEq(o1.Generation, o2.Generation)
}

func (s *FileTest) Sync_Clobbered(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Dirty the file by giving it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Replace the underyling object with a new generation.
	_, err = gcsutil.CreateObject(
		t.Ctx,
		s.bucket,
		"foo",
		"foobar")

	// Sync the file. This should not result in an error, but the new generation
	// should not be replaced.
	err = s.f1.Sync()
	t.AssertEq(nil, err)

	// Check that the new generation was not replaced.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.AssertEq(nil, err)
	t.ExpectEq("foobar", string(contents))
}

func (s *FileTest) Close_Dirty(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Give it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Close it.
	err = s.f1.Close()
	s.f1 = nil
	t.AssertEq(nil, err)

	// The contents should now be in the bucket.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.AssertEq(nil, err)
	t.ExpectEq("taco", string(contents))
}

func (s *FileTest) Close_NotDirty(t *ogletest.T) {
	var err error

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}

	o1, err := s.bucket.StatObject(t.Ctx, statReq)
	t.AssertEq(nil, err)

	// Close the file.
	err = s.f1.Close()
	s.f1 = nil
	t.AssertEq(nil, err)

	// A new generation need not have been written.
	o2, err := s.bucket.StatObject(t.Ctx, statReq)
	t.AssertEq(nil, err)

	t.ExpectEq(o1.Generation, o2.Generation)
}

func (s *FileTest) Close_Clobbered(t *ogletest.T) {
	var err error
	var n int

	// Create a file.
	s.f1, err = os.Create(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	// Dirty the file by giving it some contents.
	n, err = s.f1.Write([]byte("taco"))
	t.AssertEq(nil, err)
	t.AssertEq(4, n)

	// Replace the underyling object with a new generation.
	_, err = gcsutil.CreateObject(
		t.Ctx,
		s.bucket,
		"foo",
		"foobar")

	// Close the file. This should not result in an error, but the new generation
	// should not be replaced.
	err = s.f1.Close()
	s.f1 = nil
	t.AssertEq(nil, err)

	// Check that the new generation was not replaced.
	contents, err := gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.AssertEq(nil, err)
	t.ExpectEq("foobar", string(contents))
}

////////////////////////////////////////////////////////////////////////
// Symlinks
////////////////////////////////////////////////////////////////////////

type SymlinkTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&SymlinkTest{}) }

func (s *SymlinkTest) CreateLink(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Create a file.
	fileName := path.Join(s.Dir, "foo")
	const contents = "taco"

	err = ioutil.WriteFile(fileName, []byte(contents), 0400)
	t.AssertEq(nil, err)

	// Create a symlink to it.
	symlinkName := path.Join(s.Dir, "bar")
	err = os.Symlink("foo", symlinkName)
	t.AssertEq(nil, err)

	// Check the object in the bucket.
	o, err := s.bucket.StatObject(t.Ctx, &gcs.StatObjectRequest{Name: "bar"})

	t.AssertEq(nil, err)
	t.ExpectEq(0, o.Size)
	t.ExpectEq("foo", o.Metadata["gcsfuse_symlink_target"])

	// Read the link.
	target, err := os.Readlink(symlinkName)
	t.AssertEq(nil, err)
	t.ExpectEq("foo", target)

	// Stat the link.
	fi, err = os.Lstat(symlinkName)
	t.AssertEq(nil, err)

	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Read the parent directory.
	entries, err := fusetesting.ReadDirPicky(s.Dir)
	t.AssertEq(nil, err)
	t.AssertEq(2, len(entries))

	fi = entries[0]
	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(0, fi.Size())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Stat the target via the link.
	fi, err = os.Stat(symlinkName)
	t.AssertEq(nil, err)

	t.ExpectEq("bar", fi.Name())
	t.ExpectEq(len(contents), fi.Size())
	t.ExpectEq(filePerms, fi.Mode())
}

func (s *SymlinkTest) CreateLink_Exists(t *ogletest.T) {
	var err error

	// Create a file and a directory.
	fileName := path.Join(s.Dir, "foo")
	err = ioutil.WriteFile(fileName, []byte{}, 0400)
	t.AssertEq(nil, err)

	dirName := path.Join(s.Dir, "bar")
	err = os.Mkdir(dirName, 0700)
	t.AssertEq(nil, err)

	// Create an existing symlink.
	symlinkName := path.Join(s.Dir, "baz")
	err = os.Symlink("blah", symlinkName)
	t.AssertEq(nil, err)

	// Symlinking on top of any of them should fail.
	names := []string{
		fileName,
		dirName,
		symlinkName,
	}

	for _, n := range names {
		err = os.Symlink("blah", n)
		t.ExpectThat(err, Error(HasSubstr("exists")))
	}
}

func (s *SymlinkTest) RemoveLink(t *ogletest.T) {
	var err error

	// Create the link.
	symlinkName := path.Join(s.Dir, "foo")
	err = os.Symlink("blah", symlinkName)
	t.AssertEq(nil, err)

	// Remove it.
	err = os.Remove(symlinkName)
	t.AssertEq(nil, err)

	// It should be gone from the bucket.
	_, err = gcsutil.ReadObject(t.Ctx, s.bucket, "foo")
	t.ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

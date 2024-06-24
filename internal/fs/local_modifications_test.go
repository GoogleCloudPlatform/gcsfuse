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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestLocalModificationsTestSuite(t *testing.T) {
	suite.Run(t, new(OpenTest))
	suite.Run(t, new(MknodTest))
	suite.Run(t, new(ModesTest))
	suite.Run(t, new(DirectoryTest))
	suite.Run(t, new(FileTest))
	suite.Run(t, new(SymlinkTest))
	suite.Run(t, new(RenameTest))
}

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
		// NOTE: I can't find where this is defined, but this appears to
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
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *OpenTest) TestNonExistent_CreateFlagNotSet() {
	var err error
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0700)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// No object should have been created.
	_, err = storageutil.ReadObject(ctx, bucket, "foo")

	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *OpenTest) TestNonExistent_CreateFlagSet() {
	var err error

	// Open the file.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_RDWR|os.O_CREATE,
		0700)

	assert.Nil(t.T(), err)

	// Write some contents.
	_, err = t.f1.Write([]byte("012"))
	assert.Nil(t.T(), err)

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(1, 0)
	assert.Nil(t.T(), err)

	buf := make([]byte, 2)
	_, err = io.ReadFull(t.f1, buf)

	assert.Nil(t.T(), err)
	ExpectEq("12", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// The object should now be present in the bucket.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Nil(t.T(), err)
	ExpectEq("012", string(contents))

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("012", string(fileContents))
}

func (t *OpenTest) TestExistingFile() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("012"))
	assert.Nil(t.T(), err)

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(2, 0)
	assert.Nil(t.T(), err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(t.f1, buf)

	assert.Nil(t.T(), err)
	ExpectEq("2obu", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("012oburritoenchilada", string(fileContents))
}

func (t *OpenTest) TestExistingFile_Truncate() {
	var err error

	// Create a file.
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte("blahblahblah"),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_RDWR|os.O_TRUNC,
		0)

	assert.Nil(t.T(), err)

	// The file should be empty.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(0, fi.Size())

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("012"))
	assert.Nil(t.T(), err)

	// Read the contents.
	_, err = t.f1.Seek(0, 0)
	assert.Nil(t.T(), err)

	contentsSlice, err := ioutil.ReadAll(t.f1)
	assert.Nil(t.T(), err)
	ExpectEq("012", string(contentsSlice))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("012", string(fileContents))
}

func (t *OpenTest) TestAlreadyOpenedFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create and open a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Write some data into it.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Open another handle for reading and writing.
	t.f2, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// The contents written through the first handle should be available to the
	// second handle..
	n, err = t.f2.Read(buf[:2])
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, n)
	ExpectEq("ta", string(buf[:n]))

	// Write some contents with the second handle, which should now be at offset
	// 2.
	n, err = t.f2.Write([]byte("nk"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, n)

	// Check the overall contents now.
	contents, err := ioutil.ReadFile(t.f2.Name())
	assert.Nil(t.T(), err)
	ExpectEq("tank", string(contents))
}

func (t *OpenTest) TestLegalNames() {
	var err error

	names := interestingLegalNames()
	sort.Strings(names)

	// We should be able to create each name.
	for _, n := range names {
		err = ioutil.WriteFile(path.Join(mntDir, n), []byte(n), 0400)
		AssertEq(nil, err, "Name: %q", n)
	}

	// A listing should contain them all.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)

	AssertEq(len(names), len(entries))
	for i, n := range names {
		ExpectEq(n, entries[i].Name(), "Name: %q", n)
		ExpectEq(len(n), entries[i].Size(), "Name: %q", n)
	}

	// We should be able to read them all.
	for _, n := range names {
		contents, err := ioutil.ReadFile(path.Join(mntDir, n))
		AssertEq(nil, err, "Name: %q", n)
		ExpectEq(n, string(contents), "Name: %q", n)
	}

	// And delete each.
	for _, n := range names {
		err = os.Remove(path.Join(mntDir, n))
		AssertEq(nil, err, "Name: %q", n)
	}
}

func (t *OpenTest) TestIllegalNames() {
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
		err = ioutil.WriteFile(path.Join(mntDir, tc.name), []byte{}, 0400)
		ExpectThat(err, Error(HasSubstr(tc.err)), "Name: %q", tc.name)
	}
}

////////////////////////////////////////////////////////////////////////
// Mknod
////////////////////////////////////////////////////////////////////////

type MknodTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *MknodTest) TestFile() {
	// mknod(2) only works for root on OS X.
	if runtime.GOOS == "darwin" {
		return
	}

	var err error
	p := path.Join(mntDir, "foo")

	// Create
	err = syscall.Mknod(p, syscall.S_IFREG|0600, 0)
	assert.Nil(t.T(), err)

	// Stat
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)

	ExpectEq(path.Base(p), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Read
	contents, err := ioutil.ReadFile(p)
	assert.Nil(t.T(), err)
	ExpectEq("", string(contents))
}

func (t *MknodTest) TestDirectory() {
	// mknod(2) only works for root on OS X.
	if runtime.GOOS == "darwin" {
		return
	}

	var err error
	p := path.Join(mntDir, "foo")

	// Quoth `man 2 mknod`: "Under Linux, this call cannot be used to create
	// directories."
	err = syscall.Mknod(p, syscall.S_IFDIR|0700, 0)
	assert.Equal(t.T(), syscall.EPERM, err)
}

func (t *MknodTest) TestAlreadyExists() {
	// mknod(2) only works for root on OS X.
	if runtime.GOOS == "darwin" {
		return
	}

	var err error
	p := path.Join(mntDir, "foo")

	// Create (first)
	err = ioutil.WriteFile(p, []byte("taco"), 0600)
	assert.Nil(t.T(), err)

	// Create (second)
	err = syscall.Mknod(p, syscall.S_IFREG|0600, 0)
	assert.Equal(t.T(), syscall.EEXIST, err)

	// Read
	contents, err := ioutil.ReadFile(p)
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))
}

func (t *MknodTest) TestNonExistentParent() {
	// mknod(2) only works for root on OS X.
	if runtime.GOOS == "darwin" {
		return
	}

	var err error
	p := path.Join(mntDir, "foo/bar")

	err = syscall.Mknod(p, syscall.S_IFREG|0600, 0)
	assert.Equal(t.T(), syscall.ENOENT, err)
}

////////////////////////////////////////////////////////////////////////
// Modes
////////////////////////////////////////////////////////////////////////

type ModesTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *ModesTest) TestReadOnlyMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDONLY, 0)
	assert.Nil(t.T(), err)

	// Read its contents.
	fileContents, err := ioutil.ReadAll(t.f1)
	assert.Nil(t.T(), err)
	ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := t.f1.Write([]byte("taco"))

	assert.Equal(t.T(), 0, n)
	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *ModesTest) TestWriteOnlyMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_WRONLY, 0)
	assert.Nil(t.T(), err)

	// Reading should fail.
	_, err = ioutil.ReadAll(t.f1)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("000"))
	assert.Nil(t.T(), err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	assert.Nil(t.T(), err)

	// Seek and write past the end of the file.
	_, err = t.f1.Seek(int64(len(contents)), 0)
	assert.Nil(t.T(), err)

	_, err = t.f1.Write([]byte("222"))
	assert.Nil(t.T(), err)

	// Check the size now.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *ModesTest) TestReadWriteMode() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("000"))
	assert.Nil(t.T(), err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	assert.Nil(t.T(), err)

	// Seek and write past the end of the file.
	_, err = t.f1.Seek(int64(len(contents)), 0)
	assert.Nil(t.T(), err)

	_, err = t.f1.Write([]byte("222"))
	assert.Nil(t.T(), err)

	// Check the size now.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(4, 0)
	assert.Nil(t.T(), err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(t.f1, buf)

	assert.Nil(t.T(), err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = t.f1.ReadAt(buf, 0)

	assert.Nil(t.T(), err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *ModesTest) TestFuzzyReadWriteMode() {
	var err error

	// Create a file.
	const contents = "baz\u1100\u1161"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	ExpectEq("baz\u1100\u1161", string(fileContents))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Write to the start of the file using File.Write.
	_, err = t.f1.Write([]byte("타코世界"))
	assert.Nil(t.T(), err)

	// Seek and write immediately after the previous string.
	_, err = t.f1.Seek(int64(len("타코世界")), 0)
	assert.Nil(t.T(), err)

	_, err = t.f1.Write([]byte("\u0041\u030a"))
	assert.Nil(t.T(), err)

	// Check the size now.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(len("타코世界")+len("\u0041\u030a"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = t.f1.Seek(0, 0)
	assert.Nil(t.T(), err)

	buf := make([]byte, 3)
	_, err = io.ReadFull(t.f1, buf)

	assert.Nil(t.T(), err)
	ExpectEq("타", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len("타코世界")+len("\u0041\u030a"))
	_, err = t.f1.ReadAt(buf, 0)

	assert.Nil(t.T(), err)
	ExpectEq("타코世界\u0041\u030a", string(buf))

	// Close the file.
	AssertEq(nil, t.f1.Close())
	t.f1 = nil

	// Read back its contents.
	fileContents, err = ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("타코世界\u0041\u030a", string(fileContents))
}

func (t *ModesTest) TestAppendMode_SeekAndWrite() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR|os.O_APPEND, 0)
	assert.Nil(t.T(), err)

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = t.f1.Seek(1, 0)
	assert.Nil(t.T(), err)

	_, err = t.f1.Write([]byte("222"))
	assert.Nil(t.T(), err)

	// The seek position should have been updated.
	off, err := getFileOffset(t.f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len(contents)+len("222"), off)

	// Check the size now.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := t.f1.ReadAt(buf, 0)

	assert.Equal(t.T(), io.EOF, err)
	ExpectEq(contents+"222", string(buf[:n]))

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq(contents+"222", string(fileContents))
}

func (t *ModesTest) TestAppendMode_WriteAt() {
	var err error

	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(mntDir, "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	t.f1, err = os.OpenFile(path.Join(mntDir, "foo"), os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Seek somewhere in the file.
	_, err = t.f1.Seek(1, 0)
	assert.Nil(t.T(), err)

	// Write to the middle of the file using File.WriteAt.
	_, err = t.f1.WriteAt([]byte("111"), 4)
	assert.Nil(t.T(), err)

	// The seek position should have been unaffected.
	off, err := getFileOffset(t.f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, off)

	// Check the size now.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(len(contents), fi.Size())

	// Read the full contents with ReadAt.
	buf := make([]byte, 1024)
	n, err := t.f1.ReadAt(buf, 0)

	assert.Equal(t.T(), io.EOF, err)
	ExpectEq("taco111ritoenchilada", string(buf[:n]))

	// Read the full contents with another file handle.
	fileContents, err := ioutil.ReadFile(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("taco111ritoenchilada", string(fileContents))
}

func (t *ModesTest) TestAppendMode_WriteAt_PastEOF() {
	var err error

	// Open a file.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_RDWR|os.O_CREATE,
		0600)

	assert.Nil(t.T(), err)

	// Write three bytes.
	n, err := t.f1.Write([]byte("111"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, n)

	// Write at offset six.
	n, err = t.f1.WriteAt([]byte("222"), 6)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, n)

	// The seek position should have been unaffected.
	off, err := getFileOffset(t.f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, off)

	// Read the full contents of the file.
	contents, err := ioutil.ReadFile(t.f1.Name())
	assert.Nil(t.T(), err)

	ExpectEq("111\x00\x00\x00222", string(contents))
}

func (t *ModesTest) TestReadFromWriteOnlyFile() {
	var err error

	// Create and open a file for writing.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_WRONLY|os.O_CREATE,
		0700)

	assert.Nil(t.T(), err)

	// Attempt to read from it.
	_, err = t.f1.Read(make([]byte, 1024))

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *ModesTest) TestWriteToReadOnlyFile() {
	var err error

	// Create and open a file for reading.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_RDONLY|os.O_CREATE,
		0700)

	assert.Nil(t.T(), err)

	// Attempt to write t it.
	_, err = t.f1.Write([]byte("taco"))

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

////////////////////////////////////////////////////////////////////////
// Directory interaction
////////////////////////////////////////////////////////////////////////

type DirectoryTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *DirectoryTest) TestMkdir_OneLevel() {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	dirName := path.Join(mntDir, "dir")

	// Create a directory within the root.
	err = os.Mkdir(dirName, 0754)
	assert.Nil(t.T(), err)

	// Stat the directory.
	fi, err = os.Stat(dirName)

	assert.Nil(t.T(), err)
	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = fusetesting.ReadDirPicky(dirName)

	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())

	// Read the root.
	entries, err = fusetesting.ReadDirPicky(mntDir)

	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("dir", fi.Name())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
}

func (t *DirectoryTest) TestMkdir_TwoLevels() {
	var err error
	var fi os.FileInfo
	var entries []os.FileInfo

	// Create a directory within the root.
	err = os.Mkdir(path.Join(mntDir, "parent"), 0700)
	assert.Nil(t.T(), err)

	// Create a child of that directory.
	err = os.Mkdir(path.Join(mntDir, "parent/dir"), 0754)
	assert.Nil(t.T(), err)

	// Stat the directory.
	fi, err = os.Stat(path.Join(mntDir, "parent/dir"))

	assert.Nil(t.T(), err)
	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)

	// Read the directory.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "parent/dir"))

	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())

	// Read the parent.
	entries, err = fusetesting.ReadDirPicky(path.Join(mntDir, "parent"))

	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("dir", fi.Name())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
}

func (t *DirectoryTest) TestMkdir_AlreadyExists() {
	var err error
	dirName := path.Join(mntDir, "dir")

	// Create the directory once.
	err = os.Mkdir(dirName, 0754)
	assert.Nil(t.T(), err)

	// Attempt to create it again.
	err = os.Mkdir(dirName, 0754)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("exists")))
}

func (t *DirectoryTest) TestMkdir_IntermediateIsFile() {
	var err error

	// Create a file.
	fileName := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(fileName, []byte{}, 0700)
	assert.Nil(t.T(), err)

	// Attempt to create a directory within the file.
	dirName := path.Join(fileName, "dir")
	err = os.Mkdir(dirName, 0754)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("not a directory")))
}

func (t *DirectoryTest) TestMkdir_IntermediateIsNonExistent() {
	var err error

	// Attempt to create a sub-directory of a non-existent sub-directory.
	dirName := path.Join(mntDir, "foo/dir")
	err = os.Mkdir(dirName, 0754)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("no such file or directory")))
}

func (t *DirectoryTest) TestStat_Root() {
	fi, err := os.Stat(mntDir)
	assert.Nil(t.T(), err)

	ExpectEq(path.Base(mntDir), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *DirectoryTest) TestStat_FirstLevelDirectory() {
	var err error

	// Create a sub-directory.
	err = os.Mkdir(path.Join(mntDir, "dir"), 0700)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := os.Stat(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *DirectoryTest) TestStat_SecondLevelDirectory() {
	var err error

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(mntDir, "parent/dir"), 0700)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := os.Stat(path.Join(mntDir, "parent/dir"))
	assert.Nil(t.T(), err)

	ExpectEq("dir", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *DirectoryTest) TestReadDir_Root() {
	var err error
	var fi os.FileInfo

	// Create a file and a directory.
	createTime := mtimeClock.Now()
	err = ioutil.WriteFile(path.Join(mntDir, "bar"), []byte("taco"), 0700)
	assert.Nil(t.T(), err)

	err = os.Mkdir(path.Join(mntDir, "foo"), 0700)
	assert.Nil(t.T(), err)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *DirectoryTest) TestReadDir_SubDirectory() {
	var err error
	var fi os.FileInfo

	// Create a directory.
	parent := path.Join(mntDir, "parent")
	err = os.Mkdir(parent, 0700)
	assert.Nil(t.T(), err)

	// Create a file and a directory within it.
	createTime := mtimeClock.Now()
	err = ioutil.WriteFile(path.Join(parent, "bar"), []byte("taco"), 0700)
	assert.Nil(t.T(), err)

	err = os.Mkdir(path.Join(parent, "foo"), 0700)
	assert.Nil(t.T(), err)

	// ReadDir
	entries, err := fusetesting.ReadDirPicky(parent)
	assert.Nil(t.T(), err)
	AssertEq(2, len(entries))

	// bar
	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)

	// foo
	fi = entries[1]
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())
	ExpectTrue(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *DirectoryTest) TestRmdir_NotEmpty() {
	var err error

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(mntDir, "foo/bar"), 0754)
	assert.Nil(t.T(), err)

	// Attempt to remove the parent.
	err = os.Remove(path.Join(mntDir, "foo"))

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("not empty")))

	// The parent should still be there.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))

	assert.Nil(t.T(), err)
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *DirectoryTest) TestRmdir_Empty() {
	var err error
	var entries []os.FileInfo

	// Create two levels of directories.
	err = os.MkdirAll(path.Join(mntDir, "foo/bar"), 0754)
	assert.Nil(t.T(), err)

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

func (t *DirectoryTest) TestRmdir_OpenedForReading() {
	var err error

	// Create a directory.
	err = os.Mkdir(path.Join(mntDir, "dir"), 0700)
	assert.Nil(t.T(), err)

	// Open the directory for reading.
	t.f1, err = os.Open(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Remove the directory.
	err = os.Remove(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Create a new directory, with the same name even, and add some contents
	// within it.
	err = os.MkdirAll(path.Join(mntDir, "dir/foo"), 0700)
	assert.Nil(t.T(), err)

	err = os.MkdirAll(path.Join(mntDir, "dir/bar"), 0700)
	assert.Nil(t.T(), err)

	err = os.MkdirAll(path.Join(mntDir, "dir/baz"), 0700)
	assert.Nil(t.T(), err)

	// We should still be able to stat the open file handle.
	fi, err := t.f1.Stat()
	ExpectEq("dir", fi.Name())

	// Attempt to read from the directory. Unfortunately we can't implement the
	// guarantee that no new entries are returned, but nothing crazy should
	// happen.
	_, err = t.f1.Readdirnames(0)
	if err != nil {
		ExpectThat(err, Error(HasSubstr("no such file")))
	}
}

func (t *DirectoryTest) TestRmdir_ThenRecreateWithSameName() {
	var err error

	// Create a directory.
	err = os.Mkdir(path.Join(mntDir, "dir"), 0700)
	assert.Nil(t.T(), err)

	// Unlink the directory.
	err = os.Remove(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	// Re-create the directory with the same name. Nothing crazy should happen.
	// In the past, this used to crash (cf.
	// https://github.com/GoogleCloudPlatform/gcsfuse/issues/8).
	err = os.Mkdir(path.Join(mntDir, "dir"), 0700)
	assert.Nil(t.T(), err)

	// Statting should reveal nothing surprising.
	fi, err := os.Stat(path.Join(mntDir, "dir"))
	assert.Nil(t.T(), err)

	ExpectEq("dir", fi.Name())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectTrue(fi.IsDir())
}

func (t *DirectoryTest) TestCreateHardLink() {
	var err error

	// Write a file.
	err = ioutil.WriteFile(path.Join(mntDir, "foo"), []byte(""), 0700)
	assert.Nil(t.T(), err)

	// Attempt to hard link it. We don't support doing so.
	err = os.Link(
		path.Join(mntDir, "foo"),
		path.Join(mntDir, "bar"))

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("not implemented")))
}

func (t *DirectoryTest) TestChmod() {
	var err error

	// Create a directory.
	p := path.Join(mntDir, "foo")
	err = os.Mkdir(p, 0700)
	assert.Nil(t.T(), err)

	// Attempt to chmod it. Chmod should succeed even though we don't do anything
	// useful. The OS X Finder otherwise complains to the user when copying in a
	// file.
	err = os.Chmod(p, 0777)
	assert.Nil(t.T(), err)
}

func (t *DirectoryTest) TestChtimes() {
	var err error

	// Create a directory.
	p := path.Join(mntDir, "foo")
	err = os.Mkdir(p, 0700)
	assert.Nil(t.T(), err)

	// Chtimes should succeed even though we don't do anything useful. The OS X
	// Finder otherwise complains to the user when copying in a directory.
	err = os.Chtimes(p, time.Now(), time.Now())
	assert.Nil(t.T(), err)
}

func (t *DirectoryTest) TestAtimeCtimeAndMtime() {
	var err error

	// Create a directory.
	p := path.Join(mntDir, "foo")
	createTime := mtimeClock.Now()
	err = os.Mkdir(p, 0700)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)

	// We require only that the times be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(createTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(createTime, delta))
	ExpectThat(mtime, timeutil.TimeNear(createTime, delta))
}

func (t *DirectoryTest) TestRootAtimeCtimeAndMtime() {
	var err error
	mountTime := mtimeClock.Now()

	// Stat the root directory.
	fi, err := os.Stat(mntDir)
	assert.Nil(t.T(), err)

	// We require only that the times be "reasonable".
	atime, ctime, mtime := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(mountTime, delta))
	ExpectThat(mtime, timeutil.TimeNear(mountTime, delta))
}

func (t *DirectoryTest) TestContentTypes() {
	testCases := []string{
		"foo/",
		"foo.jpg/",
		"foo.gif/",
	}

	for _, name := range testCases {
		p := path.Join(mntDir, name)

		// Create the directory.
		err := os.Mkdir(p, 0700)
		assert.Nil(t.T(), err)

		// There should be no content type set in GCS.
		_, e, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{
			Name:                           name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})
		assert.Nil(t.T(), err)
		assert.NotNil(t.T(), e)
		ExpectEq("", e.ContentType, "name: %q", name)
	}
}

////////////////////////////////////////////////////////////////////////
// File interaction
////////////////////////////////////////////////////////////////////////

type FileTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *FileTest) TestWriteOverlapsEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Make it 4 bytes long.
	err = t.f1.Truncate(4)
	assert.Nil(t.T(), err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	assert.Nil(t.T(), err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) TestWriteStartsAtEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Make it 2 bytes long.
	err = t.f1.Truncate(2)
	assert.Nil(t.T(), err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	assert.Nil(t.T(), err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) TestWriteStartsPastEndOfFile() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Write the range [2, 6).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Read the full contents of the file.
	contents, err := ioutil.ReadAll(t.f1)
	assert.Nil(t.T(), err)
	ExpectEq("\x00\x00taco", string(contents))
}

func (t *FileTest) TestWriteAtDoesntChangeOffset_NotAppendMode() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	assert.Nil(t.T(), err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	assert.Nil(t.T(), err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, offset)
}

func (t *FileTest) TestWriteAtDoesntChangeOffset_AppendMode() {
	var err error
	var n int

	// Create a file in append mode.
	t.f1, err = os.OpenFile(
		path.Join(mntDir, "foo"),
		os.O_RDWR|os.O_CREATE,
		0600)

	assert.Nil(t.T(), err)

	// Make it 16 bytes long.
	err = t.f1.Truncate(16)
	assert.Nil(t.T(), err)

	// Seek to offset 4.
	_, err = t.f1.Seek(4, 0)
	assert.Nil(t.T(), err)

	// Write the range [10, 14).
	n, err = t.f1.WriteAt([]byte("taco"), 2)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// We should still be at offset 4.
	offset, err := getFileOffset(t.f1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, offset)
}

func validateObjectAttributes(t *fsTest, extendedAttr1, extendedAttr2 *gcs.ExtendedObjectAttributes,
	minObject1, minObject2 *gcs.MinObject) {
	assert.NotNil(t.T(), extendedAttr1)
	assert.NotNil(t.T(), extendedAttr2)
	assert.NotNil(t.T(), minObject1)
	assert.NotNil(t.T(), minObject2)
	// Validate Min Object.
	assert.Equal(t.T(), minObject1.Name, minObject2.Name)
	assert.Equal(t.T(), 0, minObject1.Size)
	assert.Equal(t.T(), FileContentsSize, minObject2.Size)
	assert.NotEqual(t.T(), minObject1.Generation, minObject2.Generation)
	ExpectTrue(minObject1.Updated.Before(minObject2.Updated))
	attr1MTime, _ := time.Parse(time.RFC3339Nano, minObject1.Metadata[gcsx.MtimeMetadataKey])
	attr2MTime, _ := time.Parse(time.RFC3339Nano, minObject2.Metadata[gcsx.MtimeMetadataKey])
	ExpectTrue(attr1MTime.Before(attr2MTime))
	assert.Equal(t.T(), minObject1.ContentEncoding, minObject2.ContentEncoding)
	assert.NotNil(t.T(), minObject1.CRC32C)
	assert.NotNil(t.T(), minObject2.CRC32C)

	// Validate Extended Object Attributes.
	assert.Equal(t.T(), extendedAttr1.ContentType, extendedAttr2.ContentType)
	assert.Equal(t.T(), extendedAttr1.ContentLanguage, extendedAttr2.ContentLanguage)
	assert.Equal(t.T(), extendedAttr1.CacheControl, extendedAttr2.CacheControl)
	assert.Equal(t.T(), extendedAttr1.Owner, extendedAttr2.Owner)
	assert.Equal(t.T(), extendedAttr1.MediaLink, extendedAttr2.MediaLink)
	assert.Equal(t.T(), extendedAttr1.StorageClass, extendedAttr2.StorageClass)
	ExpectTrue(reflect.DeepEqual(extendedAttr1.Deleted, time.Time{}))
	ExpectTrue(reflect.DeepEqual(extendedAttr2.Deleted, time.Time{}))
	assert.Equal(t.T(), extendedAttr1.ComponentCount+1, extendedAttr2.ComponentCount)
	assert.Equal(t.T(), extendedAttr1.EventBasedHold, extendedAttr2.EventBasedHold)
	assert.Equal(t.T(), extendedAttr1.Acl, extendedAttr2.Acl)
	assert.Nil(t.T(), extendedAttr1.Acl)
}

func createFile(t *fsTest, filePath string) {
	f, err := os.Create(filePath)
	assert.Nil(t.T(), err)
	err = f.Close()
	assert.Nil(t.T(), err)
}

func (t *FileTest) TestAppendFileOperation_ShouldNotChangeObjectAttributes() {
	// Create file.
	fileName := "foo"
	createFile(&t.fsTest, path.Join(mntDir, fileName))
	// Fetch object attributes before file append.
	minObject1, extendedAttr1, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: true, ReturnExtendedObjectAttributes: true})
	assert.Nil(t.T(), err)
	time.Sleep(timeSlop + timeSlop/2)

	// Append to the file.
	err = operations.WriteFileInAppendMode(path.Join(mntDir, fileName), FileContents)
	assert.Nil(t.T(), err)

	// Fetch object attributes after file append.
	minObject2, extendedAttr2, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: true, ReturnExtendedObjectAttributes: true})
	assert.Nil(t.T(), err)
	// Validate object attributes are as expected.
	validateObjectAttributes(&t.fsTest, extendedAttr1, extendedAttr2, minObject1, minObject2)
}

func (t *FileTest) TestWriteAtFileOperation_ShouldNotChangeObjectAttributes() {
	// Create file.
	fileName := "foo"
	createFile(&t.fsTest, path.Join(mntDir, fileName))
	// Fetch object attributes before file append.
	minObject1, extendedAttr1, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: true, ReturnExtendedObjectAttributes: true})
	assert.Nil(t.T(), err)
	time.Sleep(timeSlop + timeSlop/2)

	// Over-write the file.
	fh, err := os.OpenFile(path.Join(mntDir, fileName), os.O_RDWR, operations.FilePermission_0600)
	assert.Nil(t.T(), err)
	_, err = fh.WriteAt([]byte(FileContents), 0)
	assert.Nil(t.T(), err)
	err = fh.Close()
	assert.Nil(t.T(), err)
	minObject2, extendedAttr2, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: true, ReturnExtendedObjectAttributes: true})
	assert.Nil(t.T(), err)

	// Validate object attributes are as expected.
	validateObjectAttributes(&t.fsTest, extendedAttr1, extendedAttr2, minObject1, minObject2)
}

func (t *FileTest) TestReadsPastEndOfFile() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Read a range overlapping EOF.
	n, err = t.f1.ReadAt(buf[:4], 2)
	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), 2, n)
	ExpectEq("co", string(buf[:n]))

	// Read a range starting at EOF.
	n, err = t.f1.ReadAt(buf[:4], 4)
	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), 0, n)
	ExpectEq("", string(buf[:n]))

	// Read a range starting past EOF.
	n, err = t.f1.ReadAt(buf[:4], 100)
	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), 0, n)
	ExpectEq("", string(buf[:n]))
}

func (t *FileTest) TestTruncate_Smaller() {
	var err error
	fileName := path.Join(mntDir, "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	assert.Nil(t.T(), err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Truncate it.
	err = t.f1.Truncate(2)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(2, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	assert.Nil(t.T(), err)
	ExpectEq("ta", string(contents))
}

func (t *FileTest) TestTruncate_SameSize() {
	var err error
	fileName := path.Join(mntDir, "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	assert.Nil(t.T(), err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Truncate it.
	err = t.f1.Truncate(4)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(4, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) TestTruncate_Larger() {
	var err error
	fileName := path.Join(mntDir, "foo")

	// Create a file.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	assert.Nil(t.T(), err)

	// Open it for modification.
	t.f1, err = os.OpenFile(fileName, os.O_RDWR, 0)
	assert.Nil(t.T(), err)

	// Truncate it.
	err = t.f1.Truncate(6)
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)
	ExpectEq(6, fi.Size())

	// Read the contents.
	contents, err := ioutil.ReadFile(fileName)
	assert.Nil(t.T(), err)
	ExpectEq("taco\x00\x00", string(contents))
}

func (t *FileTest) TestSeek() {
	var err error
	var n int
	buf := make([]byte, 1024)

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Seek and overwrite.
	off, err := t.f1.Seek(1, 0)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, off)

	n, err = t.f1.Write([]byte("xx"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, n)

	// Read full the contents of the file.
	n, err = t.f1.ReadAt(buf, 0)
	assert.Equal(t.T(), io.EOF, err)
	ExpectEq("txxo", string(buf[:n]))
}

func (t *FileTest) TestStat() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Give it some contents.
	time.Sleep(timeSlop + timeSlop/2)
	writeTime := mtimeClock.Now()

	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	time.Sleep(timeSlop + timeSlop/2)

	// Stat it.
	fi, err := t.f1.Stat()
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(writeTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) TestStatUnopenedFile() {
	var err error

	// Create and close a file.
	time.Sleep(timeSlop + timeSlop/2)
	createTime := mtimeClock.Now()

	err = ioutil.WriteFile(path.Join(mntDir, "foo"), []byte("taco"), 0700)
	assert.Nil(t.T(), err)

	time.Sleep(timeSlop + timeSlop/2)

	// Stat it.
	fi, err := os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) TestLstatUnopenedFile() {
	var err error

	// Create and close a file.
	time.Sleep(timeSlop + timeSlop/2)
	createTime := mtimeClock.Now()

	err = ioutil.WriteFile(path.Join(mntDir, "foo"), []byte("taco"), 0700)
	assert.Nil(t.T(), err)

	time.Sleep(timeSlop + timeSlop/2)

	// Lstat it.
	fi, err := os.Lstat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(filePerms, fi.Mode())
	ExpectThat(fi, fusetesting.MtimeIsWithin(createTime, timeSlop))
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
	ExpectEq(currentUid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(currentGid(&t.fsTest), fi.Sys().(*syscall.Stat_t).Gid)
}

func (t *FileTest) TestUnlinkFile_Exists() {
	var err error

	// Write a file.
	fileName := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	assert.Nil(t.T(), err)

	// Unlink it.
	err = os.Remove(fileName)
	assert.Nil(t.T(), err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())
}

func (t *FileTest) TestUnlinkFile_NonExistent() {
	err := os.Remove(path.Join(mntDir, "foo"))

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *FileTest) TestUnlinkFile_StillOpen() {
	var err error

	fileName := path.Join(mntDir, "foo")

	// Create and open a file.
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	assert.Nil(t.T(), err)
	defer f.Close()

	// Write some data into it.
	n, err := f.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Unlink it.
	err = os.Remove(fileName)
	assert.Nil(t.T(), err)

	// The directory should no longer contain it.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())

	// We should be able to stat the file. It should still show as having
	// contents, but with no links.
	fi, err := f.Stat()

	assert.Nil(t.T(), err)
	ExpectEq(4, fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// The contents should still be available.
	buf := make([]byte, 1024)
	n, err = f.ReadAt(buf, 0)

	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), 4, n)
	ExpectEq("taco", string(buf[:4]))

	// Writing should still work, too.
	n, err = f.Write([]byte("burrito"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len("burrito"), n)
}

func (t *FileTest) TestUnlinkFile_NoLongerInBucket() {
	var err error

	// Write a file.
	fileName := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	assert.Nil(t.T(), err)

	// Delete it from the bucket through the back door.
	AssertEq(
		nil,
		bucket.DeleteObject(
			ctx,
			&gcs.DeleteObjectRequest{Name: "foo"}))

	assert.Nil(t.T(), err)

	// Attempt to unlink it.
	err = os.Remove(fileName)

	assert.NotNil(t.T(), err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *FileTest) TestUnlinkFile_FromSubDirectory() {
	var err error

	// Create a sub-directory.
	dirName := path.Join(mntDir, "dir")
	err = os.Mkdir(dirName, 0700)
	assert.Nil(t.T(), err)

	// Write a file to that directory.
	fileName := path.Join(dirName, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	assert.Nil(t.T(), err)

	// Unlink it.
	err = os.Remove(fileName)
	assert.Nil(t.T(), err)

	// Statting it should fail.
	_, err = os.Stat(fileName)

	assert.NotNil(t.T(), err)
	ExpectThat(err, Error(HasSubstr("no such file")))

	// Nothing should be in the directory.
	entries, err := fusetesting.ReadDirPicky(dirName)
	assert.Nil(t.T(), err)
	ExpectThat(entries, ElementsAre())
}

func (t *FileTest) TestUnlinkFile_ThenRecreateWithSameName() {
	var err error

	// Write a file.
	fileName := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	assert.Nil(t.T(), err)

	// Unlink it.
	err = os.Remove(fileName)
	assert.Nil(t.T(), err)

	// Re-create a file with the same name.
	err = ioutil.WriteFile(fileName, []byte("taco"), 0600)
	assert.Nil(t.T(), err)

	// Statting should result in a record for the new contents.
	fi, err := os.Stat(fileName)
	assert.Nil(t.T(), err)

	ExpectEq("foo", fi.Name())
	ExpectEq(len("taco"), fi.Size())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *FileTest) TestChmod() {
	var err error

	// Write a file.
	p := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(p, []byte(""), 0700)
	assert.Nil(t.T(), err)

	// Attempt to chmod it. Chmod should succeed even though we don't do anything
	// useful. The OS X Finder otherwise complains to the user when copying in a
	// file.
	err = os.Chmod(p, 0777)
	assert.Nil(t.T(), err)
}

func (t *FileTest) TestChtimes_InactiveFile() {
	var err error

	// Create a file.
	p := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(p, []byte{}, 0600)
	assert.Nil(t.T(), err)

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	assert.Nil(t.T(), err)

	// Stat it and confirm that it worked.
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) TestChtimes_OpenFile_Clean() {
	var err error

	// Create a file.
	p := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(p, []byte{}, 0600)
	assert.Nil(t.T(), err)

	// Open it for reading.
	f, err := os.Open(p)
	assert.Nil(t.T(), err)
	defer f.Close()

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	assert.Nil(t.T(), err)

	// Stat it by path.
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Stat it by fd.
	fi, err = f.Stat()
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Close the file, then stat it by path again.
	err = f.Close()
	assert.Nil(t.T(), err)

	fi, err = os.Stat(p)
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) TestChtimes_OpenFile_Dirty() {
	var err error

	// Create a file.
	p := path.Join(mntDir, "foo")
	f, err := os.Create(p)
	assert.Nil(t.T(), err)
	defer f.Close()

	// Dirty the file.
	_, err = f.Write([]byte("taco"))
	assert.Nil(t.T(), err)

	// Change its mtime.
	newMtime := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(p, time.Now(), newMtime)
	assert.Nil(t.T(), err)

	// Stat it by path.
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Stat it by fd.
	fi, err = f.Stat()
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))

	// Close the file, then stat it by path again.
	err = f.Close()
	assert.Nil(t.T(), err)

	fi, err = os.Stat(p)
	assert.Nil(t.T(), err)
	ExpectThat(fi.ModTime(), timeutil.TimeEq(newMtime))
}

func (t *FileTest) TestSync_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Sync it.
	err = t.f1.Sync()
	assert.Nil(t.T(), err)

	// The contents should now be in the bucket, even though we haven't closed
	// the file.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) TestSync_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	// Sync the file.
	err = t.f1.Sync()
	assert.Nil(t.T(), err)

	// The above should have created a generation for the object. Grab a record
	// for it.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}
	m1, _, err := bucket.StatObject(ctx, statReq)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), m1)

	// Sync the file again.
	err = t.f1.Sync()
	assert.Nil(t.T(), err)

	// A new generation need not have been written.
	m2, _, err := bucket.StatObject(ctx, statReq)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), m2)
	assert.Equal(t.T(), m1.Generation, m2.Generation)
}

func (t *FileTest) TestSync_Clobbered() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Dirty the file by giving it some contents.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))

	assert.Nil(t.T(), err)

	// Attempt to sync the file. This may result in an error if the OS has
	// decided to hold back the writes from above until now (in which case the
	// inode will fail to load the source object), or it may fail silently.
	// Either way, this should not result in a new generation being created.
	err = t.f1.Sync()
	if err != nil {
		ExpectThat(err, Error(HasSubstr("input/output error")))
	}

	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Nil(t.T(), err)
	ExpectEq("foobar", string(contents))
}

func (t *FileTest) TestClose_Dirty() {
	var err error
	var n int

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Give it some contents.
	n, err = t.f1.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Close it.
	err = t.f1.Close()
	t.f1 = nil
	assert.Nil(t.T(), err)

	// The contents should now be in the bucket.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))
}

func (t *FileTest) TestClose_NotDirty() {
	var err error

	// Create a file.
	t.f1, err = os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)

	// Close the file.
	err = t.f1.Close()
	t.f1 = nil
	assert.Nil(t.T(), err)

	// Verify if the object is created in GCS.
	statReq := &gcs.StatObjectRequest{
		Name: "foo",
	}
	_, _, err = bucket.StatObject(ctx, statReq)
	assert.Nil(t.T(), err)
}

func (t *FileTest) TestClose_Clobbered() {
	var err error
	var n int

	// Create a file.
	f, err := os.Create(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	defer f.Close()

	// Dirty the file by giving it some contents.
	n, err = f.Write([]byte("taco"))
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 4, n)

	// Replace the underlying object with a new generation.
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		"foo",
		[]byte("foobar"))

	assert.Nil(t.T(), err)

	// Close the file. This may result in a "generation not found" error when
	// faulting in the object's contents on Linux where close may cause cached
	// writes to be delivered to the file system. But in any case the new
	// generation should not be replaced.
	f.Close()

	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.Nil(t.T(), err)
	ExpectEq("foobar", string(contents))
}

func (t *FileTest) TestAtimeAndCtime() {
	var err error

	// Create a file.
	p := path.Join(mntDir, "foo")
	createTime := mtimeClock.Now()
	f, err := os.Create(p)
	assert.Nil(t.T(), err)
	_, err = f.Write([]byte("test contents"))
	assert.Nil(t.T(), err)
	err = f.Close()
	assert.Nil(t.T(), err)

	// Stat it.
	fi, err := os.Stat(p)
	assert.Nil(t.T(), err)

	// We require only that atime and ctime be "reasonable".
	atime, ctime, _ := fusetesting.GetTimes(fi)
	const delta = 5 * time.Hour

	ExpectThat(atime, timeutil.TimeNear(createTime, delta))
	ExpectThat(ctime, timeutil.TimeNear(createTime, delta))
}

func (t *FileTest) TestContentTypes() {
	testCases := map[string]string{
		"foo.jpg": "image/jpeg",
		"bar.gif": "image/gif",
		"baz":     "",
	}

	runOne := func(name string, expected string) {
		p := path.Join(mntDir, name)

		// Create a file.
		f, err := os.Create(p)
		assert.Nil(t.T(), err)
		err = f.Close()
		assert.Nil(t.T(), err)

		// Modify the file and cause a new generation to be written out.
		f1, err := os.OpenFile(p, os.O_WRONLY, 0)
		assert.Nil(t.T(), err)
		defer func() {
			err := f1.Close()
			assert.Nil(t.T(), err)
		}()
		_, err = f1.Write([]byte("taco"))
		assert.Nil(t.T(), err)

		err = f1.Sync()
		assert.Nil(t.T(), err)

		// The GCS content type should still be correct.
		_, e, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{
			Name:                           name,
			ForceFetchFromGcs:              true,
			ReturnExtendedObjectAttributes: true})
		assert.Nil(t.T(), err)
		assert.NotNil(t.T(), e)
		ExpectEq(expected, e.ContentType, "name: %q", name)
	}

	for name, expected := range testCases {
		runOne(name, expected)
	}
}

////////////////////////////////////////////////////////////////////////
// Symlinks
////////////////////////////////////////////////////////////////////////

type SymlinkTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *SymlinkTest) TestCreateLink() {
	var fi os.FileInfo
	var err error

	// Create a file.
	fileName := path.Join(mntDir, "foo")
	const contents = "taco"

	err = ioutil.WriteFile(fileName, []byte(contents), 0400)
	assert.Nil(t.T(), err)

	// Create a symlink to it.
	symlinkName := path.Join(mntDir, "bar")
	err = os.Symlink("foo", symlinkName)
	assert.Nil(t.T(), err)

	// Check the object in the bucket.
	m, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: "bar"})

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(), 0, m.Size)
	assert.Equal(t.T(), "foo", m.Metadata["gcsfuse_symlink_target"])

	// Read the link.
	target, err := os.Readlink(symlinkName)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "foo", target)

	// Stat the link.
	fi, err = os.Lstat(symlinkName)
	assert.Nil(t.T(), err)

	ExpectEq("bar", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Read the parent directory.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(2, len(entries))

	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Stat the target via the link.
	fi, err = os.Stat(symlinkName)
	assert.Nil(t.T(), err)

	ExpectEq("bar", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(filePerms, fi.Mode())
}

func (t *SymlinkTest) TestCreateLink_Exists() {
	var err error

	// Create a file and a directory.
	fileName := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(fileName, []byte{}, 0400)
	assert.Nil(t.T(), err)

	dirName := path.Join(mntDir, "bar")
	err = os.Mkdir(dirName, 0700)
	assert.Nil(t.T(), err)

	// Create an existing symlink.
	symlinkName := path.Join(mntDir, "baz")
	err = os.Symlink("blah", symlinkName)
	assert.Nil(t.T(), err)

	// Symlinking on top of any of them should fail.
	names := []string{
		fileName,
		dirName,
		symlinkName,
	}

	for _, n := range names {
		err = os.Symlink("blah", n)
		ExpectThat(err, Error(HasSubstr("exists")))
	}
}

func (t *SymlinkTest) TestRemoveLink() {
	var err error

	// Create the link.
	symlinkName := path.Join(mntDir, "foo")
	err = os.Symlink("blah", symlinkName)
	assert.Nil(t.T(), err)

	// Remove it.
	err = os.Remove(symlinkName)
	assert.Nil(t.T(), err)

	// It should be gone from the bucket.
	_, err = storageutil.ReadObject(ctx, bucket, "foo")
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

////////////////////////////////////////////////////////////////////////
// Rename
////////////////////////////////////////////////////////////////////////

type RenameTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *RenameTest) TestDirectoryNamingConflicts() {
	var err error

	oldPath := path.Join(mntDir, "foo")
	err = os.Mkdir(oldPath, 0700)
	assert.Nil(t.T(), err)

	conflictingPath := path.Join(mntDir, "bar")
	err = os.Mkdir(conflictingPath, 0700)
	assert.Nil(t.T(), err)

	conflictingFile := path.Join(conflictingPath, "placeholder.txt")
	err = ioutil.WriteFile(conflictingFile, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	err = syscall.Rename(oldPath, conflictingPath)
	ExpectThat(err, Error(HasSubstr("directory not empty")))

	err = os.Remove(conflictingFile)
	assert.Nil(t.T(), err)

	err = syscall.Rename(oldPath, conflictingPath)
	assert.Nil(t.T(), err)
}

func (t *RenameTest) TestDirectoryContainingFiles() {
	var err error

	// Create a directory.
	oldPath := path.Join(mntDir, "foo")
	err = os.Mkdir(oldPath, 0700)
	assert.Nil(t.T(), err)

	for i := 0; i < int(RenameDirLimit); i++ {
		file := fmt.Sprintf("%s/%d.txt", oldPath, i)
		err = ioutil.WriteFile(file, []byte("taco"), 0400)
		assert.Nil(t.T(), err)
	}

	// Attempt to rename it.
	newPath := path.Join(mntDir, "bar")
	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)

	// File count exceeds the limit.
	file := fmt.Sprintf("%s/%d.txt", newPath, RenameDirLimit)
	err = ioutil.WriteFile(file, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	// Attempt to rename it.
	err = os.Rename(newPath, oldPath)
	ExpectThat(err, Error(HasSubstr("too many open files")))
}

func (t *RenameTest) TestDirectoryContainingDirectories() {
	var err error

	// Create a directory.
	oldPath := path.Join(mntDir, "foo")
	err = os.Mkdir(oldPath, 0700)
	assert.Nil(t.T(), err)

	// Create a subdirectory.
	subPath := path.Join(oldPath, "baz")
	err = os.Mkdir(subPath, 0700)
	assert.Nil(t.T(), err)

	// Create a subsubdirectory.
	subSubPath := path.Join(subPath, "qux")
	err = os.Mkdir(subSubPath, 0700)
	assert.Nil(t.T(), err)

	// Create files.
	filePath1 := path.Join(subPath, "file1")
	err = ioutil.WriteFile(filePath1, []byte("taco"), 0400)
	assert.Nil(t.T(), err)
	filePath2 := path.Join(subSubPath, "file2")
	err = ioutil.WriteFile(filePath2, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	// Rename the directory.
	newPath := path.Join(mntDir, "bar")
	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)
	files, err := ioutil.ReadDir(newPath)
	assert.Nil(t.T(), err)
	AssertEq(1, len(files))
	ExpectEq("baz", files[0].Name())
	ExpectTrue(files[0].IsDir())
}

func (t *RenameTest) TestEmptyDirectory() {
	var err error

	// Create a directory.
	oldPath := path.Join(mntDir, "foo")
	err = os.Mkdir(oldPath, 0700)
	assert.Nil(t.T(), err)

	// Rename it.
	newPath := path.Join(mntDir, "bar")
	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)

	_, err = os.Stat(oldPath)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	file, err := os.Stat(newPath)
	assert.Nil(t.T(), err)
	ExpectTrue(file.IsDir())
}

func (t *RenameTest) TestWithinDir() {
	var err error

	// Create a parent directory.
	parentPath := path.Join(mntDir, "parent")

	err = os.Mkdir(parentPath, 0700)
	assert.Nil(t.T(), err)

	// And a file within it.
	oldPath := path.Join(parentPath, "foo")

	err = ioutil.WriteFile(oldPath, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	// Rename it.
	newPath := path.Join(parentPath, "bar")

	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)

	// The old name shouldn't work.
	_, err = os.Stat(oldPath)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	_, err = ioutil.ReadFile(oldPath)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// The new name should.
	fi, err := os.Stat(newPath)
	assert.Nil(t.T(), err)
	ExpectEq(len("taco"), fi.Size())

	contents, err := ioutil.ReadFile(newPath)
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))

	// There should only be the new entry in the directory.
	entries, err := fusetesting.ReadDirPicky(parentPath)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))
	fi = entries[0]

	ExpectEq(path.Base(newPath), fi.Name())
	ExpectEq(len("taco"), fi.Size())
}

func (t *RenameTest) TestAcrossDirs() {
	var err error

	// Create two parent directories.
	oldParentPath := path.Join(mntDir, "old")
	newParentPath := path.Join(mntDir, "new")

	err = os.Mkdir(oldParentPath, 0700)
	assert.Nil(t.T(), err)

	err = os.Mkdir(newParentPath, 0700)
	assert.Nil(t.T(), err)

	// And a file within the first.
	oldPath := path.Join(oldParentPath, "foo")

	err = ioutil.WriteFile(oldPath, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	// Rename it.
	newPath := path.Join(newParentPath, "bar")

	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)

	// The old name shouldn't work.
	_, err = os.Stat(oldPath)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	_, err = ioutil.ReadFile(oldPath)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// The new name should.
	fi, err := os.Stat(newPath)
	assert.Nil(t.T(), err)
	ExpectEq(len("taco"), fi.Size())

	contents, err := ioutil.ReadFile(newPath)
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))

	// Check the old parent.
	entries, err := fusetesting.ReadDirPicky(oldParentPath)
	assert.Nil(t.T(), err)
	AssertEq(0, len(entries))

	// And the new one.
	entries, err = fusetesting.ReadDirPicky(newParentPath)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))
	fi = entries[0]

	ExpectEq(path.Base(newPath), fi.Name())
	ExpectEq(len("taco"), fi.Size())
}

func (t *RenameTest) TestOutOfFileSystem() {
	var err error

	// Create a file.
	oldPath := path.Join(mntDir, "foo")

	err = ioutil.WriteFile(oldPath, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	// Attempt to move it out of the file system.
	tempDir, err := ioutil.TempDir("", "memfs_test")
	assert.Nil(t.T(), err)
	defer os.RemoveAll(tempDir)

	err = os.Rename(oldPath, path.Join(tempDir, "bar"))
	ExpectThat(err, Error(HasSubstr("cross-device")))
}

func (t *RenameTest) TestIntoFileSystem() {
	var err error

	// Create a file outside of our file system.
	f, err := ioutil.TempFile("", "memfs_test")
	assert.Nil(t.T(), err)
	defer f.Close()

	oldPath := f.Name()
	defer os.Remove(oldPath)

	// Attempt to move it into the file system.
	err = os.Rename(oldPath, path.Join(mntDir, "bar"))
	ExpectThat(err, Error(HasSubstr("cross-device")))
}

func (t *RenameTest) TestOverExistingFile() {
	var err error

	// Create two files.
	oldPath := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(oldPath, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	newPath := path.Join(mntDir, "bar")
	err = ioutil.WriteFile(newPath, []byte("burrito"), 0600)
	assert.Nil(t.T(), err)

	// Rename one over the other.
	err = os.Rename(oldPath, newPath)
	assert.Nil(t.T(), err)

	// Check the file contents.
	contents, err := ioutil.ReadFile(newPath)
	assert.Nil(t.T(), err)
	ExpectEq("taco", string(contents))

	// And the parent listing.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	assert.Nil(t.T(), err)
	AssertEq(1, len(entries))
	fi := entries[0]

	ExpectEq(path.Base(newPath), fi.Name())
	ExpectEq(len("taco"), fi.Size())
}

func (t *RenameTest) TestOverExisting_WrongType() {
	var err error

	// Create a file and a directory.
	filePath := path.Join(mntDir, "foo")
	err = ioutil.WriteFile(filePath, []byte("taco"), 0400)
	assert.Nil(t.T(), err)

	dirPath := path.Join(mntDir, "bar")
	err = os.Mkdir(dirPath, 0700)
	assert.Nil(t.T(), err)

	// Renaming one over the other shouldn't work.
	err = os.Rename(filePath, dirPath)
	ExpectThat(err, Error(MatchesRegexp("file exists|is a directory")))

	err = os.Rename(dirPath, filePath)
	ExpectThat(err, Error(HasSubstr("not a directory")))
}

func (t *RenameTest) TestNonExistentFile() {
	var err error

	err = os.Rename(path.Join(mntDir, "foo"), path.Join(mntDir, "bar"))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

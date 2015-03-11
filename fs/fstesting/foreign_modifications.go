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
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcsfuse/fs"
	"github.com/jacobsa/gcsfuse/timeutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getFileNames(entries []os.FileInfo) (names []string) {
	for _, e := range entries {
		names = append(names, e.Name())
	}

	return
}

// REQUIRES: n % 4 == 0
func randString(n int) string {
	bytes := make([]byte, n)
	for i := 0; i < n; i += 4 {
		u32 := rand.Uint32()
		bytes[i] = byte(u32 >> 0)
		bytes[i+1] = byte(u32 >> 8)
		bytes[i+2] = byte(u32 >> 16)
		bytes[i+3] = byte(u32 >> 24)
	}

	return string(bytes)
}

func readRange(r io.ReadSeeker, offset int64, n int) (s string, err error) {
	if _, err = r.Seek(offset, 0); err != nil {
		return
	}

	bytes := make([]byte, n)
	if _, err = io.ReadFull(r, bytes); err != nil {
		return
	}

	s = string(bytes)
	return
}

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type fsTest struct {
	ctx    context.Context
	clock  timeutil.Clock
	bucket gcs.Bucket
	mfs    *fuse.MountedFileSystem
}

var _ fsTestInterface = &fsTest{}

func (t *fsTest) setUpFsTest(deps FSTestDeps) {
	t.ctx = context.Background()
	t.clock = deps.Clock
	t.bucket = deps.Bucket

	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Mount a file system.
	fileSystem, err := fs.NewFileSystem(t.clock, t.bucket)
	if err != nil {
		panic("NewFileSystem: " + err.Error())
	}

	t.mfs, err = fuse.Mount(mountPoint, fileSystem)
	if err != nil {
		panic("Mount: " + err.Error())
	}

	if err := t.mfs.WaitForReady(t.ctx); err != nil {
		panic("MountedFileSystem.WaitForReady: " + err.Error())
	}
}

func (t *fsTest) tearDownFsTest() {
	// Unmount the file system. Try again on "resource busy" errors.
	delay := 10 * time.Millisecond
	for {
		err := t.mfs.Unmount()
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		panic("MountedFileSystem.Unmount: " + err.Error())
	}

	if err := t.mfs.Join(t.ctx); err != nil {
		panic("MountedFileSystem.Join: " + err.Error())
	}
}

func (t *fsTest) createWithContents(name string, contents string) error {
	return t.createObjects(
		[]*gcsutil.ObjectInfo{
			&gcsutil.ObjectInfo{
				Attrs: storage.ObjectAttrs{
					Name: name,
				},
				Contents: contents,
			},
		})
}

func (t *fsTest) createObjects(objects []*gcsutil.ObjectInfo) error {
	_, err := gcsutil.CreateObjects(t.ctx, t.bucket, objects)
	return err
}

func (t *fsTest) createEmptyObjects(names []string) error {
	_, err := gcsutil.CreateEmptyObjects(t.ctx, t.bucket, names)
	return err
}

// Ensure that the clock will report a different time after returning.
func (t *fsTest) advanceTime() {
	// For simulated clocks, we can just advance the time.
	if c, ok := t.clock.(*timeutil.SimulatedClock); ok {
		c.AdvanceTime(time.Second)
		return
	}

	// Otherwise, sleep a moment.
	time.Sleep(time.Millisecond)
}

// Return a matcher that matches event times as reported by the bucket
// corresponding to the supplied start time as measured by the test.
func (t *fsTest) matchesStartTime(start time.Time) Matcher {
	// For simulated clocks we can use exact equality.
	if _, ok := t.clock.(*timeutil.SimulatedClock); ok {
		return timeutil.TimeEq(start)
	}

	// Otherwise, we need to take into account latency between the start of our
	// call and the time the server actually executed the operation.
	const slop = 60 * time.Second
	return timeutil.TimeNear(start, slop)
}

////////////////////////////////////////////////////////////////////////
// Read-only interaction
////////////////////////////////////////////////////////////////////////

type foreignModsTest struct {
	fsTest
}

// Repeatedly call ioutil.ReadDir until an error is encountered or until the
// result has the given length. After each successful call with the wrong
// length, advance the clock by more than the directory listing cache TTL in
// order to flush the cache before the next call.
//
// This is a hacky workaround for the lack of list-after-write consistency in
// GCS that must be used when interacting with GCS through a side channel
// rather than through the file system. We set up some objects through a back
// door, then list repeatedly until we see the state we hope to see.
func (t *foreignModsTest) readDirUntil(
	desiredLen int,
	dir string) (entries []os.FileInfo, err error) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	for i := 0; ; i++ {
		entries, err = ioutil.ReadDir(dir)
		if err != nil || len(entries) == desiredLen {
			return
		}

		// TODO(jacobsa): Remove this?
		// t.clock.AdvanceTime(2 * fs.ListingCacheTTL)

		// Should we stop?
		if time.Now().After(endTime) {
			err = errors.New("Timeout waiting for the given length.")
			break
		}

		// Sleep for awhile.
		const baseDelay = 10 * time.Millisecond
		time.Sleep(time.Duration(math.Pow(1.3, float64(i)) * float64(baseDelay)))

		// If this is taking awhile, log that fact so that the user can tell why
		// the test is hanging.
		if time.Since(startTime) > time.Second {
			var names []string
			for _, fi := range entries {
				names = append(names, fi.Name())
			}

			log.Printf(
				"readDirUntil waiting for length %v. Current: %v, names: %v",
				desiredLen,
				len(entries),
				names)
		}
	}

	return
}

func (t *foreignModsTest) EmptyRoot() {
	// ReadDir
	entries, err := t.readDirUntil(0, t.mfs.Dir())
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *foreignModsTest) ContentsInRoot() {
	// Set up contents.
	createTime := t.clock.Now()
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "foo",
					},
					Contents: "taco",
				},

				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "bar/",
					},
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "baz",
					},
					Contents: "burrito",
				},
			}))

	// Make sure the time below doesn't match.
	t.advanceTime()

	// ReadDir
	entries, err := t.readDirUntil(3, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
}

func (t *foreignModsTest) EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	AssertEq(nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	entries, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	entries, err = t.readDirUntil(0, path.Join(t.mfs.Dir(), "bar"))
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *foreignModsTest) UnreachableObjects() {
	// Set up objects that appear to be directory contents, but for which there
	// is no directory object.
	_, err := gcsutil.CreateEmptyObjects(
		t.ctx,
		t.bucket,
		[]string{
			"foo/0",
			"foo/1",
			"bar/0/",
		})

	AssertEq(nil, err)

	// Nothing should show up in the root.
	_, err = t.readDirUntil(0, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// Statting the directories shouldn't work.
	_, err = os.Stat(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *foreignModsTest) ContentsInSubDirectory() {
	// Set up contents.
	createTime := t.clock.Now()
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
				// Placeholder
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/",
					},
					Contents: "",
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/foo",
					},
					Contents: "taco",
				},

				// Directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/bar/",
					},
				},

				// File
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/baz",
					},
					Contents: "burrito",
				},
			}))

	// Make sure the time below doesn't match.
	t.advanceTime()

	// Wait for the directory to show up in the file system.
	_, err := t.readDirUntil(1, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// ReadDir
	entries, err := t.readDirUntil(3, path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectThat(e.ModTime(), t.matchesStartTime(createTime))
	ExpectFalse(e.IsDir())
}

func (t *foreignModsTest) ListDirectoryTwice_NoChange() {
	// Set up initial contents.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar",
		}))

	// List once.
	entries, err := t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	// List again.
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *foreignModsTest) ListDirectoryTwice_Changed_CacheStillValid() {
	// Set up initial contents.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar",
		}))

	// List once.
	entries, err := t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	// Add "baz" and remove "bar".
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "bar"))
	AssertEq(nil, t.createEmptyObjects([]string{"baz"}))

	// Advance the clock to just before the cache expiry.
	// t.clock.AdvanceTime(fs.ListingCacheTTL - time.Millisecond)
	AssertTrue(false, "TODO(jacobsa): Figure out what to do here.")

	// List again.
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *foreignModsTest) ListDirectoryTwice_Changed_CacheInvalidated() {
	// Set up initial contents.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar",
		}))

	// List once.
	entries, err := t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())

	// Add "baz" and remove "bar".
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "bar"))
	AssertEq(nil, t.createEmptyObjects([]string{"baz"}))

	// Advance the clock to just after the cache expiry.
	// t.clock.AdvanceTime(fs.ListingCacheTTL + time.Millisecond)
	AssertTrue(false, "TODO(jacobsa): Figure out what to do here.")

	// List again.
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("baz", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *foreignModsTest) Inodes() {
	// Set up two files and a directory placeholder.
	AssertEq(
		nil,
		t.createEmptyObjects([]string{
			"foo",
			"bar/",
			"baz",
		}))

	// List.
	entries, err := t.readDirUntil(3, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(3, len(entries), "Names: %v", getFileNames(entries))

	// Confirm all of the inodes are distinct.
	inodesSeen := make(map[uint64]struct{})
	for _, fileInfo := range entries {
		stat := fileInfo.Sys().(*syscall.Stat_t)
		_, ok := inodesSeen[stat.Ino]
		AssertFalse(
			ok,
			"Duplicate inode (%v). File info: %v",
			stat.Ino,
			fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (t *foreignModsTest) OpenNonExistentFile() {
	_, err := os.Open(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo")))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *foreignModsTest) ReadFromFile_Small() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("tacoburritoenchilada", string(slice))

	// Read various ranges of it.
	var s string

	s, err = readRange(f, int64(len("taco")), len("burrito"))
	AssertEq(nil, err)
	ExpectEq("burrito", s)

	s, err = readRange(f, 0, len("taco"))
	AssertEq(nil, err)
	ExpectEq("taco", s)

	s, err = readRange(f, int64(len("tacoburrito")), len("enchilada"))
	AssertEq(nil, err)
	ExpectEq("enchilada", s)
}

func (t *foreignModsTest) ReadFromFile_Large() {
	const contentLen = 1 << 20
	contents := randString(contentLen)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Read its entire contents.
	slice, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	if contents != string(slice) {
		ExpectTrue(
			false,
			"Expected:\n%v\n\nActual:\n%v",
			hex.Dump([]byte(contents)),
			hex.Dump(slice))
	}

	// Read from parts of it.
	referenceReader := strings.NewReader(contents)
	for trial := 0; trial < 1000; trial++ {
		offset := rand.Int63n(contentLen + 1)
		size := rand.Intn(int(contentLen - offset))

		expected, err := readRange(referenceReader, offset, size)
		AssertEq(nil, err)

		actual, err := readRange(f, offset, size)
		AssertEq(nil, err)

		if expected != actual {
			AssertTrue(
				expected == actual,
				"Expected:\n%s\nActual:\n%s",
				hex.Dump([]byte(expected)),
				hex.Dump([]byte(actual)))
		}
	}
}

func (t *foreignModsTest) ReadBeyondEndOfFile() {
	const contents = "tacoburritoenchilada"
	const contentLen = len(contents)

	// Create an object.
	AssertEq(nil, t.createWithContents("foo", contents))

	// Wait for it to show up in the file system.
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	// Attempt to open it.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() { AssertEq(nil, f.Close()) }()

	// Attempt to read beyond the end of the file.
	_, err = f.Seek(int64(contentLen-1), 0)
	AssertEq(nil, err)

	buf := make([]byte, 2)
	n, err := f.Read(buf)
	AssertEq(1, n, "err: %v", err)
	AssertEq(contents[contentLen-1], buf[0])

	if err == nil {
		n, err = f.Read(buf)
		AssertEq(0, n)
	}
}

func (t *foreignModsTest) ObjectIsOverwritten() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Overwrite the object.
	AssertEq(nil, t.createWithContents("foo", "burrito"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should yield the new version.
	newF, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, newF.Close())
	}()

	fi, err = newF.Stat()
	AssertEq(nil, err)
	ExpectEq(len("burrito"), fi.Size())
	ExpectEq(1, fi.Sys().(*syscall.Stat_t).Nlink)
}

func (t *foreignModsTest) ObjectIsDeleted() {
	// Create an object.
	AssertEq(nil, t.createWithContents("foo", "taco"))

	// Open the corresponding file for reading.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)
	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Delete the object.
	AssertEq(nil, t.bucket.DeleteObject(t.ctx, "foo"))

	// The file should appear to be unlinked, but with the previous contents.
	fi, err := f.Stat()

	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())
	ExpectEq(0, fi.Sys().(*syscall.Stat_t).Nlink)

	// Opening again should not work.
	_, err = os.Open(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

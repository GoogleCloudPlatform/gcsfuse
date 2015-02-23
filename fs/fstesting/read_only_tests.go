// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// A collection of tests for a file system where we do not attempt to write to
// the file system at all. Rather we set up contents in a GCS bucket out of
// band, wait for them to be available, and then read them via the file system.
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"encoding/hex"
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

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcsfuse/fs"
	"github.com/jacobsa/gcsfuse/fuseutil"
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
	clock  timeutil.SimulatedClock
	bucket gcs.Bucket
	mfs    *fuseutil.MountedFileSystem
}

var _ fsTestInterface = &fsTest{}

func (t *fsTest) setUpFsTest(b gcs.Bucket) {
	t.ctx = context.Background()
	t.bucket = b

	// Set up a temporary directory for mounting.
	mountPoint, err := ioutil.TempDir("", "fs_test")
	if err != nil {
		panic("ioutil.TempDir: " + err.Error())
	}

	// Mount a file system.
	fileSystem, err := fs.NewFuseFS(&t.clock, b)
	if err != nil {
		panic("NewFuseFS: " + err.Error())
	}

	t.mfs = fuseutil.MountFileSystem(mountPoint, fileSystem)
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

////////////////////////////////////////////////////////////////////////
// Read-only interaction
////////////////////////////////////////////////////////////////////////

type readOnlyTest struct {
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
func (t *readOnlyTest) readDirUntil(
	desiredLen int,
	dir string) (entries []os.FileInfo, err error) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	for i := 0; ; i++ {
		entries, err = ioutil.ReadDir(dir)
		if err != nil || len(entries) == desiredLen {
			return
		}

		t.clock.AdvanceTime(2 * fs.ListingCacheTTL)

		// Should we stop?
		if time.Now().After(endTime) {
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

func (t *readOnlyTest) EmptyRoot() {
	// ReadDir
	entries, err := t.readDirUntil(0, t.mfs.Dir())
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *readOnlyTest) ContentsInRoot() {
	// Set up contents.
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

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "qux/asdf",
					},
					Contents: "",
				},
			}))

	// ReadDir
	entries, err := t.readDirUntil(4, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(4, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) EmptySubDirectory() {
	// Set up an empty directory placeholder called 'bar'.
	AssertEq(nil, t.createEmptyObjects([]string{"bar/"}))

	// ReadDir
	_, err := t.readDirUntil(1, t.mfs.Dir())
	AssertEq(nil, err)

	entries, err := t.readDirUntil(0, path.Join(t.mfs.Dir(), "bar"))
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *readOnlyTest) ContentsInSubDirectory_PlaceholderPresent() {
	// Set up contents.
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

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/qux/asdf",
					},
					Contents: "",
				},
			}))

	// Wait for the directory to show up in the file system.
	_, err := t.readDirUntil(1, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// ReadDir
	entries, err := t.readDirUntil(4, path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(4, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) ContentsInSubDirectory_PlaceholderNotPresent() {
	// Set up contents.
	AssertEq(
		nil,
		t.createObjects(
			[]*gcsutil.ObjectInfo{
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

				// File in sub-directory
				&gcsutil.ObjectInfo{
					Attrs: storage.ObjectAttrs{
						Name: "dir/qux/asdf",
					},
					Contents: "",
				},
			}))

	// Wait for the directory to show up in the file system.
	_, err := t.readDirUntil(1, path.Join(t.mfs.Dir()))
	AssertEq(nil, err)

	// ReadDir
	entries, err := t.readDirUntil(4, path.Join(t.mfs.Dir(), "dir"))
	AssertEq(nil, err)

	AssertEq(4, len(entries), "Names: %v", getFileNames(entries))
	var e os.FileInfo

	// bar
	e = entries[0]
	ExpectEq("bar", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())

	// baz
	e = entries[1]
	ExpectEq("baz", e.Name())
	ExpectEq(len("burrito"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// foo
	e = entries[2]
	ExpectEq("foo", e.Name())
	ExpectEq(len("taco"), e.Size())
	ExpectEq(os.FileMode(0), e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectFalse(e.IsDir())

	// qux
	e = entries[3]
	ExpectEq("qux", e.Name())
	ExpectEq(0, e.Size())
	ExpectEq(os.ModeDir, e.Mode() & ^os.ModePerm)
	ExpectLt(
		math.Abs(time.Since(e.ModTime()).Seconds()), 30,
		"ModTime: %v", e.ModTime())
	ExpectTrue(e.IsDir())
}

func (t *readOnlyTest) ListDirectoryTwice_NoChange() {
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

func (t *readOnlyTest) ListDirectoryTwice_Changed_CacheStillValid() {
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
	t.clock.AdvanceTime(fs.ListingCacheTTL - time.Millisecond)

	// List again.
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("bar", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *readOnlyTest) ListDirectoryTwice_Changed_CacheInvalidated() {
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
	t.clock.AdvanceTime(fs.ListingCacheTTL + time.Millisecond)

	// List again.
	entries, err = t.readDirUntil(2, t.mfs.Dir())
	AssertEq(nil, err)

	AssertEq(2, len(entries), "Names: %v", getFileNames(entries))
	ExpectEq("baz", entries[0].Name())
	ExpectEq("foo", entries[1].Name())
}

func (t *readOnlyTest) Inodes() {
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
		AssertFalse(ok, "Duplicate inode: %v", fileInfo)

		inodesSeen[stat.Ino] = struct{}{}
	}
}

func (t *readOnlyTest) OpenNonExistentFile() {
	_, err := os.Open(path.Join(t.mfs.Dir(), "foo"))

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("foo")))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *readOnlyTest) ReadFromFile_Small() {
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

func (t *readOnlyTest) ReadFromFile_Large() {
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
	ExpectEq(contents, string(slice))

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

func (t *readOnlyTest) ReadBeyondEndOfFile() {
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

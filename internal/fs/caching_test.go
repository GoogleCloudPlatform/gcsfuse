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

package fs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	bucket2 "github.com/googlecloudplatform/gcsfuse/internal/storage/bucket"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/requests"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

const ttl = 10 * time.Minute

var (
	uncachedBucket bucket2.Bucket
)

type cachingTestCommon struct {
	fsTest
}

func (t *cachingTestCommon) SetUpTestSuite() {
	// Wrap the bucket in a stat caching layer for the purposes of the file
	// system.
	uncachedBucket = fake.NewFakeBucket(timeutil.RealClock(), "some_bucket")

	const statCacheCapacity = 1000
	statCache := caching.NewStatCache(statCacheCapacity)
	bucket = caching.NewFastStatBucket(
		ttl,
		statCache,
		&cacheClock,
		uncachedBucket)

	// Enable directory type caching.
	t.serverCfg.DirTypeCacheTTL = ttl

	// Call through.
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Caching
////////////////////////////////////////////////////////////////////////

type CachingTest struct {
	cachingTestCommon
}

func init() {
	RegisterTestSuite(&CachingTest{})
}

func (t *CachingTest) EmptyBucket() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *CachingTest) FileCreatedRemotely() {
	const name = "foo"
	const contents = "taco"

	var fi os.FileInfo

	// Create an object in GCS.
	_, err := storageutil.CreateObject(
		ctx,
		uncachedBucket,
		name,
		[]byte(contents))

	AssertEq(nil, err)

	// It should immediately show up in a listing.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq(name, fi.Name())
	ExpectEq(len(contents), fi.Size())

	// And we should be able to stat it.
	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)

	ExpectEq(name, fi.Name())
	ExpectEq(len(contents), fi.Size())

	// And read it.
	b, err := ioutil.ReadFile(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))

	// And overwrite it, and read it back again.
	err = ioutil.WriteFile(path.Join(mntDir, name), []byte("burrito"), 0500)
	AssertEq(nil, err)

	b, err = ioutil.ReadFile(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectEq("burrito", string(b))
}

func (t *CachingTest) FileChangedRemotely() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a file via the file system.
	err = ioutil.WriteFile(path.Join(mntDir, name), []byte("taco"), 0500)
	AssertEq(nil, err)

	// Overwrite the object in GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedBucket,
		name,
		[]byte("burrito"))

	AssertEq(nil, err)

	// Because we are caching, the file should still appear to be the local
	// version.
	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())

	// After the TTL elapses, we should see the new version.
	cacheClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectEq(len("burrito"), fi.Size())

	// Reading should work as expected.
	b, err := ioutil.ReadFile(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectEq("burrito", string(b))
}

func (t *CachingTest) DirectoryRemovedRemotely() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(mntDir, name), 0700)
	AssertEq(nil, err)

	// Remove the backing object in GCS.
	err = uncachedBucket.DeleteObject(
		ctx,
		&requests.DeleteObjectRequest{Name: name + "/"})

	AssertEq(nil, err)

	// Because we are caching, the directory should still appear to exist.
	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	// After the TTL elapses, we should see it disappear.
	cacheClock.AdvanceTime(ttl + time.Millisecond)

	_, err = os.Stat(path.Join(mntDir, name))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *CachingTest) ConflictingNames_RemoteModifier() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(mntDir, name), 0700)
	AssertEq(nil, err)

	// Create a file with the same name via GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedBucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the file
	// when statting.
	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	_, err = os.Stat(path.Join(mntDir, name+inode.ConflictingFileNameSuffix))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see both.
	cacheClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	fi, err = os.Stat(path.Join(mntDir, name+inode.ConflictingFileNameSuffix))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
}

func (t *CachingTest) TypeOfNameChanges_LocalModifier() {
	const name = "test"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(mntDir, name), 0700)
	AssertEq(nil, err)

	// Delete it and recreate as a file.
	err = os.Remove(path.Join(mntDir, name))
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(mntDir, name), []byte("taco"), 0400)
	AssertEq(nil, err)

	// All caches should have been updated.
	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
	ExpectEq(len("taco"), fi.Size())
}

func (t *CachingTest) TypeOfNameChanges_RemoteModifier() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	fmt.Printf("Mkdir\n")
	err = os.Mkdir(path.Join(mntDir, name), 0700)
	AssertEq(nil, err)

	// Remove the backing object in GCS, updating the bucket cache (but not the
	// file system type cache)
	fmt.Printf("DeleteObject\n")
	err = bucket.DeleteObject(
		ctx,
		&requests.DeleteObjectRequest{Name: name + "/"})

	AssertEq(nil, err)

	// Create a file with the same name via GCS, again updating the bucket cache.
	fmt.Printf("CreateObject\n")
	_, err = storageutil.CreateObject(
		ctx,
		bucket,
		name,
		[]byte("taco"))

	AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the name.
	fmt.Printf("Stat\n")
	_, err = os.Stat(path.Join(mntDir, name))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see it turn into a file.
	cacheClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(mntDir, name))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
}

////////////////////////////////////////////////////////////////////////
// Caching with implicit directories
////////////////////////////////////////////////////////////////////////

type CachingWithImplicitDirsTest struct {
	cachingTestCommon
}

func init() {
	RegisterTestSuite(&CachingWithImplicitDirsTest{})
}

func (t *CachingWithImplicitDirsTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.cachingTestCommon.SetUpTestSuite()
}

func (t *CachingWithImplicitDirsTest) ImplicitDirectory_DefinedByFile() {
	var fi os.FileInfo
	var err error

	// Set up a file object implicitly defining a directory in GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedBucket,
		"foo/bar",
		[]byte(""))

	AssertEq(nil, err)

	// The directory should appear to exist.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *CachingWithImplicitDirsTest) ImplicitDirectory_DefinedByDirectory() {
	var fi os.FileInfo
	var err error

	// Set up a directory object implicitly defining a directory in GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedBucket,
		"foo/bar/",
		[]byte(""))

	AssertEq(nil, err)

	// The directory should appear to exist.
	fi, err = os.Stat(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())
}

func (t *CachingWithImplicitDirsTest) SymlinksWork() {
	var fi os.FileInfo
	var err error

	// Create a file.
	fileName := path.Join(mntDir, "foo")
	const contents = "taco"

	err = ioutil.WriteFile(fileName, []byte(contents), 0400)
	AssertEq(nil, err)

	// Create a symlink to it.
	symlinkName := path.Join(mntDir, "bar")
	err = os.Symlink("foo", symlinkName)
	AssertEq(nil, err)

	// Stat the link.
	fi, err = os.Lstat(symlinkName)
	AssertEq(nil, err)

	ExpectEq("bar", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// Stat the target via the link.
	fi, err = os.Stat(symlinkName)
	AssertEq(nil, err)

	ExpectEq("bar", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(filePerms, fi.Mode())
}

func (t *CachingWithImplicitDirsTest) SymlinksAreTypeCached() {
	var fi os.FileInfo
	var err error

	// Create a symlink.
	symlinkName := path.Join(mntDir, "foo")
	err = os.Symlink("blah", symlinkName)
	AssertEq(nil, err)

	// Create a directory object out of band, so the root inode doesn't notice.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedBucket,
		"foo/",
		[]byte(""))

	AssertEq(nil, err)

	// The directory should not yet be visible, because the root inode should
	// have cached that the symlink is present under the name "foo".
	fi, err = os.Lstat(path.Join(mntDir, "foo"))

	AssertEq(nil, err)
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// After the TTL elapses, we should see the directory.
	cacheClock.AdvanceTime(ttl + time.Millisecond)
	fi, err = os.Lstat(path.Join(mntDir, "foo"))

	AssertEq(nil, err)
	ExpectEq(dirPerms|os.ModeDir, fi.Mode())

	// And should be able to stat the symlink under the alternative name.
	fi, err = os.Lstat(path.Join(mntDir, "foo"+inode.ConflictingFileNameSuffix))

	AssertEq(nil, err)
	ExpectEq("foo"+inode.ConflictingFileNameSuffix, fi.Name())
	ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

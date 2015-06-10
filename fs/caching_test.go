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
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

const ttl = 10 * time.Minute

type cachingTestCommon struct {
	fsTest
	uncachedBucket gcs.Bucket
}

func (s *cachingTestCommon) SetUp(t *ogletest.T) {
	// Wrap the bucket in a stat caching layer for the purposes of the file
	// system.
	s.uncachedBucket = gcsfake.NewFakeBucket(&s.clock, "some_bucket")

	const statCacheCapacity = 1000
	statCache := gcscaching.NewStatCache(statCacheCapacity)
	s.bucket = gcscaching.NewFastStatBucket(
		ttl,
		statCache,
		&s.clock,
		s.uncachedBucket)

	// Enable directory type caching.
	s.serverCfg.DirTypeCacheTTL = ttl

	// Call through.
	s.fsTest.SetUp(t)
}

////////////////////////////////////////////////////////////////////////
// Caching
////////////////////////////////////////////////////////////////////////

type CachingTest struct {
	cachingTestCommon
}

func init() { ogletest.RegisterTestSuite(&CachingTest{}) }

func (s *CachingTest) EmptyBucket(t *ogletest.T) {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(s.Dir)
	t.AssertEq(nil, err)

	t.ExpectThat(entries, ElementsAre())
}

func (s *CachingTest) FileCreatedRemotely(t *ogletest.T) {
	const name = "foo"
	const contents = "taco"

	var fi os.FileInfo

	// Create an object in GCS.
	_, err := gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		name,
		contents)

	t.AssertEq(nil, err)

	// It should immediately show up in a listing.
	entries, err := fusetesting.ReadDirPicky(s.Dir)
	t.AssertEq(nil, err)
	t.AssertEq(1, len(entries))

	fi = entries[0]
	t.ExpectEq(name, fi.Name())
	t.ExpectEq(len(contents), fi.Size())

	// And we should be able to stat it.
	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)

	t.ExpectEq(name, fi.Name())
	t.ExpectEq(len(contents), fi.Size())

	// And read it.
	b, err := ioutil.ReadFile(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectEq(contents, string(b))

	// And overwrite it, and read it back again.
	err = ioutil.WriteFile(path.Join(s.Dir, name), []byte("burrito"), 0500)
	t.AssertEq(nil, err)

	b, err = ioutil.ReadFile(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectEq("burrito", string(b))
}

func (s *CachingTest) FileChangedRemotely(t *ogletest.T) {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a file via the file system.
	err = ioutil.WriteFile(path.Join(s.Dir, name), []byte("taco"), 0500)
	t.AssertEq(nil, err)

	// Overwrite the object in GCS.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		name,
		"burrito")

	t.AssertEq(nil, err)

	// Because we are caching, the file should still appear to be the local
	// version.
	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectEq(len("taco"), fi.Size())

	// After the TTL elapses, we should see the new version.
	s.clock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectEq(len("burrito"), fi.Size())

	// Reading should work as expected.
	b, err := ioutil.ReadFile(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectEq("burrito", string(b))
}

func (s *CachingTest) DirectoryRemovedRemotely(t *ogletest.T) {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(s.Dir, name), 0700)
	t.AssertEq(nil, err)

	// Remove the backing object in GCS.
	err = s.uncachedBucket.DeleteObject(s.ctx, name+"/")
	t.AssertEq(nil, err)

	// Because we are caching, the directory should still appear to exist.
	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectTrue(fi.IsDir())

	// After the TTL elapses, we should see it disappear.
	s.clock.AdvanceTime(ttl + time.Millisecond)

	_, err = os.Stat(path.Join(s.Dir, name))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (s *CachingTest) ConflictingNames_RemoteModifier(t *ogletest.T) {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(s.Dir, name), 0700)
	t.AssertEq(nil, err)

	// Create a file with the same name via GCS.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		name,
		"taco")

	t.AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the file
	// when statting.
	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectTrue(fi.IsDir())

	_, err = os.Stat(path.Join(s.Dir, name+inode.ConflictingFileNameSuffix))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see both.
	s.clock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectTrue(fi.IsDir())

	fi, err = os.Stat(path.Join(s.Dir, name+inode.ConflictingFileNameSuffix))
	t.AssertEq(nil, err)
	t.ExpectFalse(fi.IsDir())
}

func (s *CachingTest) TypeOfNameChanges_LocalModifier(t *ogletest.T) {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(s.Dir, name), 0700)
	t.AssertEq(nil, err)

	// Delete it and recreate as a file.
	err = os.Remove(path.Join(s.Dir, name))
	t.AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(s.Dir, name), []byte("taco"), 0400)
	t.AssertEq(nil, err)

	// All caches should have been updated.
	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectFalse(fi.IsDir())
	t.ExpectEq(len("taco"), fi.Size())
}

func (s *CachingTest) TypeOfNameChanges_RemoteModifier(t *ogletest.T) {
	const name = "foo"
	var fi os.FileInfo
	var err error

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(s.Dir, name), 0700)
	t.AssertEq(nil, err)

	// Remove the backing object in GCS, updating the bucket cache (but not the
	// file system type cache)
	err = s.bucket.DeleteObject(s.ctx, name+"/")
	t.AssertEq(nil, err)

	// Create a file with the same name via GCS, again updating the bucket cache.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.bucket,
		name,
		"taco")

	t.AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the name.
	_, err = os.Stat(path.Join(s.Dir, name))
	t.ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see it turn into a file.
	s.clock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(s.Dir, name))
	t.AssertEq(nil, err)
	t.ExpectFalse(fi.IsDir())
}

////////////////////////////////////////////////////////////////////////
// Caching with implicit directories
////////////////////////////////////////////////////////////////////////

type CachingWithImplicitDirsTest struct {
	cachingTestCommon
}

func init() { ogletest.RegisterTestSuite(&CachingWithImplicitDirsTest{}) }

func (s *CachingWithImplicitDirsTest) SetUp(t *ogletest.T) {
	s.serverCfg.ImplicitDirectories = true
	s.cachingTestCommon.SetUp(t)
}

func (s *CachingWithImplicitDirsTest) ImplicitDirectory_DefinedByFile(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Set up a file object implicitly defining a directory in GCS.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		"foo/bar",
		"")

	t.AssertEq(nil, err)

	// The directory should appear to exist.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *CachingWithImplicitDirsTest) ImplicitDirectory_DefinedByDirectory(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Set up a directory object implicitly defining a directory in GCS.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		"foo/bar/",
		"")

	t.AssertEq(nil, err)

	// The directory should appear to exist.
	fi, err = os.Stat(path.Join(s.mfs.Dir(), "foo"))
	t.AssertEq(nil, err)

	t.ExpectEq("foo", fi.Name())
	t.ExpectTrue(fi.IsDir())
}

func (s *CachingWithImplicitDirsTest) SymlinksWork(t *ogletest.T) {
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

	// Stat the link.
	fi, err = os.Lstat(symlinkName)
	t.AssertEq(nil, err)

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

func (s *CachingWithImplicitDirsTest) SymlinksAreTypeCached(t *ogletest.T) {
	var fi os.FileInfo
	var err error

	// Create a symlink.
	symlinkName := path.Join(s.Dir, "foo")
	err = os.Symlink("blah", symlinkName)
	t.AssertEq(nil, err)

	// Create a directory object out of band, so the root inode doesn's notice.
	_, err = gcsutil.CreateObject(
		s.ctx,
		s.uncachedBucket,
		"foo/",
		"")

	t.AssertEq(nil, err)

	// The directory should not yet be visible, because the root inode should
	// have cached that the symlink is present under the name "foo".
	fi, err = os.Lstat(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())

	// After the TTL elapses, we should see the directory.
	s.clock.AdvanceTime(ttl + time.Millisecond)
	fi, err = os.Lstat(path.Join(s.Dir, "foo"))

	t.AssertEq(nil, err)
	t.ExpectEq(dirPerms|os.ModeDir, fi.Mode())

	// And should be able to stat the symlink under the alternative name.
	fi, err = os.Lstat(path.Join(s.Dir, "foo"+inode.ConflictingFileNameSuffix))

	t.AssertEq(nil, err)
	t.ExpectEq("foo"+inode.ConflictingFileNameSuffix, fi.Name())
	t.ExpectEq(filePerms|os.ModeSymlink, fi.Mode())
}

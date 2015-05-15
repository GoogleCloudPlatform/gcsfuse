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

package fstesting

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Caching
////////////////////////////////////////////////////////////////////////

const ttl = 10 * time.Minute

type cachingTest struct {
	fsTest
	uncachedBucket gcs.Bucket
	simulatedClock *timeutil.SimulatedClock
}

func init() { registerSuitePrototype(&cachingTest{}) }

func (t *cachingTest) setUpFSTest(cfg FSTestConfig) {
	// Wrap the bucket in a stat caching layer, saving the original.
	t.uncachedBucket = cfg.ServerConfig.Bucket

	const statCacheCapacity = 1000
	statCache := gcscaching.NewStatCache(statCacheCapacity)
	cfg.ServerConfig.Bucket = gcscaching.NewFastStatBucket(
		ttl,
		statCache,
		cfg.ServerConfig.Clock,
		t.uncachedBucket)

	// Enable directory type caching.
	cfg.ServerConfig.DirTypeCacheTTL = ttl

	// Call through.
	t.fsTest.setUpFSTest(cfg)

	// Is the clock simulated?
	t.simulatedClock, _ = t.clock.(*timeutil.SimulatedClock)
}

func (t *cachingTest) EmptyBucket() {
	// ReadDir
	entries, err := fusetesting.ReadDirPicky(t.Dir)
	AssertEq(nil, err)

	ExpectThat(entries, ElementsAre())
}

func (t *cachingTest) InteractWithNewFile() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) FileCreatedRemotely() {
	const name = "foo"
	const contents = "taco"

	var fi os.FileInfo

	// Create an object in GCS.
	_, err := gcsutil.CreateObject(
		t.ctx,
		t.uncachedBucket,
		name,
		contents)

	AssertEq(nil, err)

	// It should immediately show up in a listing.
	entries, err := fusetesting.ReadDirPicky(t.Dir)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq(name, fi.Name())
	ExpectEq(len(contents), fi.Size())

	// And we should be able to stat it.
	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)

	ExpectEq(name, fi.Name())
	ExpectEq(len(contents), fi.Size())

	// And read it.
	b, err := ioutil.ReadFile(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))

	// And overwrite it, and read it back again.
	err = ioutil.WriteFile(path.Join(t.Dir, name), []byte("burrito"), 0500)
	AssertEq(nil, err)

	b, err = ioutil.ReadFile(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectEq("burrito", string(b))
}

func (t *cachingTest) FileChangedRemotely() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	if t.simulatedClock == nil {
		log.Println("Test requires a simulated clock; skipping.")
		return
	}

	// Create a file via the file system.
	err = ioutil.WriteFile(path.Join(t.Dir, name), []byte("taco"), 0500)
	AssertEq(nil, err)

	// Overwrite the object in GCS.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.uncachedBucket,
		name,
		"burrito")

	AssertEq(nil, err)

	// Because we are caching, the file should still appear to be the local
	// version.
	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectEq(len("taco"), fi.Size())

	// After the TTL elapses, we should see the new version.
	t.simulatedClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectEq(len("burrito"), fi.Size())

	// Reading should work as expected.
	b, err := ioutil.ReadFile(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectEq("burrito", string(b))
}

func (t *cachingTest) InteractWithExistingDirectory() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) DirectoryRemovedRemotely() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	if t.simulatedClock == nil {
		log.Println("Test requires a simulated clock; skipping.")
		return
	}

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(t.Dir, name), 0700)
	AssertEq(nil, err)

	// Remove the backing object in GCS.
	err = t.uncachedBucket.DeleteObject(t.ctx, name+"/")
	AssertEq(nil, err)

	// Because we are caching, the directory should still appear to exist.
	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	// After the TTL elapses, we should see it disappear.
	t.simulatedClock.AdvanceTime(ttl + time.Millisecond)

	_, err = os.Stat(path.Join(t.Dir, name))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)
}

func (t *cachingTest) CreateNewDirectory() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) ImplicitDirectories() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) ConflictingNames_LocalModifier() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) ConflictingNames_RemoteModifier() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	if t.simulatedClock == nil {
		log.Println("Test requires a simulated clock; skipping.")
		return
	}

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(t.Dir, name), 0700)
	AssertEq(nil, err)

	// Create a file with the same name via GCS.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.uncachedBucket,
		name,
		"taco")

	AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the file
	// when statting.
	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	_, err = os.Stat(path.Join(t.Dir, name+inode.ConflictingFileNameSuffix))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see both.
	t.simulatedClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectTrue(fi.IsDir())

	fi, err = os.Stat(path.Join(t.Dir, name+inode.ConflictingFileNameSuffix))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
}

func (t *cachingTest) TypeOfNameChanges_LocalModifier() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	if t.simulatedClock == nil {
		log.Println("Test requires a simulated clock; skipping.")
		return
	}

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(t.Dir, name), 0700)
	AssertEq(nil, err)

	// Delete it and recreate as a file.
	err = os.Remove(path.Join(t.Dir, name))
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.Dir, name), []byte("taco"), 0400)
	AssertEq(nil, err)

	// All caches should have been updated.
	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
	ExpectEq(len("taco"), fi.Size())
}

func (t *cachingTest) TypeOfNameChanges_RemoteModifier() {
	const name = "foo"
	var fi os.FileInfo
	var err error

	if t.simulatedClock == nil {
		log.Println("Test requires a simulated clock; skipping.")
		return
	}

	// Create a directory via the file system.
	err = os.Mkdir(path.Join(t.Dir, name), 0700)
	AssertEq(nil, err)

	// Remove the backing object in GCS, updating the bucket cache (but not the
	// file system type cache)
	err = t.bucket.DeleteObject(t.ctx, name+"/")
	AssertEq(nil, err)

	// Create a file with the same name via GCS, again updating the bucket cache.
	_, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		name,
		"taco")

	AssertEq(nil, err)

	// Because the file system is caching types, it will fail to find the name.
	_, err = os.Stat(path.Join(t.Dir, name))
	ExpectTrue(os.IsNotExist(err), "err: %v", err)

	// After the TTL elapses, we should see it turn into a file.
	t.simulatedClock.AdvanceTime(ttl + time.Millisecond)

	fi, err = os.Stat(path.Join(t.Dir, name))
	AssertEq(nil, err)
	ExpectFalse(fi.IsDir())
}

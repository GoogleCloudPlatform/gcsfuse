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
	"os"
	"path"
	"time"

	"github.com/jacobsa/fuse/fusetesting"
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
}

func init() { registerSuitePrototype(&cachingTest{}) }

func (t *cachingTest) setUpFSTest(cfg FSTestConfig) {
	// Wrap the bucket in a stat caching layer.
	const statCacheCapacity = 1000
	statCache := gcscaching.NewStatCache(statCacheCapacity)
	cfg.ServerConfig.Bucket = gcscaching.NewFastStatBucket(
		ttl,
		statCache,
		cfg.ServerConfig.Clock,
		cfg.ServerConfig.Bucket)

	// Enable directory type caching.
	cfg.ServerConfig.DirTypeCacheTTL = ttl

	// Call through
	t.fsTest.setUpFSTest(cfg)
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
		t.bucket,
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
	AssertTrue(false, "TODO")
}

func (t *cachingTest) InteractWithExistingDirectory() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) DirectoryChangedRemotely() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) CreateNewDirectory() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) ImplicitDirectories() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) ConflictingNames() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) TypeOfNameChanges_LocalModifier() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) TypeOfNameChanges_RemoteModifier() {
	AssertTrue(false, "TODO")
}

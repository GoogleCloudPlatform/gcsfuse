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
	"time"

	"github.com/jacobsa/gcloud/gcs/gcscaching"
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
	AssertTrue(false, "TODO")
}

func (t *cachingTest) InteractWithExistingFile() {
	AssertTrue(false, "TODO")
}

func (t *cachingTest) InteractWithNewFile() {
	AssertTrue(false, "TODO")
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

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
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Caching
////////////////////////////////////////////////////////////////////////

type cachingTest struct {
	fsTest
}

func init() { registerSuitePrototype(&cachingTest{}) }

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

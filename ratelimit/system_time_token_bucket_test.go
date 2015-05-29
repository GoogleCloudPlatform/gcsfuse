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

package ratelimit_test

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestSystemTimeTokenBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SystemTimeTokenBucketTest struct {
}

func init() { RegisterTestSuite(&SystemTimeTokenBucketTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SystemTimeTokenBucketTest) SingleActor() {
	AssertFalse(true, "TODO")
}

func (t *SystemTimeTokenBucketTest) MultipleActors() {
	AssertFalse(true, "TODO")
}

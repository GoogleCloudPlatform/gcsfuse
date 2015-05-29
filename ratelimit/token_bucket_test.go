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
	"time"

	"github.com/googlecloudplatform/gcsfuse/ratelimit"
	. "github.com/jacobsa/ogletest"
)

func TestTokenBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TokenBucketTest struct {
}

func init() { RegisterTestSuite(&TokenBucketTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TokenBucketTest) CarefulAccounting() {
	// Set up a bucket that ticks at the resolution of time.Duration (1 ns) and
	// has a depth of four.
	AssertEq(1, time.Nanosecond)
	tb := ratelimit.NewTokenBucket(1e9, 4)

	// The token starts empty, so initially we should be required to wait one
	// tick per token.
	AssertEq(2, tb.Remove(0, 2))
	AssertEq(3, tb.Remove(2, 1))

	// After the bucket recharges fully, we should be allowed to claim up to its
	// capacity immediately.
	AssertEq(4, tb.Remove(4, 1))
	AssertEq(8, tb.Remove(8, 4))

	// When the bucket fills, it stays full and doesn't let you take more than
	// its capacity immediately.
	AssertEq(100, tb.Remove(100, 4))
	AssertEq(101, tb.Remove(100, 1))
	AssertEq(103, tb.Remove(102, 2))

	// Taking capacity "concurrently" works fine.
	AssertEq(200, tb.Remove(200, 1))
	AssertEq(200, tb.Remove(200, 3))
	AssertEq(201, tb.Remove(200, 1))

	// Attempting to take capacity in the past doesn't screw up the accounting.
	AssertEq(300, tb.Remove(300, 1))
	AssertEq(300, tb.Remove(0, 3))
	AssertEq(302, tb.Remove(301, 2))
}

func (t *TokenBucketTest) AllowsBurstsOfLegalSize() {
	AssertTrue(false, "TODO")
}

func (t *TokenBucketTest) DoesntAllowBurstsOfIllegalSize() {
	AssertTrue(false, "TODO")
}

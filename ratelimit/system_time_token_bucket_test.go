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
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/ratelimit"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestSystemTimeTokenBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func processArrivals(
	tb *ratelimit.SystemTimeTokenBucket,
	arrivalRateHz float64,
	d time.Duration) (processed uint64) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SystemTimeTokenBucketTest struct {
}

func init() { RegisterTestSuite(&SystemTimeTokenBucketTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SystemTimeTokenBucketTest) LimitsSuccessfully() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	const perCaseDuration = time.Second

	// Set up several test cases where we have N goroutines simulating arrival of
	// packets at a given rate, asking a token bucket when to admit them.
	testCases := []struct {
		numActors     int
		arrivalRateHz float64
		limitRateHz   float64
	}{
		// Single actor
		{1, 50, 100},
		{1, 100, 100},
		{1, 150, 100},

		// Multiple actors
		{4, 50, 100},
		{4, 100, 100},
		{4, 150, 100},
	}

	// Run each test case.
	for i, tc := range testCases {
		// Create a token bucket.
		capacity, err := ratelimit.ChooseTokenBucketCapacity(
			tc.limitRateHz,
			perCaseDuration)

		AssertEq(nil, err)

		tb := &ratelimit.SystemTimeTokenBucket{
			Bucket:    ratelimit.NewTokenBucket(tc.limitRateHz, capacity),
			StartTime: time.Now(),
		}

		// Start workers.
		var wg sync.WaitGroup
		var totalProcessed uint64

		for i := 0; i < tc.numActors; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processed := processArrivals(tb, tc.arrivalRateHz, perCaseDuration)
				atomic.AddUint64(&totalProcessed, processed)
			}()
		}

		// Wait for them all to finish.
		wg.Wait()

		// We should have processed about the correct number of arrivals.
		smallerRateHz := tc.arrivalRateHz
		if smallerRateHz > tc.limitRateHz {
			smallerRateHz = tc.limitRateHz
		}

		expected := smallerRateHz * float64(perCaseDuration/time.Second)
		ExpectThat(
			totalProcessed,
			AllOf(
				GreaterThan(expected*0.95),
				LessThan(expected*1.05)),
			"Test case %d. expected: %f",
			i,
			expected)
	}
}

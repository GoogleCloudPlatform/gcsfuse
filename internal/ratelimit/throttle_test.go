// Copyright 2023 Google Inc. All Rights Reserved.
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

// It is performing integration tests for throttle.go
// Set up several test cases where we have N goroutines simulating the arrival of
// packets at a given rate, asking a limiter when to admit them.
// limiter can accept  number of packets equivalent to capacity. After that,
// it will wait until limiter get space to receive the new packet.
package ratelimit_test

import (
	cryptorand "crypto/rand"
	"io"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/ratelimit"
	"golang.org/x/net/context"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestThrottle(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeSeed() (seed int64) {
	var buf [8]byte
	_, err := io.ReadFull(cryptorand.Reader, buf[:])
	if err != nil {
		panic(err)
	}

	seed = (int64(buf[0])>>1)<<56 |
		int64(buf[1])<<48 |
		int64(buf[2])<<40 |
		int64(buf[3])<<32 |
		int64(buf[4])<<24 |
		int64(buf[5])<<16 |
		int64(buf[6])<<8 |
		int64(buf[7])<<0

	return
}

func processArrivals(
	ctx context.Context,
	throttle ratelimit.Throttle,
	arrivalRateHz float64,
	d time.Duration) (processed uint64) {
	// Set up an independent source of randomness.
	randSrc := rand.New(rand.NewSource(makeSeed()))

	// Tick into a channel at a steady rate, buffering over delays caused by the
	// limiter.
	arrivalPeriod := time.Duration((1.0 / arrivalRateHz) * float64(time.Second))
	ticks := make(chan struct{}, 3*int(float64(d)/float64(arrivalPeriod)))

	go func() {
		ticker := time.NewTicker(arrivalPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				select {
				case ticks <- struct{}{}:
				default:
					panic("Buffer exceeded?")
				}
			}
		}
	}()

	// Simulate until we're supposed to stop.
	for {
		// Accumulate a few packets.
		toAccumulate := uint64(randSrc.Int63n(5))

		var accumulated uint64
		for accumulated < toAccumulate {
			select {
			case <-ctx.Done():
				return

			case <-ticks:
				accumulated++
			}
		}

		// Wait.
		err := throttle.Wait(ctx, accumulated)
		if err != nil {
			return
		}

		processed += accumulated
	}
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ThrottleTest struct {
}

func init() { RegisterTestSuite(&ThrottleTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ThrottleTest) IntegrationTest() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	const perCaseDuration = 1 * time.Second

	// Set up several test cases where we have N goroutines simulating arrival of
	// packets at a given rate, asking a limiter when to admit them.
	testCases := []struct {
		numActors     int
		arrivalRateHz float64
		limitRateHz   float64
	}{
		// Single actor
		{1, 150, 200},
		{1, 200, 200},
		{1, 250, 200},

		// Multiple actors
		{4, 150, 200},
		{4, 200, 200},
		{4, 250, 200},
	}

	// Run each test case.
	for i, tc := range testCases {
		// Create a throttle.
		capacity, err := ratelimit.ChooseLimiterCapacity(
			tc.limitRateHz,
			perCaseDuration)

		AssertEq(nil, err)

		throttle := ratelimit.NewThrottle(tc.limitRateHz, capacity)

		// Start workers.
		var wg sync.WaitGroup
		var totalProcessed uint64

		ctx, _ := context.WithDeadline(
			context.Background(),
			time.Now().Add(perCaseDuration))

		for i := 0; i < tc.numActors; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processed := processArrivals(
					ctx,
					throttle,
					tc.arrivalRateHz/float64(tc.numActors),
					perCaseDuration)

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

		expected := smallerRateHz * (float64(perCaseDuration) / float64(time.Second))
		ExpectThat(
			totalProcessed,
			AllOf(
				GreaterThan(expected*0.90),
				LessThan(expected*1.10)),
			"Test case %d. expected: %f",
			i,
			expected)
	}
}

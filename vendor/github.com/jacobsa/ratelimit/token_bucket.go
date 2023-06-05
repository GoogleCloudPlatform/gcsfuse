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

package ratelimit

import (
	"fmt"
	"math"
	"time"
)

// A measurement of the amount of real time since some fixed epoch.
//
// TokenBucket doesn't care about calendar time, time of day, etc.
// Unfortunately time.Time takes these things into account, and in particular
// time.Now() is not monotonic -- it may jump arbitrarily far into the future
// or past when the system's wall time is changed.
//
// Instead we reckon in terms of a monotonic measurement of time elapsed since
// the bucket was initialized, and leave it up to the user to provide this. See
// SystemTimeTokenBucket for a convenience in doing so.
type MonotonicTime time.Duration

// A bucket of tokens that refills at a specific rate up to a particular
// capacity. Users can remove tokens in sizes up to that capacity, can are told
// how long they should wait before proceeding.
//
// If users cooperate by waiting to take whatever action they are rate limiting
// as told by the token bucket, the overall action rate will be limited to the
// token bucket's fill rate.
//
// Not safe for concurrent access; requires external synchronization.
//
// Cf. http://en.wikipedia.org/wiki/Token_bucket
type TokenBucket interface {
	CheckInvariants()

	// Return the maximum number of tokens that the bucket can hold.
	Capacity() (c uint64)

	// Remove the specified number of tokens from the token bucket at the given
	// time. The user should wait until sleepUntil before proceeding in order to
	// obey the rate limit.
	//
	// REQUIRES: tokens <= Capacity()
	Remove(
		now MonotonicTime,
		tokens uint64) (sleepUntil MonotonicTime)
}

// Choose a token bucket capacity that ensures that the action gated by the
// token bucket will be limited to within a few percent of `rateHz * window`
// for any window of the given size.
//
// This is not be possible for all rates and windows. In that case, an error
// will be returned.
func ChooseTokenBucketCapacity(
	rateHz float64,
	window time.Duration) (capacity uint64, err error) {
	// Check that the input is reasonable.
	if rateHz <= 0 || math.IsInf(rateHz, 0) {
		err = fmt.Errorf("Illegal rate: %f", rateHz)
		return
	}

	if window <= 0 {
		err = fmt.Errorf("Illegal window: %v", window)
		return
	}

	// We cannot help but allow the rate to exceed the configured maximum by some
	// factor in an arbitrary window, no matter how small we scale the max
	// accumulated credit -- the bucket may be full at the start of the window,
	// be immediately exhausted, then be repeatedly exhausted just before filling
	// throughout the window.
	//
	// For example: let the window W = 10 seconds, and the bandwidth B = 20 MiB/s.
	// Set the max accumulated credit C = W*B/2 = 100 MiB. Then this
	// sequence of events is allowed:
	//
	//  *  T=0:        Allow through 100 MiB.
	//  *  T=4.999999: Allow through nearly 100 MiB.
	//  *  T=9.999999: Allow through nearly 100 MiB.
	//
	// Above we allow through nearly 300 MiB, exceeding the allowed bytes for the
	// window by nearly 50%. Note however that this trend cannot continue into
	// the next window, so this must be a transient spike.
	//
	// In general if we set C <= W*B/N, then we're off by no more than a factor
	// of (N+1)/N within any window of size W.
	//
	// Choose a reasonable N.
	const N = 50 // At most 2% error

	w := float64(window) / float64(time.Second)
	capacityFloat := math.Floor(w * rateHz / N)
	if !(capacityFloat >= 1 && capacityFloat < float64(math.MaxUint64)) {
		err = fmt.Errorf(
			"Can't use a token bucket to limit to %f Hz over a window of %v "+
				"(result is a capacity of %f)",
			rateHz,
			window,
			capacityFloat)

		return
	}

	capacity = uint64(capacityFloat)
	if capacity == 0 {
		panic(fmt.Sprintf(
			"Calculated a zero capacity for inputs %f, %v. Float version: %f",
			rateHz,
			window,
			capacityFloat))
	}

	return
}

// Create a token bucket that fills at the given rate in tokens per second, up
// to the given capacity. ChooseTokenBucketCapacity may help you decide on a
// capacity.
//
// The token bucket starts full at time zero. If you would like it to start
// empty, call tb.Remove(0, capacity).
//
// REQUIRES: rateHz > 0
// REQUIRES: capacity > 0
func NewTokenBucket(
	rateHz float64,
	capacity uint64) (tb TokenBucket) {
	tb = &tokenBucket{
		rateHz:   rateHz,
		capacity: capacity,

		creditTime: 0,
		credit:     float64(capacity),
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type tokenBucket struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	rateHz   float64
	capacity uint64

	/////////////////////////
	// Mutable state
	/////////////////////////

	// The time that we last updated the bucket's credit. Only moves forward.
	creditTime MonotonicTime

	// The number of credits that were available at creditTime.
	//
	// INVARIANT: credit <= float64(capacity)
	credit float64
}

func (tb *tokenBucket) CheckInvariants() {
	// INVARIANT: credit <= float64(capacity)
	if !(tb.credit <= float64(tb.capacity)) {
		panic(fmt.Sprintf(
			"Illegal credit: %f, capacity: %d",
			tb.credit,
			tb.capacity))
	}
}

func (tb *tokenBucket) Capacity() (c uint64) {
	c = tb.capacity
	return
}

func (tb *tokenBucket) Remove(
	now MonotonicTime,
	tokens uint64) (sleepUntil MonotonicTime) {
	if tokens > tb.capacity {
		panic(fmt.Sprintf(
			"Token count %d out of range; capacity is %d",
			tokens,
			tb.capacity))
	}

	// First play the clock forward until now, crediting any tokens that have
	// accumulated in the meantime, up to the bucket's capacity.
	if tb.creditTime < now {
		diff := now - tb.creditTime

		// Don't forget to cap at the capacity.
		tb.credit += tb.rateHz * float64(diff) / float64(time.Second)
		if !(tb.credit <= float64(tb.capacity)) {
			tb.credit = float64(tb.capacity)
		}

		tb.creditTime = now
	}

	// Deduct the requested tokens. The user will need to wait until the credit
	// makes it back to zero, which is when it would have otherwise made it to
	// `tokens`.
	tb.credit -= float64(tokens)

	sleepUntil = tb.creditTime
	if tb.credit < 0 {
		seconds := -tb.credit / tb.rateHz
		sleepUntil = tb.creditTime + MonotonicTime(seconds*float64(time.Second))
	}

	return
}

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

package syncutil

import "time"

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
// Cf. http://en.wikipedia.org/wiki/Token_bucket
type TokenBucket interface {
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

// TODO(jacobsa): Comments.
func ChooseTokenBucketCapacity(
	rateHz float64,
	window time.Duration) (capacity uint64, err error)

// Create a token bucket that fills at the given rate in tokens per second, up
// to the given capacity.
//
// REQUIRES: rateHz > 0
// REQUIRES: capacity > 0
func NewTokenBucket(
	rateHz float64,
	capacity uint64) (tb TokenBucket)

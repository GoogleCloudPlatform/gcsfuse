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
	"time"

	"github.com/jacobsa/syncutil"

	"golang.org/x/net/context"
)

// A simple interface for limiting the rate of some event. Unlike TokenBucket,
// does not allow the user control over what time means.
//
// Safe for concurrent access.
type Throttle interface {
	// Return the maximum number of tokens that can be requested in a call to
	// Wait.
	Capacity() (c uint64)

	// Acquire the given number of tokens from the underlying token bucket, then
	// sleep until when it says to wake. If the context is cancelled before then,
	// return early with an error.
	//
	// REQUIRES: tokens <= capacity
	Wait(ctx context.Context, tokens uint64) (err error)
}

// Create a throttle that uses time.Now to judge the time given to the
// underlying token bucket.
//
// Be aware of the monotonicity issues. In particular:
//
//  *  If the system clock jumps into the future, the throttle will let through
//     a burst of traffic.
//
//  *  If the system clock jumps into the past, it will halt all traffic for
//     a potentially very long amount of time.
//
func NewThrottle(
	rateHz float64,
	capacity uint64) (t Throttle) {
	typed := &throttle{
		startTime: time.Now(),
		bucket:    NewTokenBucket(rateHz, capacity),
	}

	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	t = typed
	return
}

type throttle struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	startTime time.Time

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// INVARIANT: bucket.CheckInvariants()
	//
	// GUARDED_BY(mu)
	bucket TokenBucket
}

// LOCKS_REQUIRED(t.mu)
func (t *throttle) checkInvariants() {
	// INVARIANT: bucket.CheckInvariants()
	t.bucket.CheckInvariants()
}

// LOCKS_EXCLUDED(t.mu)
func (t *throttle) Capacity() (c uint64) {
	t.mu.Lock()
	c = t.bucket.Capacity()
	t.mu.Unlock()

	return
}

// LOCKS_EXCLUDED(t.mu)
func (t *throttle) Wait(
	ctx context.Context,
	tokens uint64) (err error) {
	now := MonotonicTime(time.Now().Sub(t.startTime))

	t.mu.Lock()
	sleepUntil := t.bucket.Remove(now, tokens)
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case <-time.After(time.Duration(sleepUntil - now)):
		return
	}
}

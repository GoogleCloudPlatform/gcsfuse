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

package ratelimit

import (
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
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

type limiter struct {
	*rate.Limiter
}

func NewThrottle(
	rateHz float64,
	capacity int) (t Throttle) {
	t = &limiter{rate.NewLimiter(rate.Limit(rateHz), capacity)}
	return
}

func (l *limiter) Capacity() (c uint64) {
	return uint64(l.Burst())
}

func (l *limiter) Wait(
	ctx context.Context,
	tokens uint64) (err error) {
	return l.WaitN(ctx, int(tokens))
}

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

import "time"

// A simple wrapper around TokenBucket that groks time.Time. Time values are
// converted into MonotonicTime by subtracting StartTime.
//
// If you use this with time.Now, be aware of the monotonicity issues. In
// particular:
//
//  *  If the system clock jumps into the future, the token bucket will let
//     through a burst of traffic.
//
//  *  If the system clock jumps into the past, it will halt all traffic for
//     a potentially very long amount of time.
//
type SystemTimeTokenBucket struct {
	Bucket    TokenBucket
	StartTime time.Time
}

func (tb *SystemTimeTokenBucket) Capacity(c uint64) {
	panic("TODO")
}

func (tb *SystemTimeTokenBucket) Remove(
	now time.Time,
	tokens uint64) (sleepUntil time.Time) {
	panic("TODO")
}

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
// TimeBasedTokenBucket for a convenience in doing so.
type MonotonicTime time.Duration

// TODO(jacobsa): Comments.
type TokenBucket interface {
	// The maximum number of tokens that the bucket can hold.
	Capacity() (c uint64)

	// TODO(jacobsa): Comments.
	Remove(
		now MonotonicTime,
		tokens uint64) (sleepUntil MonotonicTime)
}

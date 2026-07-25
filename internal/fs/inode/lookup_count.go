// Copyright 2015 Google LLC
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

package inode

import (
	"fmt"
	"sync/atomic"
)

// A helper struct for implementing lookup counts. The only value added is some
// paranoid panics. External synchronization is required.
//
// May be embedded within a larger struct. Use Init to initialize.
type lookupCount int64

func (lc *lookupCount) IncrementLookupCount() {
	val := atomic.LoadInt64((*int64)(lc))
	if val == -1 {
		panic("Inode has already been destroyed")
	}
	atomic.AddInt64((*int64)(lc), 1)
}

func (lc *lookupCount) DecrementLookupCount(n uint64) (destroy bool) {
	val := atomic.LoadInt64((*int64)(lc))
	if val == -1 {
		panic("Inode has already been destroyed")
	}

	if n > uint64(val) {
		panic(fmt.Sprintf(
			"n is greater than lookup count: %v vs. %v",
			n,
			val))
	}

	newVal := atomic.AddInt64((*int64)(lc), -int64(n))
	return newVal == 0
}

func (lc *lookupCount) Destroy() {
	atomic.StoreInt64((*int64)(lc), -1)
}

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

package inode

import (
	"fmt"

	"github.com/jacobsa/fuse/fuseops"
)

// A helper struct for implementing lookup counts. The only value added is some
// paranoid panics. External synchronization is required.
//
// May be embedded within a larger struct. Use Init to initialize.
type lookupCount struct {
	id        fuseops.InodeID
	count     uint64
	destroyed bool
}

func (lc *lookupCount) Init(id fuseops.InodeID) {
	lc.id = id
}

func (lc *lookupCount) Inc() {
	if lc.destroyed {
		panic(fmt.Sprintf("Inode %v has already been destroyed", lc.id))
	}

	lc.count++
}

func (lc *lookupCount) Dec(n uint64) (destroy bool) {
	if lc.destroyed {
		panic(fmt.Sprintf("Inode %v has already been destroyed", lc.id))
	}

	// Make sure n is in range.
	if n > lc.count {
		panic(fmt.Sprintf(
			"n is greater than lookup count: %v vs. %v",
			n,
			lc.count))
	}

	// Decrement.
	lc.count -= n

	destroy = lc.count == 0
	return
}

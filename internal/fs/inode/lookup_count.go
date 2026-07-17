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

	"github.com/jacobsa/fuse/fuseops"
)

// A helper struct for implementing lookup counts. The only value added is some
// paranoid panics. External synchronization is required.
//
// May be embedded within a larger struct. Use Init to initialize.
type lookupCount int64

func (lc *lookupCount) Inc(id fuseops.InodeID) {
	if *lc == -1 {
		panic(fmt.Sprintf("Inode %v has already been destroyed", id))
	}

	(*lc)++
}

func (lc *lookupCount) Dec(id fuseops.InodeID, n uint64) (destroy bool) {
	if *lc == -1 {
		panic(fmt.Sprintf("Inode %v has already been destroyed", id))
	}

	// Make sure n is in range.
	if n > uint64(*lc) {
		panic(fmt.Sprintf(
			"n is greater than lookup count: %v vs. %v",
			n,
			*lc))
	}

	// Decrement.
	*lc -= lookupCount(n)

	destroy = *lc == 0
	return
}

func (lc *lookupCount) Destroy() {
	*lc = -1
}

// Copyright 2026 Google LLC
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
	"math"
	"testing"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
)

func TestLookupCount_Inc_Normal(t *testing.T) {
	var lc lookupCount
	id := fuseops.InodeID(1)

	lc.Inc(id)

	assert.Equal(t, lookupCount(1), lc)
}

func TestLookupCount_Inc_PanicsWhenDestroyed(t *testing.T) {
	var lc lookupCount = -1
	id := fuseops.InodeID(1)

	assert.Panics(t, func() {
		lc.Inc(id)
	})
}

func TestLookupCount_Dec_Normal(t *testing.T) {
	var lc lookupCount = 2
	id := fuseops.InodeID(1)

	destroy := lc.Dec(id, 1)

	assert.False(t, destroy)
	assert.Equal(t, lookupCount(1), lc)
}

func TestLookupCount_Dec_ReachesZero(t *testing.T) {
	var lc lookupCount = 1
	id := fuseops.InodeID(1)

	destroy := lc.Dec(id, 1)

	assert.True(t, destroy)
	assert.Equal(t, lookupCount(0), lc)
}

func TestLookupCount_Dec_PanicsWhenDestroyed(t *testing.T) {
	var lc lookupCount = -1
	id := fuseops.InodeID(1)

	assert.Panics(t, func() {
		lc.Dec(id, 1)
	})
}

func TestLookupCount_Dec_PanicsOnUnderflow(t *testing.T) {
	var lc lookupCount = 1
	id := fuseops.InodeID(1)

	assert.Panics(t, func() {
		lc.Dec(id, 2)
	})
}

func TestLookupCount_Dec_ProtectsAgainstOverflowWrap(t *testing.T) {
	var lc lookupCount = 1
	id := fuseops.InodeID(1)

	// Providing math.MaxUint64 would wrap to -1 if we casted it directly to int64.
	// This ensures our uint64 comparison works safely.
	assert.Panics(t, func() {
		lc.Dec(id, math.MaxUint64)
	})
}

func TestLookupCount_Destroy_IsIdempotent(t *testing.T) {
	var lc lookupCount = 5

	lc.Destroy()
	lc.Destroy() // Second call should not panic

	assert.Equal(t, lookupCount(-1), lc)
}

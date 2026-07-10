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

package lru_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
)

// mutableTestData simulates an object whose size grows dynamically in memory,
// such as an in-memory file chunk buffer during download in GCSFuse.
type mutableTestData struct {
	size uint64
}

func (m *mutableTestData) Size() uint64 {
	return m.size
}

// TestUpdateSize_DoubleCountingDivergence demonstrates the double-counting bug in radixCache.
// When an underlying cached object's size is mutated before calling UpdateSize(key, delta),
// mapCache succeeds, but radixCache fails with ErrInvalidEntrySize because it adds delta
// to the already-updated node.value.Size().
func TestUpdateSize_DoubleCountingDivergence(t *testing.T) {
	const maxSize = 100
	const initialSize = 50
	const sizeDelta = 50 // New total size will be 50 + 50 = 100 (exactly at maxSize)

	mapCache := lru.NewCache(maxSize)
	radixCache := lru.NewRadixCache(maxSize)

	mapVal := &mutableTestData{size: initialSize}
	radixVal := &mutableTestData{size: initialSize}

	// 1. Insert initial entries into both caches
	_, err := mapCache.Insert("file.txt", mapVal)
	assert.NoError(t, err)
	_, err = radixCache.Insert("file.txt", radixVal)
	assert.NoError(t, err)

	// 2. Simulate incremental file growth in memory (e.g., downloading a 50-byte chunk)
	// Both objects now report Size() == 100.
	mapVal.size += sizeDelta
	radixVal.size += sizeDelta

	// 3. Notify mapCache of the growth.
	// mapCache simply adds sizeDelta to c.currentSize without double counting.
	err = mapCache.UpdateSize("file.txt", sizeDelta)
	assert.NoError(t, err, "mapCache should succeed when updated size (100) <= maxSize (100)")

	// 4. Notify radixCache of the growth.
	// radixCache evaluates node.value.Size() (100) + sizeDelta (50) = 150 > maxSize (100).
	// This double-counts sizeDelta and incorrectly rejects a valid update.
	err = radixCache.UpdateSize("file.txt", sizeDelta)
	assert.ErrorIs(t, err, lru.ErrInvalidEntrySize, "radixCache incorrectly fails due to double counting (100 + 50 = 150 > 100)")
}

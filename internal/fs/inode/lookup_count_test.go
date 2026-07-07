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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupCount_Basic(t *testing.T) {
	var lc lookupCount
	lc.Init(123)

	assert.Equal(t, uint64(0), lc.count)

	lc.Inc()
	assert.Equal(t, uint64(1), lc.count)

	lc.Inc()
	assert.Equal(t, uint64(2), lc.count)

	destroy := lc.Dec(1)
	assert.False(t, destroy)
	assert.Equal(t, uint64(1), lc.count)

	destroy = lc.Dec(1)
	assert.True(t, destroy)
	assert.Equal(t, uint64(0), lc.count)
}

func TestLookupCount_PanicOnInvalidDec(t *testing.T) {
	var lc lookupCount
	lc.Init(123)

	assert.Panics(t, func() {
		lc.Dec(1)
	})

	lc.Inc()
	assert.Panics(t, func() {
		lc.Dec(2)
	})
}

func TestLookupCount_PanicOnDestroyed(t *testing.T) {
	var lc lookupCount
	lc.Init(123)
	lc.destroyed = true

	assert.Panics(t, func() {
		lc.Inc()
	})

	assert.Panics(t, func() {
		lc.Dec(1)
	})
}

func TestLookupCount_Concurrent(t *testing.T) {
	var lc lookupCount
	lc.Init(123)

	const numWorkers = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				lc.Inc()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, uint64(numWorkers*iterations), lc.count)

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				lc.Dec(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, uint64(0), lc.count)
}

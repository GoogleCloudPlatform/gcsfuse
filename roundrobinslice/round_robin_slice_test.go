// Copyright 2025 Google LLC
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

package roundrobinslice

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Empty(t *testing.T) {
	rr := New[int](nil)
	assert.NotNil(t, rr)
	val, ok := rr.Get()
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	rr = New([]int{})
	assert.NotNil(t, rr)
	val, ok = rr.Get()
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestGet_SingleElement(t *testing.T) {
	rr := New([]int{42})

	// First Get
	val, ok := rr.Get()
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	// Second Get
	val, ok = rr.Get()
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestGet_MultipleElements(t *testing.T) {
	items := []string{"a", "b", "c"}
	rr := New(items)

	expectedSequence := []string{"a", "b", "c"}

	// First cycle
	for i, expected := range expectedSequence {
		val, ok := rr.Get()
		assert.True(t, ok, fmt.Sprintf("Cycle 1, Iteration %d", i))
		assert.Equal(t, expected, val, fmt.Sprintf("Cycle 1, Iteration %d", i))
	}

	// Second cycle
	for i, expected := range expectedSequence {
		val, ok := rr.Get()
		assert.True(t, ok, fmt.Sprintf("Cycle 2, Iteration %d", i))
		assert.Equal(t, expected, val, fmt.Sprintf("Cycle 2, Iteration %d", i))
	}
}

func TestGet_ThreadSafety(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	rr := New(items)
	numGoroutines := 10
	getsPerGoroutine := 100
	totalGets := numGoroutines * getsPerGoroutine

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make(chan int, totalGets)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < getsPerGoroutine; j++ {
				val, ok := rr.Get()
				if ok {
					results <- val
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	assert.Equal(t, totalGets, len(results))

	// We can check the distribution
	counts := make(map[int]int)
	for val := range results {
		counts[val]++
	}

	// Each item should appear exactly `totalGets / len(items)` times.
	expectedCount := totalGets / len(items)
	assert.Equal(t, len(items), len(counts))
	for _, item := range items {
		assert.Equal(t, expectedCount, counts[item], "Distribution of item %d should be even", item)
	}
}

func TestGet_Structs(t *testing.T) {
	type myStruct struct {
		ID   int
		Name string
	}

	items := []myStruct{
		{1, "one"},
		{2, "two"},
	}
	rr := New(items)

	val, ok := rr.Get()
	assert.True(t, ok)
	assert.Equal(t, items[0], val)

	val, ok = rr.Get()
	assert.True(t, ok)
	assert.Equal(t, items[1], val)

	val, ok = rr.Get()
	assert.True(t, ok)
	assert.Equal(t, items[0], val)
}

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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_NilSlice(t *testing.T) {
	var items []int // nil

	rr := New(items)
	val, ok := rr.Get()

	assert.NotNil(t, rr)
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestNew_EmptySlice(t *testing.T) {
	items := []int{}

	rr := New(items)
	val, ok := rr.Get()

	assert.NotNil(t, rr)
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestGet_SingleElement(t *testing.T) {
	items := []int{42}
	rr := New(items)

	val, ok := rr.Get()

	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestGet_SingleElement_MultipleTimes(t *testing.T) {
	items := []int{42}
	rr := New(items)

	// Call Get multiple times to ensure it consistently returns the single element.
	for range 3 {
		val, ok := rr.Get()

		assert.True(t, ok)
		assert.Equal(t, 42, val)
	}
}

func TestGet_MultipleElements(t *testing.T) {
	items := []string{"a", "b", "c"}
	rr := New(items)
	expectedSequence := []string{"a", "b", "c", "a", "b", "c"}

	for i, expected := range expectedSequence {
		val, ok := rr.Get()

		assert.True(t, ok, "Iteration %d: Get should succeed", i)
		assert.Equal(t, expected, val, "Iteration %d: Incorrect value", i)
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

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range getsPerGoroutine {
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
	// Check the distribution.
	counts := make(map[int]int)
	for val := range results {
		counts[val]++
	}
	// Each item should appear exactly `totalGets / len(items)` times.
	//assert.Equal(t, totalGets, len(results))
	expectedCount := totalGets / len(items)
	assert.Equal(t, len(items), len(counts))
	for _, item := range items {
		assert.Equal(t, expectedCount, counts[item])
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
	// Check for wrap-around.
	expectedSequence := []myStruct{items[0], items[1], items[0]}

	for i, expected := range expectedSequence {
		val, ok := rr.Get()

		assert.True(t, ok, "Iteration %d: Get should succeed", i)
		assert.Equal(t, expected, val, "Iteration %d: Incorrect value", i)
	}
}

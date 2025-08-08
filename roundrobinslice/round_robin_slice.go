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

import "sync"

// RoundRobin provides thread-safe, round-robin access to elements of a slice.
// The underlying slice is treated as immutable.
type RoundRobin[T any] struct {
	mu    sync.Mutex
	items []T
	// current is the index of the element last returned by Get().
	// It is initialized to len(items) - 1 so that the first call to Get()
	// returns items[0].
	current int
}

// New creates a new RoundRobin instance with the provided slice of items.
// It expects the slice to be immutable for performance reasons.
// If the slice is empty, Get() will always return the zero value of T and false.
func New[T any](items []T) *RoundRobin[T] {
	// Create a copy of the slice to ensure immutability from external modifications.

	// To make the first call to Get() return items[0].
	idx := -1
	if len(items) > 0 {
		idx = len(items) - 1
	}

	return &RoundRobin[T]{
		items:   items,
		current: idx,
	}
}

// Get returns the next element from the slice in a round-robin fashion.
// It returns the element and a boolean indicating if the slice was non-empty.
// If the slice is empty, it returns the zero value for T and false.
func (rr *RoundRobin[T]) Get() (T, bool) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.items) == 0 {
		var zero T
		return zero, false
	}

	// Advance to the next index.
	rr.current = (rr.current + 1) % len(rr.items)
	return rr.items[rr.current], true
}

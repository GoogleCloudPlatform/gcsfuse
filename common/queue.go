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

package common

// node represents a node in the queue.
type node[T any] struct {
	value T
	next  *node[T]
}

// Queue is a generic queue implementation using a linked list.
type Queue[T any] struct {
	start, end *node[T]
	size       int
}

// NewQueue creates a new empty queue.
func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		nil,
		nil,
		0,
	}
}

// IsEmpty returns true if the queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return q.size == 0
}

// Peek returns the front of the queue without removing it.
func (q *Queue[T]) Peek() T {
	if q.size == 0 {
		// Returns zero value of type T if the queue is empty.
		var zero T
		return zero
	}
	return q.start.value
}

// Push puts an item on the end of the queue.
func (q *Queue[T]) Push(value T) {
	n := &node[T]{value, nil}
	if q.size == 0 {
		q.start = n
		q.end = n
	} else {
		q.end.next = n
		q.end = n
	}
	q.size++
}

// Pop removes and returns the front item from the queue.
func (q *Queue[T]) Pop() T {
	if q.size == 0 {
		// Returns zero value of type T if the queue is empty.
		var zero T
		return zero
	}

	n := q.start
	if q.size == 1 {
		q.start = nil
		q.end = nil
	} else {
		q.start = q.start.next
	}
	q.size--
	return n.value
}

// Len returns the number of items in the queue.
func (q *Queue[T]) Len() int {
	return q.size
}

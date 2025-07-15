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

// Queue is a generic interface for a queue data structure.
type Queue[T any] interface {
	// IsEmpty checks if the queue is empty.
	IsEmpty() bool

	// PeekStart returns the front of the queue without removing it.
	// Panics if the queue is empty.
	PeekStart() T

	// PeekEnd returns the back of the queue without removing it.
	// Panics if the queue is empty.
	PeekEnd() T

	// Push puts an item on the end of the queue.
	Push(value T)

	// Pop removes and returns the front item from the queue.
	// Panics if the queue is empty.
	Pop() T

	// Len returns the number of items in the queue.
	Len() int
}

// node represents a node in the queue.
type node[T any] struct {
	value T
	next  *node[T]
}

// linkedListQueue is a linked list implementation of a queue.
type linkedListQueue[T any] struct {
	start, end *node[T]
	size       int
}

// NewQueue creates a new empty queue.
func NewLinkedListQueue[T any]() Queue[T] {
	return &linkedListQueue[T]{}
}

// IsEmpty returns true if the queue is empty.
func (q *linkedListQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// PeekStart returns the front of the queue without removing it.
// Panics if the queue is empty.
func (q *linkedListQueue[T]) PeekStart() T {
	if q.size == 0 {
		panic("PeekStart called on an empty queue.")
	}
	return q.start.value
}

// PeekEnd returns the back of the queue without removing it.
// Panics if the queue is empty.
func (q *linkedListQueue[T]) PeekEnd() T {
	if q.size == 0 {
		panic("PeekEnd called on an empty queue.")
	}
	return q.end.value
}

// Push puts an item on the end of the queue.
func (q *linkedListQueue[T]) Push(value T) {
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
// Panics if the queue is empty.
func (q *linkedListQueue[T]) Pop() T {
	if q.size == 0 {
		panic("Pop called on an empty queue.")
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
func (q *linkedListQueue[T]) Len() int {
	return q.size
}

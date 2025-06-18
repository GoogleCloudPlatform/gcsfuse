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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue[int]()

	assert.NotNil(t, q, "NewQueue() should return a non-nil queue.")
	assert.True(t, q.IsEmpty(), "A new queue should be empty.")
	assert.Equal(t, 0, q.Len(), "A new queue should have a size of 0.")
}

func TestQueue_Push(t *testing.T) {
	q := NewQueue[int]()

	q.Push(4)
	q.Push(5)

	assert.Equal(t, 4, q.Peek())
	assert.False(t, q.IsEmpty())
}

func TestQueue_Pop(t *testing.T) {
	q := NewQueue[int]()
	q.Push(4)
	q.Push(5)
	require.Equal(t, 4, q.Peek())
	require.False(t, q.IsEmpty())

	val := q.Pop()
	assert.Equal(t, 4, val)
	assert.Equal(t, 5, q.Peek())

	val = q.Pop()
	assert.Equal(t, 5, val)
	assert.Zero(t, q.Peek())
	assert.True(t, q.IsEmpty())

	val = q.Pop()
	assert.Zero(t, val)
	assert.True(t, q.IsEmpty())
}

func TestQueue_Peek(t *testing.T) {
	// Empty queue.
	q := NewQueue[int]()
	val := q.Peek()
	assert.Zero(t, val)

	// Non-empty queue.
	q.Push(4)
	val = q.Peek()
	assert.Equal(t, 4, val)

	// Peek again to ensure it doesn't remove the element.
	val = q.Peek()
	assert.Equal(t, 4, val)
	assert.False(t, q.IsEmpty())
	assert.Equal(t, 1, q.Len())
}

func TestQueue_IsEmpty(t *testing.T) {
	// Empty queue.
	q := NewQueue[int]()
	assert.True(t, q.IsEmpty())

	// Non-empty queue.
	q.Push(4)
	assert.False(t, q.IsEmpty())

	// Make empty by poping the only element.
	val := q.Pop()
	assert.Equal(t, 4, val)
	assert.True(t, q.IsEmpty())

	// Pop from empty queue.
	val = q.Pop()
	assert.Zero(t, val)
	assert.True(t, q.IsEmpty())
}

func TestQueue_Len(t *testing.T) {
	q := NewQueue[int]()
	assert.Equal(t, 0, q.Len())

	q.Push(4)
	assert.Equal(t, 1, q.Len())

	q.Push(5)
	assert.Equal(t, 2, q.Len())

	q.Push(6)
	assert.Equal(t, 3, q.Len())

	val := q.Pop()
	assert.Equal(t, 4, val)
	assert.Equal(t, 2, q.Len())

	val = q.Pop()
	assert.Equal(t, 5, val)
	assert.Equal(t, 1, q.Len())

	val = q.Pop()
	assert.Equal(t, 6, val)
	assert.Equal(t, 0, q.Len())

	// Check Len after popping from empty
	val = q.Pop()
	assert.Zero(t, val)
	assert.Equal(t, 0, q.Len())
}

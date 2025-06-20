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

func TestNewLinkedListQueue(t *testing.T) {
	q := NewLinkedListQueue[int]()

	assert.NotNil(t, q, "NewLinkedListQueue() should return a non-nil queue.")
	assert.True(t, q.IsEmpty(), "A new queue should be empty.")
	assert.Equal(t, 0, q.Len(), "A new queue should have a size of 0.")
}

func TestLinkedListQueue_Push(t *testing.T) {
	q := NewLinkedListQueue[int]()

	q.Push(4)
	q.Push(5)

	assert.Equal(t, 4, q.Peek())
	assert.False(t, q.IsEmpty())
}

func TestLinkedListQueue_SinglePop(t *testing.T) {
	q := NewLinkedListQueue[int]()
	q.Push(4)
	q.Push(5)
	require.Equal(t, 4, q.Peek())
	require.False(t, q.IsEmpty())

	val := q.Pop()

	assert.Equal(t, 4, val)
	assert.Equal(t, 5, q.Peek())
}

func TestLinkedListQueue_MultiplePops(t *testing.T) {
	q := NewLinkedListQueue[int]()
	q.Push(4)
	q.Push(5)
	require.Equal(t, 4, q.Peek())
	require.False(t, q.IsEmpty())
	val := q.Pop()
	require.Equal(t, 4, val)
	require.Equal(t, 5, q.Peek())

	val = q.Pop()

	assert.Equal(t, 5, val)
	assert.True(t, q.IsEmpty())
}

func TestLinkedListQueue_PopEmptyQueue(t *testing.T) {
	assert.Panics(t, func() {
		NewLinkedListQueue[int]().Pop()
	}, "Pop should panic when called on an empty queue.")
}

func TestLinkedListQueue_Peek(t *testing.T) {
	q := NewLinkedListQueue[int]()
	q.Push(4)
	require.Equal(t, 1, q.Len())

	val := q.Peek()

	assert.Equal(t, 4, val)
	assert.Equal(t, 1, q.Len()) // Length should remain unchanged.
	assert.False(t, q.IsEmpty())
}

func TestLinkedListQueue_PeekEmptyQueue(t *testing.T) {
	assert.Panics(t, func() {
		NewLinkedListQueue[int]().Peek()
	}, "Peek should panic when called on an empty queue.")
}

func TestLinkedListQueue_IsEmptyTrue(t *testing.T) {
	q := NewLinkedListQueue[int]()
	q.Push(4)
	q.Pop()

	assert.True(t, q.IsEmpty())
}

func TestLinkedListQueue_IsEmptyFalse(t *testing.T) {
	q := NewLinkedListQueue[int]()
	q.Push(4)

	assert.False(t, q.IsEmpty())
}

func TestLinkedListQueue_Len(t *testing.T) {
	q := NewLinkedListQueue[int]()
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
}

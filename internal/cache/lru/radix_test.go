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

package lru

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockValue implements the ValueType interface for testing
type mockValue struct{ size uint64 }

func (m mockValue) Size() uint64 { return m.size }

func TestRadixTree_Insert(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	val1 := mockValue{10}

	// Act
	node1, isNew := tree.insertNode("foo/bar", val1)

	// Assert
	assert.True(t, isNew)
	assert.NotNil(t, node1)
	assert.Equal(t, 1, tree.size)
}

func TestRadixTree_InsertNil(t *testing.T) {
	// Arrange
	tree := newRadixTree()

	// Act
	node, isNew := tree.insertNode("foo/bar", nil)

	// Assert
	assert.False(t, isNew)
	assert.Nil(t, node)
	assert.Equal(t, 0, tree.size)
}

func TestRadixTree_Get(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	val1 := mockValue{10}
	tree.insertNode("foo/bar", val1)

	// Act
	gotNode, ok := tree.getNode("foo/bar")

	// Assert
	assert.True(t, ok)
	assert.Equal(t, val1, gotNode.value)
}

func TestRadixTree_Overwrite(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	tree.insertNode("foo/bar", mockValue{10})
	val2 := mockValue{20}

	// Act
	node2, isNew2 := tree.insertNode("foo/bar", val2)

	// Assert
	assert.False(t, isNew2)
	assert.Equal(t, val2, node2.value)
	assert.Equal(t, 1, tree.size)
}

func TestRadixTree_GetNonExistent(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	tree.insertNode("foo/bar", mockValue{10})

	// Act
	_, ok := tree.getNode("foo/baz")

	// Assert
	assert.False(t, ok)
}

func TestRadixTree_PrefixSplitting(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	tree.insertNode("foo/bar", mockValue{1})

	// Act
	tree.insertNode("foo/baz", mockValue{2}) // Splits "foo/ba" -> "r", "z"
	n1, _ := tree.getNode("foo/bar")
	n2, _ := tree.getNode("foo/baz")

	// Assert
	assert.Equal(t, 2, tree.size)
	assert.Equal(t, n1.parent, n2.parent)
	assert.Equal(t, "foo/ba", n1.parent.prefix)
	assert.Nil(t, n1.parent.value) // Parent is just a routing node
}

func TestRadixTree_DeleteAndCompress(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	tree.insertNode("foo/bar", mockValue{1})
	tree.insertNode("foo/baz", mockValue{2})
	n1, _ := tree.getNode("foo/bar")

	// Act
	tree.deleteNode(n1)
	_, ok := tree.getNode("foo/bar")
	n2, _ := tree.getNode("foo/baz")

	// Assert
	assert.Equal(t, 1, tree.size)
	assert.False(t, ok)
	assert.Equal(t, "foo/baz", n2.prefix)
	assert.Equal(t, tree.root, n2.parent) // Path compressed all the way up to the root!
}

func TestRadixTree_LRU_PushFront(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	val1 := mockValue{10}
	val2 := mockValue{20}
	node1, _ := tree.insertNode("foo/1", val1)
	node2, _ := tree.insertNode("foo/2", val2)

	// Act
	tree.pushFront(node1)
	tree.pushFront(node2)

	// Assert
	assert.Equal(t, 2, tree.len)
	assert.Equal(t, node2, tree.head)
	assert.Equal(t, node1, tree.tail)
	assert.Equal(t, node1, node2.next)
	assert.Equal(t, node2, node1.prev)
}

func TestRadixTree_LRU_MoveToFront(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	node1, _ := tree.insertNode("foo/1", mockValue{10})
	node2, _ := tree.insertNode("foo/2", mockValue{20})
	tree.pushFront(node1)
	tree.pushFront(node2)

	// Act
	tree.moveToFront(node1) // Move tail to head

	// Assert
	assert.Equal(t, node1, tree.head)
	assert.Equal(t, node2, tree.tail)
}

func TestRadixTree_LRU_Remove(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	node1, _ := tree.insertNode("foo/1", mockValue{10})
	tree.pushFront(node1)

	// Act
	tree.remove(node1)

	// Assert
	assert.Equal(t, 0, tree.len)
	assert.Nil(t, tree.head)
	assert.Nil(t, tree.tail)
}

func TestRadixTree_LRU_EvictOne(t *testing.T) {
	// Arrange
	tree := newRadixTree()
	node1, _ := tree.insertNode("foo/1", mockValue{10})
	node2, _ := tree.insertNode("foo/2", mockValue{20})
	tree.pushFront(node1)
	tree.pushFront(node2)

	// Act
	evictedValue := tree.evictOne()

	// Assert
	assert.Equal(t, 1, tree.len)
	assert.Equal(t, mockValue{10}, evictedValue) // node1 was tail (least recently used)
	assert.Equal(t, node2, tree.head)
	assert.Equal(t, node2, tree.tail)
}

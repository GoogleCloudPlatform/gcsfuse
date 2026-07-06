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

func TestRadixTree_InsertAndGet(t *testing.T) {
	tree := newRadixTree()
	val1 := mockValue{10}
	val2 := mockValue{20}

	// Test Insert
	node1, isNew := tree.insertNode("foo/bar", val1)
	assert.True(t, isNew)
	assert.NotNil(t, node1)
	assert.Equal(t, 1, tree.size)

	// Test Get
	gotNode, ok := tree.getNode("foo/bar")
	assert.True(t, ok)
	assert.Equal(t, val1, gotNode.value)

	// Test Overwrite (should not increase size)
	node2, isNew2 := tree.insertNode("foo/bar", val2)
	assert.False(t, isNew2)
	assert.Equal(t, val2, node2.value)
	assert.Equal(t, 1, tree.size)

	// Test non-existent get
	_, ok = tree.getNode("foo/baz")
	assert.False(t, ok)
}

func TestRadixTree_PrefixSplitting(t *testing.T) {
	tree := newRadixTree()

	tree.insertNode("foo/bar", mockValue{1})
	tree.insertNode("foo/baz", mockValue{2}) // Splits "foo/ba" -> "r", "z"

	assert.Equal(t, 2, tree.size)

	n1, _ := tree.getNode("foo/bar")
	n2, _ := tree.getNode("foo/baz")

	// Both should share the same parent routing node "foo/ba"
	assert.Equal(t, n1.parent, n2.parent)
	assert.Equal(t, "foo/ba", n1.parent.prefix)
	assert.Nil(t, n1.parent.value) // Parent is just a routing node
}

func TestRadixTree_DeleteAndCompress(t *testing.T) {
	tree := newRadixTree()

	tree.insertNode("foo/bar", mockValue{1})
	tree.insertNode("foo/baz", mockValue{2})

	n1, _ := tree.getNode("foo/bar")
	tree.deleteNode(n1)

	assert.Equal(t, 1, tree.size)
	_, ok := tree.getNode("foo/bar")
	assert.False(t, ok)

	// After deleting "foo/bar", the routing node "foo/ba" should compress back
	// into the surviving leaf "foo/baz", turning it back into "foo/baz".
	n2, _ := tree.getNode("foo/baz")
	assert.Equal(t, "foo/baz", n2.prefix)
	assert.Equal(t, tree.root, n2.parent) // Path compressed all the way up to the root!
}

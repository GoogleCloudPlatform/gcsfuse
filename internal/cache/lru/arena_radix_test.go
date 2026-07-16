// Copyright 2026 Google LLC
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

func setupArenaRadix() *arenaRadix {
	c := &arenaRadix{
		freeHead: nilNode,
	}
	c.root = c.allocateNode()
	return c
}

func TestArenaRadix_AllocateAndFree(t *testing.T) {
	c := setupArenaRadix()

	// Initial root node is allocated
	assert.Equal(t, 1, len(c.nodes))
	assert.Equal(t, nilNode, c.freeHead)

	// Allocate a new node
	id1 := c.allocateNode()
	assert.Equal(t, uint32(1), id1)
	assert.Equal(t, 2, len(c.nodes))

	// Free the node
	c.freeNode(id1)
	assert.Equal(t, id1, c.freeHead)

	// Allocate again, should reuse
	id2 := c.allocateNode()
	assert.Equal(t, id1, id2)
	assert.Equal(t, nilNode, c.freeHead)
}

func TestArenaRadix_ChildOperations(t *testing.T) {
	c := setupArenaRadix()

	// Create nodes
	idA := c.allocateNode()
	c.nodes[idA].prefix = "apple"

	idB := c.allocateNode()
	c.nodes[idB].prefix = "banana"

	idC := c.allocateNode()
	c.nodes[idC].prefix = "cherry"

	// Add children out of order, they should be sorted by prefix[0]
	c.addChild(c.root, idC)
	c.addChild(c.root, idA)
	c.addChild(c.root, idB)

	// Verify order: apple (A) -> banana (B) -> cherry (C)
	child1 := c.nodes[c.root].child
	assert.Equal(t, idA, child1)

	child2 := c.nodes[child1].sibling
	assert.Equal(t, idB, child2)

	child3 := c.nodes[child2].sibling
	assert.Equal(t, idC, child3)

	assert.Equal(t, nilNode, c.nodes[child3].sibling)

	// Verify getChild
	assert.Equal(t, idA, c.getChild(c.root, 'a'))
	assert.Equal(t, idB, c.getChild(c.root, 'b'))
	assert.Equal(t, idC, c.getChild(c.root, 'c'))
	assert.Equal(t, nilNode, c.getChild(c.root, 'z'))

	// Replace B with new node D ("berry")
	idD := c.allocateNode()
	c.nodes[idD].prefix = "berry"
	c.replaceChild(c.root, idB, idD)

	// verify old node is detached
	assert.Equal(t, nilNode, c.nodes[idB].sibling)
	assert.Equal(t, nilNode, c.nodes[idB].parent)

	// verify new node is linked correctly
	assert.Equal(t, idD, c.nodes[idA].sibling)
	assert.Equal(t, idC, c.nodes[idD].sibling)

	// Remove A
	c.removeChild(c.root, idA)
	assert.Equal(t, nilNode, c.nodes[idA].sibling)
	assert.Equal(t, nilNode, c.nodes[idA].parent)

	assert.Equal(t, idD, c.nodes[c.root].child)
}

func TestArenaRadix_HashString(t *testing.T) {
	hash1 := hashString("hello")
	hash2 := hashString("hello")
	assert.Equal(t, hash1, hash2)

	hash3 := hashString("world")
	assert.NotEqual(t, hash1, hash3)

	hashEmpty := hashString("")
	assert.NotEqual(t, uint64(0), hashEmpty)
}

func TestArenaRadix_GetFullKey(t *testing.T) {
	c := setupArenaRadix()

	id1 := c.allocateNode()
	c.nodes[id1].prefix = "a/"
	c.nodes[id1].parent = c.root

	id2 := c.allocateNode()
	c.nodes[id2].prefix = "b/"
	c.nodes[id2].parent = id1

	id3 := c.allocateNode()
	c.nodes[id3].prefix = "c.txt"
	c.nodes[id3].parent = id2

	assert.Equal(t, "a/b/c.txt", c.getFullKey(id3))
	assert.Equal(t, "a/", c.getFullKey(id1))
	assert.Equal(t, "", c.getFullKey(c.root))
}

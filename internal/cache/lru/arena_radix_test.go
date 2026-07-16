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

func setupArenaRadixChildren() (*arenaRadix, uint32, uint32, uint32) {
	c := setupArenaRadix()

	idA := c.allocateNode()
	c.nodes[idA].prefix = "apple"

	idB := c.allocateNode()
	c.nodes[idB].prefix = "banana"

	idC := c.allocateNode()
	c.nodes[idC].prefix = "cherry"

	// Add out of order to ensure sorting logic executes
	c.addChild(c.root, idC)
	c.addChild(c.root, idA)
	c.addChild(c.root, idB)

	return c, idA, idB, idC
}

func TestArenaRadix_AddChild(t *testing.T) {
	c, idA, idB, idC := setupArenaRadixChildren()

	// Verify order: apple (A) -> banana (B) -> cherry (C)
	child1 := c.nodes[c.root].child
	assert.Equal(t, idA, child1)

	child2 := c.nodes[child1].sibling
	assert.Equal(t, idB, child2)

	child3 := c.nodes[child2].sibling
	assert.Equal(t, idC, child3)

	assert.Equal(t, nilNode, c.nodes[child3].sibling)
	assert.Equal(t, c.root, c.nodes[idA].parent)
	assert.Equal(t, c.root, c.nodes[idB].parent)
	assert.Equal(t, c.root, c.nodes[idC].parent)
}

func TestArenaRadix_GetChild(t *testing.T) {
	c, idA, idB, idC := setupArenaRadixChildren()

	assert.Equal(t, idA, c.getChild(c.root, 'a'))
	assert.Equal(t, idB, c.getChild(c.root, 'b'))
	assert.Equal(t, idC, c.getChild(c.root, 'c'))
	assert.Equal(t, nilNode, c.getChild(c.root, 'z'))
}

func TestArenaRadix_ReplaceChild(t *testing.T) {
	c, idA, idB, idC := setupArenaRadixChildren()

	// Replace B with new node D ("berry")
	idD := c.allocateNode()
	c.nodes[idD].prefix = "berry"
	c.replaceChild(c.root, idB, idD)

	// verify old node is completely detached
	assert.Equal(t, nilNode, c.nodes[idB].sibling)
	assert.Equal(t, nilNode, c.nodes[idB].parent)

	// verify new node is linked correctly in the middle of A and C
	assert.Equal(t, idD, c.nodes[idA].sibling)
	assert.Equal(t, idC, c.nodes[idD].sibling)
}

func TestArenaRadix_RemoveChild(t *testing.T) {
	c, idA, idB, idC := setupArenaRadixChildren()

	// Remove A (the first child)
	c.removeChild(c.root, idA)
	assert.Equal(t, nilNode, c.nodes[idA].sibling)
	assert.Equal(t, nilNode, c.nodes[idA].parent)
	assert.Equal(t, idB, c.nodes[c.root].child)

	// Remove C (the last child)
	c.removeChild(c.root, idC)
	assert.Equal(t, nilNode, c.nodes[idC].sibling)
	assert.Equal(t, nilNode, c.nodes[idC].parent)
	assert.Equal(t, nilNode, c.nodes[idB].sibling) // B is now the last and only child
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

type testValue struct {
	size uint64
}

func (tv testValue) Size() uint64 { return tv.size }

func setupArenaRadixLRU() (*arenaRadix, uint32, uint32, uint32) {
	c := setupArenaRadix()
	c.head = nilNode
	c.tail = nilNode
	c.len = 0
	c.nodeMap = make(map[uint64]uint32)

	id1 := c.allocateNode()
	c.nodes[id1].value = testValue{size: 10}
	c.nodes[id1].prefix = "node1"
	c.addChild(c.root, id1)

	id2 := c.allocateNode()
	c.nodes[id2].value = testValue{size: 20}
	c.nodes[id2].prefix = "node2"
	c.addChild(c.root, id2)

	id3 := c.allocateNode()
	c.nodes[id3].value = testValue{size: 30}
	c.nodes[id3].prefix = "node3"
	c.addChild(c.root, id3)

	return c, id1, id2, id3
}

func TestArenaRadix_PushFront(t *testing.T) {
	c, id1, id2, _ := setupArenaRadixLRU()

	c.pushFront(id1)
	assert.Equal(t, id1, c.head)
	assert.Equal(t, id1, c.tail)
	assert.Equal(t, 1, c.len)

	c.pushFront(id2)
	assert.Equal(t, id2, c.head)
	assert.Equal(t, id1, c.tail)
	assert.Equal(t, id1, c.nodes[id2].next)
	assert.Equal(t, id2, c.nodes[id1].prev)
	assert.Equal(t, 2, c.len)
}

func TestArenaRadix_MoveToFront(t *testing.T) {
	c, id1, id2, id3 := setupArenaRadixLRU()
	c.pushFront(id1)
	c.pushFront(id2)
	c.pushFront(id3)
	// Order: id3 <-> id2 <-> id1

	c.moveToFront(id2) // Move from middle
	assert.Equal(t, id2, c.head)
	assert.Equal(t, id1, c.tail)
	assert.Equal(t, id3, c.nodes[id2].next)
	// Order: id2 <-> id3 <-> id1

	c.moveToFront(id1) // Move from tail
	assert.Equal(t, id1, c.head)
	assert.Equal(t, id3, c.tail)
	assert.Equal(t, id2, c.nodes[id1].next)
	// Order: id1 <-> id2 <-> id3

	c.moveToFront(id1) // Move already at head
	assert.Equal(t, id1, c.head)
}

func TestArenaRadix_Remove(t *testing.T) {
	c, id1, id2, id3 := setupArenaRadixLRU()
	c.pushFront(id1)
	c.pushFront(id2)
	c.pushFront(id3)
	// Order: id3 <-> id2 <-> id1

	c.remove(id2) // Remove from middle
	assert.Equal(t, 2, c.len)
	assert.Equal(t, id1, c.nodes[id3].next)
	assert.Equal(t, id3, c.nodes[id1].prev)
	assert.Equal(t, nilNode, c.nodes[id2].next)
	assert.Equal(t, nilNode, c.nodes[id2].prev)

	c.remove(id3) // Remove from head
	assert.Equal(t, id1, c.head)
	assert.Equal(t, 1, c.len)

	c.remove(id1) // Remove from tail
	assert.Equal(t, nilNode, c.head)
	assert.Equal(t, nilNode, c.tail)
	assert.Equal(t, 0, c.len)
}

func TestArenaRadix_EvictOne(t *testing.T) {
	c, id1, id2, id3 := setupArenaRadixLRU()
	c.pushFront(id1)
	c.pushFront(id2)
	c.pushFront(id3)
	// Order: id3 <-> id2 <-> id1 (id1 is tail)

	c.currentSize = 60
	c.nodeMap[hashString("node1")] = id1

	val := c.evictOne()
	assert.Equal(t, uint64(10), val.Size())

	// id1 should be removed
	assert.Equal(t, 2, c.len)
	assert.Equal(t, id2, c.tail)
	assert.Equal(t, uint64(50), c.currentSize)

	// verify id1 is deleted from tree and nodeMap
	_, exists := c.nodeMap[hashString("node1")]
	assert.False(t, exists)
	assert.Nil(t, c.nodes[id1].value)
}

func TestArenaRadix_InsertNode(t *testing.T) {
	c := setupArenaRadix()
	c.nodeMap = make(map[uint64]uint32)

	// Insert "apple"
	id1, old1 := c.insertNode("apple", testValue{size: 10})
	assert.Nil(t, old1)
	assert.Equal(t, testValue{size: 10}, c.nodes[id1].value)

	// Insert "app" (causes a split in "apple")
	id2, old2 := c.insertNode("app", testValue{size: 5})
	assert.Nil(t, old2)
	assert.Equal(t, testValue{size: 5}, c.nodes[id2].value)

	// Update "app"
	id3, old3 := c.insertNode("app", testValue{size: 15})
	assert.Equal(t, id2, id3)
	assert.Equal(t, testValue{size: 5}, old3)
	assert.Equal(t, testValue{size: 15}, c.nodes[id2].value)
}

func TestArenaRadix_GetNodeKey(t *testing.T) {
	c := setupArenaRadix()
	c.nodeMap = make(map[uint64]uint32)

	id1, _ := c.insertNode("a/b/c", testValue{size: 10})
	c.nodeMap[hashString("a/b/c")] = id1

	id2, _ := c.insertNode("a/b/d", testValue{size: 20})
	c.nodeMap[hashString("a/b/d")] = id2

	// Test getNodeKey
	foundId, ok := c.getNodeKey("a/b/c")
	assert.True(t, ok)
	assert.Equal(t, id1, foundId)

	foundId, ok = c.getNodeKey("a/b/d")
	assert.True(t, ok)
	assert.Equal(t, id2, foundId)

	_, ok = c.getNodeKey("a/b/e")
	assert.False(t, ok)

	// Test verifyKey directly
	assert.True(t, c.verifyKey(id1, "a/b/c"))
	assert.False(t, c.verifyKey(id1, "a/b/d"))
	assert.False(t, c.verifyKey(id1, "a/b"))
}

func TestArenaRadix_DeleteNodeAndCompress(t *testing.T) {
	c := setupArenaRadix()

	// Insert "a/b/c.txt"
	id1, _ := c.insertNode("a/b/c.txt", testValue{size: 10})

	// Insert "a/b/d.txt" to force a split
	// Tree becomes: root -> "a/b/" (routing node) -> "c.txt", "d.txt"
	id2, _ := c.insertNode("a/b/d.txt", testValue{size: 20})

	// Delete "c.txt"
	c.deleteNode(id1)

	// "a/b/" now only has one child ("d.txt").
	// compressPathUpwards should automatically merge "a/b/" and "d.txt" into "a/b/d.txt"
	// id2 should now be a direct child of root with the full prefix!
	child1 := c.nodes[c.root].child
	assert.Equal(t, id2, child1)
	assert.Equal(t, "a/b/d.txt", c.nodes[id2].prefix)
	assert.Equal(t, c.root, c.nodes[id2].parent)
	assert.Equal(t, nilNode, c.nodes[id2].sibling)
}

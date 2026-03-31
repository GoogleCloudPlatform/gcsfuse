package lru

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringVal struct {
	v string
}

func (s stringVal) Size() uint64 {
	return uint64(len(s.v))
}

func TestRadixTree_InsertAndGet(t *testing.T) {
	tree := newRadixTree()

	// Insert "abc"
	node, isNew := tree.Insert("abc", stringVal{"val1"})
	require.True(t, isNew)
	assert.Equal(t, "abc", node.prefix)
	assert.Equal(t, stringVal{"val1"}, node.value)
	assert.Equal(t, 1, tree.Len())

	// Get "abc"
	n, ok := tree.Get("abc")
	require.True(t, ok)
	assert.Equal(t, "abc", n.prefix)
	assert.Equal(t, stringVal{"val1"}, n.value)

	// Insert "abx" (Splits "abc" -> "ab" -> "c", "x")
	node, isNew = tree.Insert("abx", stringVal{"val2"})
	require.True(t, isNew)
	assert.Equal(t, "x", node.prefix) // leaf node prefix
	assert.Equal(t, 2, tree.Len())

	// Get "abx"
	n, ok = tree.Get("abx")
	require.True(t, ok)
	assert.Equal(t, "x", n.prefix)
	assert.Equal(t, stringVal{"val2"}, n.value)

	// Get "abc" again to ensure it survived the split
	n, ok = tree.Get("abc")
	require.True(t, ok)
	assert.Equal(t, "c", n.prefix)
	assert.Equal(t, stringVal{"val1"}, n.value)

	// Insert "ab" (Ends exactly at the split node)
	node, isNew = tree.Insert("ab", stringVal{"val3"})
	require.True(t, isNew)
	assert.Equal(t, "ab", node.prefix) // Internal node now has a value
	assert.Equal(t, 3, tree.Len())

	n, ok = tree.Get("ab")
	require.True(t, ok)
	assert.Equal(t, "ab", n.prefix)
	assert.Equal(t, stringVal{"val3"}, n.value)

	// Update existing "ab"
	node, isNew = tree.Insert("ab", stringVal{"val4"})
	require.False(t, isNew)
	assert.Equal(t, "ab", node.prefix)
	assert.Equal(t, 3, tree.Len())

	n, ok = tree.Get("ab")
	require.True(t, ok)
	assert.Equal(t, stringVal{"val4"}, n.value)

	// Non-existent get
	_, ok = tree.Get("a")
	assert.False(t, ok)
	_, ok = tree.Get("abcd")
	assert.False(t, ok)
}

func TestRadixTree_DeleteNode_PruneAndMerge(t *testing.T) {
	tree := newRadixTree()

	// Build a tree with branches
	//        "" (root)
	//         |
	//        "a"
	//       /   \
	//     "b"   "c"
	//    /   \
	//  "d"   "e"

	tree.Insert("abd", stringVal{"abd"})
	tree.Insert("abe", stringVal{"abe"})
	tree.Insert("ac", stringVal{"ac"})

	assert.Equal(t, 3, tree.Len())

	// Delete "abe". Node "ab" should merge with "d" to become "abd"
	n, ok := tree.Get("abe")
	require.True(t, ok)
	tree.DeleteNode(n)

	assert.Equal(t, 2, tree.Len())
	_, ok = tree.Get("abe")
	assert.False(t, ok)

	// Let's verify "abd" is still there and merged
	n, ok = tree.Get("abd")
	require.True(t, ok)
	assert.Equal(t, "bd", n.prefix) // "b" + "d" = "bd"
	assert.Equal(t, stringVal{"abd"}, n.value)

	// Delete "abd". "a" should merge with "c" to become "ac"
	n, ok = tree.Get("abd")
	require.True(t, ok)
	tree.DeleteNode(n)

	assert.Equal(t, 1, tree.Len())
	_, ok = tree.Get("abd")
	assert.False(t, ok)

	n, ok = tree.Get("ac")
	require.True(t, ok)
	assert.Equal(t, "ac", n.prefix) // "a" + "c" = "ac"

	// Delete "ac". Tree should be completely empty (except root)
	n, ok = tree.Get("ac")
	require.True(t, ok)
	tree.DeleteNode(n)

	assert.Equal(t, 0, tree.Len())
	assert.Nil(t, tree.root.child)
}

func TestRadixTree_WalkPrefix(t *testing.T) {
	tree := newRadixTree()
	tree.Insert("a/b/c", stringVal{"1"})
	tree.Insert("a/b/d", stringVal{"2"})
	tree.Insert("a/x", stringVal{"3"})
	tree.Insert("b/y", stringVal{"4"})

	var visited []stringVal
	walkFn := func(n *radixNode) bool {
		visited = append(visited, n.value.(stringVal))
		return false
	}

	// Exact match with internal node that splits into children
	tree.WalkPrefix("a/b/", walkFn)
	assert.Len(t, visited, 2)
	assert.Contains(t, visited, stringVal{"1"})
	assert.Contains(t, visited, stringVal{"2"})

	// Partial match in the middle of a node's prefix
	visited = nil
	tree.WalkPrefix("a/", walkFn)
	assert.Len(t, visited, 3)
	assert.Contains(t, visited, stringVal{"1"})
	assert.Contains(t, visited, stringVal{"2"})
	assert.Contains(t, visited, stringVal{"3"})

	// Prefix completely misses
	visited = nil
	tree.WalkPrefix("z", walkFn)
	assert.Empty(t, visited)

	// Prefix is longer than existing string but diverges
	visited = nil
	tree.WalkPrefix("a/b/c/d", walkFn)
	assert.Empty(t, visited)

	// Prefix is exact match for a leaf node
	visited = nil
	tree.WalkPrefix("a/x", walkFn)
	assert.Len(t, visited, 1)
	assert.Contains(t, visited, stringVal{"3"})

	// Walk from root
	visited = nil
	tree.WalkPrefix("", walkFn)
	assert.Len(t, visited, 4)
}

func TestRadixTree_AddChildOrdering(t *testing.T) {
	tree := newRadixTree()

	// Insert in reverse alphabetical order to ensure they get sorted by byte
	tree.Insert("z", stringVal{"z"})
	tree.Insert("c", stringVal{"c"})
	tree.Insert("a", stringVal{"a"})
	tree.Insert("y", stringVal{"y"})

	var visited []stringVal
	tree.WalkPrefix("", func(n *radixNode) bool {
		visited = append(visited, n.value.(stringVal))
		return false
	})

	assert.Equal(t, []stringVal{{"a"}, {"c"}, {"y"}, {"z"}}, visited)
}

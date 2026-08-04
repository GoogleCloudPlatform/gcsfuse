# Arena-based Radix Cache (LRU)

This document provides a comprehensive overview of the **Arena-based Radix Cache** implemented in the `internal/cache/lru` package of GCSFuse (`radix_arena.go` and `radix_arena_lru.go`). It is designed to help new contributors understand the memory optimizations and performance benefits of this implementation compared to the standard pointer-based radix cache.

## Overview

The `arenaRadix` is an advanced LRU (Least Recently Used) cache backed by a Radix Tree. While it logically behaves exactly like the pointer-based radix tree (using a Left-Child Right-Sibling structure), it is physically stored in a contiguous array (an **Arena**) instead of dynamically allocated heap objects. 

This design completely eliminates pointer chasing, vastly reduces Go garbage collection (GC) overhead, improves CPU cache locality, and uses a secondary Hash Map to achieve **O(1) lookups** while preserving the prefix-deletion benefits of a Radix Tree.

## Core Architecture & Optimizations

### 1. Arena (Slice-backed) Allocation

Instead of allocating memory on the heap for every tree node using pointers (`*arenaRadixNode`), all nodes are stored in a pre-allocated slice: `nodes []arenaRadixNode`. 

Links between nodes are represented by `uint32` indices rather than 64-bit pointers. The constant `nilNode` (`math.MaxUint32`) is used to represent a null reference.

**Benefits:**
- **Zero Heap Allocations per node**: Node creation simply involves appending to the slice or reusing a freed index.
- **Smaller Node Size**: `uint32` indices take half the space of 64-bit pointers.
- **Cache Locality**: Nodes are packed tightly in contiguous memory, minimizing CPU cache misses during tree traversal.

### 2. Node Structure (`arenaRadixNode`)

```go
type arenaRadixNode struct {
	prefix  string
	value   ValueType
	
	// Tree relationships (indices in the `nodes` slice)
	parent  uint32
	child   uint32
	sibling uint32
	
	// LRU Linked list (indices in the `nodes` slice)
	prev    uint32
	next    uint32
}
```

### 3. Fast Lookups with `nodeMap`

A standard Radix Tree requires traversing down the tree character-by-character for a lookup. The Arena Radix Cache optimizes this by maintaining a secondary index:
`nodeMap map[uint64]uint32`

- **Hashing**: The full string key is hashed using an extremely fast, allocation-free **FNV-1a 64-bit** hash (`hashString`).
- **O(1) Retrieval**: The hash is used to instantly look up the `uint32` node index from the `nodeMap`.
- **Collision Safety (`verifyKey`)**: Because hash collisions are possible, `verifyKey` takes the retrieved node and walks *upwards* to the root using the `parent` index, comparing the key backwards to perfectly verify it against the search key. This upward traversal is allocation-free.

### 4. Memory Reuse (The Free List)

When a node is deleted (e.g., during eviction or prefix deletion), we cannot easily shrink the `nodes` slice. Instead, the Arena uses a **Free List**.

- The `arenaRadix` struct maintains a `freeHead uint32` index.
- When `freeNode` is called, the node's fields are zeroed out, and its `next` index is pointed to the current `freeHead`. `freeHead` is then updated to this node's index.
- When `allocateNode` is called, it pops from `freeHead`. If `freeHead` is empty (`nilNode`), it appends a new node to the `nodes` slice.

## Key Operations

### Tree and LRU Operations

The logical tree operations (Insertion, Splitting, Deletion) and LRU operations (MoveToFront, Evict) are structurally identical to the pointer-based radix tree, but strictly manipulate `uint32` indices instead of pointers.
- **Insertion**: May create new nodes from the free list, manipulating `child` and `sibling` indices. It inserts the node ID into the `nodeMap` once constructed.
- **Eviction/Erasure**: Removes the node from the `nodes` array by placing it in the free list, unlinks it from the LRU via `prev`/`next` indices, and deletes the hash from `nodeMap`.

### Prefix Deletion (`EraseEntriesWithGivenPrefix`)

This cache achieves the best of both worlds:
1. Lookups are O(1) thanks to the `nodeMap`.
2. Prefix deletions are highly efficient because the hierarchical tree structure is maintained.

When deleting by prefix, the tree is traversed downwards to find the boundary node. The subtree is detached by removing its link from its parent's sibling chain. Finally, we call `freeSubtree` which iterates through the detached subtree, removing each child's hash from `nodeMap` and returning the node indices back to the free list.

## Invariant Checking

Like the pointer radix, `checkInvariants()` ensures the tree hierarchy (measured via traversal) perfectly aligns with the elements physically present in the LRU linked list. Due to the array structure, these traversals happen entirely via index jumps across the contiguous memory slice, preventing panic-inducing nil-pointer dereferences common in corrupted pointer-based linked lists.


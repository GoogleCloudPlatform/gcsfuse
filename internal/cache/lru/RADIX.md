# Pointer-based Radix Cache (LRU)

This document provides a comprehensive overview of the **Pointer-based Radix Cache** implemented in the `internal/cache/lru` package of GCSFuse (`radix.go`). It is designed to help new contributors understand the architecture, data structures, and algorithms used in this cache implementation.

## Overview

The `radixCache` is an implementation of an LRU (Least Recently Used) cache backed by a custom Radix Tree. It is specifically designed to efficiently store and retrieve prefix-based string keys while maintaining a strict memory bound (`maxSize`). 

By combining a radix tree with a doubly linked list, the cache achieves fast prefix lookups, insertions, and deletions, while efficiently evicting the least recently used entries.

## Core Data Structures

### 1. Left-Child Right-Sibling (LCRS) Representation

Standard radix trees or tries often use an array or a hash map to store children (e.g., `map[byte]*Node` or `[256]*Node`). This approach incurs heavy memory overhead and frequent slice allocations. 

To avoid these allocations, this implementation uses the **Left-Child Right-Sibling (LCRS)** representation.
Every node has exactly three tree-related pointers:
- `parent`: Pointer to the parent node.
- `child`: Pointer to the **first** child node.
- `sibling`: Pointer to the **next** sibling node.

In this structure, the children of a node are represented as a linked list. To find a specific child, the tree iterates through the `child` and its `sibling` chain. The siblings are maintained in lexicographical order (sorted by the first byte of their prefix), allowing for early exits during search.

### 2. `radixNode` Struct

The `radixNode` structure encapsulates both the tree hierarchy and the LRU doubly linked list:

```go
type radixNode struct {
	prefix  string      // The string segment matched at this node
	value   ValueType   // The cached value (nil for routing nodes)
	
	// Tree pointers
	parent  *radixNode
	child   *radixNode
	sibling *radixNode
	
	// LRU Doubly Linked List pointers
	prev *radixNode
	next *radixNode
}
```

- **Routing Nodes vs. Value Nodes**: A node may just be a "routing node" holding a common prefix for its children, in which case its `value` is `nil`. If it holds a cache entry, `value` is non-nil.

### 3. `radixCache` Struct

The `radixCache` struct holds the state of the cache:

```go
type radixCache struct {
	maxSize     uint64       // Maximum allowed size of all values combined
	currentSize uint64       // Current sum of all value sizes
	
	root        *radixNode   // Root of the radix tree
	
	head        *radixNode   // MRU (Most Recently Used) end of the list
	tail        *radixNode   // LRU (Least Recently Used) end of the list
	len         int          // Number of elements in the LRU list
	
	mu          locker.RWLocker // Read-Write lock for concurrency safety
}
```

## Key Operations

### Tree Operations

1.  **Insertion (`insertNode`)**:
    When inserting a key, the tree is traversed character by character. 
    - If a matching prefix is partially found, the existing node is **split** into two nodes. The common prefix becomes the parent, and the remaining parts become children.
    - If no child matches the next byte, a new leaf node is created and added to the sibling list.
2.  **Lookup (`getNode`)**:
    Traverses the tree by matching the search key with node prefixes. Because siblings are ordered lexicographically, the search is highly efficient.
3.  **Deletion & Compression (`compressPathUpwards`)**:
    When an entry is removed, its value is set to `nil`. If the node becomes a leaf (no children) and has no value, it is removed. To prevent the tree from becoming deep and sparse, `compressPathUpwards` walks up the tree. If it finds a routing node with only one child, it merges the node with its child, concatenating their prefixes.

### LRU Operations

The LRU cache is managed by updating the `prev` and `next` pointers of `radixNode` when values are accessed or inserted.
-   **`pushFront`**: Inserts a new node at the MRU end (`head`).
-   **`moveToFront`**: Moves an accessed node to the MRU end.
-   **`evictOne`**: Removes the node at the LRU end (`tail`) from both the linked list and the tree.

### Prefix Deletion (`EraseEntriesWithGivenPrefix`)

One of the main benefits of using a Radix Tree over a flat Hash Map is the ability to efficiently delete all entries sharing a specific prefix. 
Instead of iterating through all keys, the cache traverses down to the node representing the prefix. It severs that subtree entirely and recursively sweeps it to free memory, update `currentSize`, and unlink the nodes from the LRU list.

## Invariant Checking

For robust concurrency and memory safety, the cache maintains strict invariants verified by `checkInvariants()`:
- `currentSize` must never exceed `maxSize`.
- Every element in the LRU list must have a non-nil value.
- The number of value-bearing nodes in the tree must exactly match the length of the LRU list.

These invariants ensure the tree structure and the LRU linked list are perfectly synchronized.


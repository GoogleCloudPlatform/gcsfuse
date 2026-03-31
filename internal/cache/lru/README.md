# LRU Cache Implementations

This package (`internal/cache/lru`) provides two variants of a generic LRU (Least Recently Used) Cache implementation, indexed by string keys, for caching `lru.ValueType` interfaces. Both variants provide `O(1)` or near `O(1)` performance for inserts, lookups, and deletions.

The choice of cache implementation depends on the workload. The `mapCache` is simple and map-backed, while the `radixCache` is optimized for prefix-based deletions and zero allocation overhead during tree manipulation.

## Interface

The core interaction is defined by the `lru.Cache` interface:

```go
type Cache interface {
	Insert(key string, value ValueType) ([]ValueType, error)
	Erase(key string) (value ValueType)
	LookUp(key string) (value ValueType)
	LookUpWithoutChangingOrder(key string) (value ValueType)
	UpdateWithoutChangingOrder(key string, value ValueType) error
	UpdateSize(key string, sizeDelta uint64) error
	EraseEntriesWithGivenPrefix(prefix string)
}
```

## `mapCache`

The `mapCache` (`lru.NewCache`) is the traditional map-backed LRU implementation. 
*   **LRU Tracking:** It utilizes a standard doubly-linked list (`container/list.List`) to keep track of the most recently used entries.
*   **Index:** A `map[string]*list.Element` acts as a fast `O(1)` lookup index for entries in the linked list.
*   **Prefix Deletions:** Because a standard Go map is un-ordered, finding entries to delete by a prefix (`EraseEntriesWithGivenPrefix`) requires a full table scan (`O(N)` keys), acquiring a read lock for the entire scan before deleting.

## `radixCache` (Custom Radix Tree)

The `radixCache` (`lru.NewRadixCache`) implementation addresses the map's performance bottleneck during prefix deletions (`EraseEntriesWithGivenPrefix`), which is a common operation in GCSFuse (e.g., recursive directory deletions).

It utilizes a highly optimized, zero-allocation custom Radix Tree (compressed trie).

### Why a Custom Implementation?

Standard Radix Tree implementations (like `github.com/armon/go-radix`) are optimized for lookups but often incur significant heap allocation overhead during updates:
*   They typically use slices to store child pointers. When a node splits, or a new child is added, slices must be reallocated.
*   They require completely separate nodes for the LRU linked list.

### Key Optimizations

The `radixCache` incorporates two major architectural optimizations to minimize heap allocations and memory footprint.

#### 1. Zero-Allocation Tree Structure (Left-Child Right-Sibling)

Instead of using a slice `[]*radixNode` to store children, the `radixNode` uses a Left-Child Right-Sibling pointer structure.

```go
type radixNode struct {
	prefix string
	value  ValueType

	// Tree pointers
	parent  *radixNode
	child   *radixNode // Points to the FIRST child
	sibling *radixNode // Points to the NEXT sibling of this node
    // ...
}
```

This ensures that adding, removing, splitting, or merging nodes only involves updating pointers. No slices are ever allocated or resized. The sibling list is kept sorted alphabetically by the first byte of each prefix.

#### 2. Embedded LRU Pointers

The `radixNode` *is* the LRU linked-list element.

```go
type radixNode struct {
	// ... tree pointers
	
	// LRU Linked List pointers
	prev *radixNode
	next *radixNode
}
```

When a node is created in the radix tree, it is simultaneously linked into the LRU doubly-linked list (`c.head`, `c.tail`). This completely eliminates the need for separate `container/list.Element` allocations wrapper objects.

### Radix Tree Operations

#### Insertions & Node Splitting

When a key is inserted, the tree is traversed character by character. If a key partially matches an existing node's prefix, the existing node is **split**.

Example: Inserting `"abx"` when `"abc"` exists:
1.  Original Node: `[prefix="abc"]`
2.  Longest common prefix is `"ab"`.
3.  A new split node `[prefix="ab"]` is created.
4.  The original node's prefix is shortened to `[prefix="c"]`, and it becomes a child of the split node.
5.  A new leaf node `[prefix="x"]` is created as the second child of the split node.

#### Deletions & Node Merging

Because nodes contain parent pointers, deleting a node (`DeleteNode`) takes `O(1)` time after the node is located via the LRU tail pointer or a direct lookup.

To keep the tree compressed, the tree self-optimizes upon deletions.
1.  **Pruning:** If a deleted node has no children, it is removed. The tree then walks upward via parent pointers, deleting any empty internal routing nodes.
2.  **Merging:** If an internal node has exactly one child, and the internal node itself does not hold a value (it is purely a routing node), it is **merged** with its single child. The parent's prefix is prepended to the child's prefix, and the child replaces the parent in the tree hierarchy.
    *   *Note: Merging replaces the parent with the child rather than mutating the parent in-place. This ensures the child node object, which is permanently referenced by the embedded LRU list pointers, is never orphaned.*

#### Prefix Traversal

`WalkPrefix` is highly efficient. It walks down the tree matching the prefix exactly. Once the prefix is fully consumed, it performs a DFS traversal of all descendants, avoiding a full cache scan.

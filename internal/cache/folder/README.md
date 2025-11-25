# Trie Data Structure

## Overview

This document describes a specialized, thread-safe prefix tree (trie) implementation in Go. It is designed for managing hierarchical data, such as file system paths, in memory. The key features include:

- **Concurrency**: Fine-grained locking at the node level allows for high-concurrency reads, writes, and structural modifications.
- **Memory Efficiency**: Lazy initialization of child maps and automatic pruning of empty branches reduce the memory footprint.
- **File System Semantics**: Provides methods that mimic file system operations like `Insert`, `Get`, `Delete`, `Move`, and `ListPathsWithPrefix`.
- **Automatic Eviction**: An optional two-phase, TTL-based eviction mechanism helps manage the trie's size by removing least recently accessed files when capacity limits are reached.

## Architecture

### Core Components

- **`Trie`**: The main struct that holds the root of the tree and global properties, such as file counts and eviction settings.
- **`TrieNode`**: Represents a single node in the trie, corresponding to a path component (e.g., a directory or file name). Each node contains:
    - `mu *sync.RWMutex`: A mutex for protecting the node's immediate state (its children map and file info).
    - `children map[string]*TrieNode`: A map from a path component name to the child node. This is lazily initialized to save memory on leaf nodes.
    - `isLeaf bool`: A flag indicating if this node represents the end of a path for an inserted file.
    - `file *FileInfo`: A pointer to the file's metadata, present only if `isLeaf` is true.
- **`FileInfo`**: Stores metadata for a file, including:
    - `atime`, `mtime`, `ctime`: Access, modification, and creation timestamps.
    - `size`: The size of the file.
    - `data interface{}`: A generic field that can be used to cache file content or other associated data.

### Concurrency Model

The trie employs a fine-grained locking strategy to maximize concurrency. Instead of a single global lock, each `TrieNode` has its own `sync.RWMutex`.

Operations that traverse the trie (like `Insert`, `Get`, `Delete`) use a technique called **lock coupling** (or lock chaining). When moving from a parent node to a child node, the algorithm locks the child node *before* unlocking the parent node. This ensures that the path being traversed remains consistent and prevents race conditions where a concurrent operation might modify the structure of the trie.

This approach allows unrelated branches of the trie to be modified concurrently without contention. For example, one thread can write to `/a/b/c` while another writes to `/x/y/z` without blocking each other.

Global counters on the `Trie` struct, such as `leaf_counts` and `file_counts`, are protected by a separate `sync.RWMutex` on the `Trie` struct itself.

### Memory Management

The trie is designed to be memory-conscious:

1.  **Lazy Initialization**: A `TrieNode`'s `children` map is only created when the first child is added. This avoids allocating empty maps for the vast number of leaf nodes in a large file system tree.
2.  **Pruning**: When a file is deleted using the `Delete` method, the trie automatically prunes empty parent directories. If deleting a file causes its parent directory to become empty (i.e., it has no other children and is not a file itself), that parent is removed. This process continues recursively up the tree, reclaiming memory from unused branches.

## Automatic Eviction

The trie can be configured to automatically evict files based on access time when the total number of files exceeds a certain threshold. This is configured via `NewTrieWithTTL`.

### Configuration

- `hwm_file_count` (High Water Mark): When the total file count exceeds this value, the eviction process is triggered.
- `lwm_file_count` (Low Water Mark): The target file count that the eviction process aims to reach.
- `ttl_long`: A longer TTL (in minutes). Files not accessed within this duration are the first candidates for eviction.
- `ttl`: A shorter TTL (in minutes). If the file count is still above the `lwm_file_count` after the first phase, files not accessed within this shorter duration are evicted.

### Two-Phase Eviction Logic

When an `Insert` operation causes the file count to surpass the `hwm_file_count`, a background goroutine starts the eviction process:

1.  **Phase 1 (Long TTL)**: The trie is scanned, and all files with an access time (`atime`) older than `now - ttl_long` are removed.
2.  **Phase 2 (Short TTL)**: If, after Phase 1, the file count is still above the `lwm_file_count`, the trie is scanned again. This time, all files with an `atime` older than `now - ttl` are removed.

This two-tiered approach allows for a more graceful eviction, first removing very old files and only applying a more aggressive policy if necessary.

## Method Usage

### Initialization

**`NewTrie() *Trie`**

Creates a standard trie without automatic eviction.

```go
trie := folder.NewTrie()
```

**`NewTrieWithTTL(ttl int, ttl_long int, lwm_file_count int64, hwm_file_count int64) *Trie`**

Creates a trie with the two-phase TTL eviction mechanism enabled.

```go
trie := folder.NewTrieWithTTL(5, 60, 10000, 15000) // Short TTL: 5m, Long TTL: 60m, LWM: 10k, HWM: 15k
```

### Core Operations

**`Insert(path string, fileInfo *FileInfo)`**

Thread-safely inserts a file at the given path. It creates intermediate directory nodes as needed. If the path already exists, it overwrites the `FileInfo`.

**`Get(path string) (*FileInfo, bool)`**

Retrieves the `FileInfo` for a given path. It returns the data and `true` if the file exists, or `nil` and `false` otherwise. This operation updates the file's access time (`atime`), making it relevant for TTL eviction.

**`Delete(path string)`**

Removes a file and **prunes any parent directories that become empty** as a result. This is the standard way to remove a file and reclaim memory from its path.

**`DeleteFile(path string) (*FileInfo, bool)`**

Removes a file at a given path **without pruning** parent nodes. This is useful if you intend to keep the directory structure intact. It returns the `FileInfo` of the deleted file.

**`InsertDir(path string)`**

Ensures a directory path exists. It creates all necessary intermediate nodes but does not mark the final node as a leaf.

**`PathExists(path string) bool`**

Checks if a path exists in the trie, either as a file or an intermediate directory.

### Advanced Operations

**`Move(sourcePath, destPath string) bool`**

Moves a node (and its entire subtree) from a source path to a destination path. This operation is atomic and prunes the old path if it becomes empty. It returns `false` for invalid moves, such as moving a directory into itself or if the destination already exists.

**`ListPathsWithPrefix(prefix string) []string`**

Returns a slice of all full file paths that start with the given prefix.

**`InsertDirect(path *string, fileInfo *FileInfo)`**

A non-thread-safe version of `Insert`. It is intended for use cases where the trie is being populated by a single thread, such as during initial bulk loading, to avoid locking overhead. **Do not use in a concurrent context.**

### Utility Methods

**`CountFiles() int`**

Returns the total number of nodes that have non-nil `FileInfo`, representing actual files.

**`CountLeafs() int`**

Returns the total number of nodes marked as `isLeaf`. This count may temporarily differ from `CountFiles` during certain operations.


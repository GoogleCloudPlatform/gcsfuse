# Analysis Report: Read-Locking Optimizations for Inode Lookups in `gcsfuse`

## 1. Codebase Architecture & Location of Symbols

### LookUpInode
- **File**: `internal/fs/fs.go`
- **Line**: ~1986
- **Functionality**: FUSE operation handler for lookups. Resolves parent directory inode, calls `lookUpOrCreateChildInode` to obtain child inode, retrieves child attributes, and registers entry with FUSE.

### coreToDirentPlus
- **File**: `internal/fs/fs.go`
- **Line**: ~1774
- **Functionality**: Converts cached child `inode.Core` metadata into a FUSE directory entry structure (`fuseutil.DirentPlus`). Performs an inode lookup internally to fetch and refresh attributes.

### lookUpOrCreateInodeIfNotStale
- **File**: `internal/fs/fs.go`
- **Line**: ~1113
- **Functionality**: Looks up an existing inode or creates/mints a new one from a GCS object core. Handles generation comparisons and updates the file size if the remote generation is identical but size differs.

---

## 2. Inode Locking & `locker.RWLocker` Implementation

### `locker.RWLocker` Interface
- **File**: `internal/locker/rw_locker.go`
- **Definition**:
  ```go
  type RWLocker interface {
      sync.Locker
      RLock()
      RUnlock()
  }
  ```
  This is a reader-writer lock interface embedding standard `sync.Locker` (for write locking `Lock()` and `Unlock()`) and providing shared read locking `RLock()` and `RUnlock()`.

### Directory Inodes Implementation
- Directory inodes (both `dirInode` representing standard GCS directories, and `baseDirInode` representing the root bucket-level directory) implement the `inode.DirInode` interface, which inherits `locker.RWLocker` methods (`RLock`, `RUnlock`, `Lock`, `Unlock`).
- Internally, they hold a private `mu locker.RWLocker` backing field, instantiated via `locker.NewRW()`.
- Shared lock helper `LockForChildLookup()` takes:
  - Shared read lock (`mu.RLock()`) for child directories (`dirInode`), allowing concurrent lookups.
  - Exclusive write lock (`mu.Lock()`) for base directory (`baseDirInode`), since bucket index maps are modified during lookups.

### File Inodes Implementation
- File inodes (`FileInode` in `internal/fs/inode/file.go`) only implement standard exclusive `sync.Locker` locking (`Lock()` and `Unlock()`). They do not implement `locker.RWLocker` and do not support shared read locking. They lock state using a `syncutil.InvariantMutex`.

---

## 3. Locking Logic Analysis & Proposed Read-Locking Design

### Present Locking Issues & Read-Locking Goal
Currently, child lookups return inodes in an exclusively write-locked state (`Lock()`). During heavy concurrent lookup loops or directory listings, this results in serializing requests, causing lock contention and writer starvation.
The goal is to allow child inodes that support reader-writer locking (directories implementing `locker.RWLocker`) to be read-locked instead of write-locked during read-only lookup and attribute retrieval paths.

### 1. Thread-safe Lookup Counts
Because `in.IncrementLookupCount()` and `in.DecrementLookupCount()` are called when inodes are locked, if we only acquire a read lock, concurrent lookups will race on mutating `in.lc.count` field.
- **Proposed Solution**: Modify `lookupCount` in `internal/fs/inode/lookup_count.go` to use atomic primitives via the `"sync/atomic"` package for `count` updates and `destroyed` checks.

### 2. Parameterizing Inode Lookups
Introduce a `readLock bool` flag to propagate the caller's locking preference:
- `LookUpInode` (lookup path) and `coreToDirentPlus` (listing path) request `readLock = true`.
- Tree modification paths (e.g. `MkDir`, `createFile`, `CreateSymlink`, `RmDir`, `Rename`) request `readLock = false` to guarantee exclusive writer locking.

### 3. Read Lock Acquisition
In `lookUpOrCreateInodeIfNotStale`, check if the target inode implements `locker.RWLocker` (using dynamic type assertion) and if `readLock` is requested:
- If yes, acquire shared lock using `RLock()`.
- Else, fall back to exclusive lock using `Lock()`.

### 4. Lock Upgrading (Release Read, Acquire Write, Retry)
If a remote size mismatch is discovered under a read lock (`cmp == 2` in `lookUpOrCreateInodeIfNotStale`), we cannot update `inode.UpdateSize()` directly because modifying size state requires write privilege.
- **Proposed Solution**:
  1. Release the child's read lock (`RUnlock()`).
  2. Clear the `localReadLock` preference flag to `false`.
  3. Re-run the loop iteration, which will drop the filesystem lock `fs.mu`, acquire the child's exclusive write lock (`Lock()`), re-acquire `fs.mu`, and safely perform the size update.

This ensures lock ordering is fully respected (`child` lock acquired before filesystem `fs.mu` lock), preventing deadlock.

# Analysis: Read-Locking Optimizations for Inode Lookups

This analysis covers the locking structure, inode lookup paths, directory listing paths, and the proposed read-locking/lock-upgrading design.

## 1. Locations of Core Functions

- **`LookUpInode`**:
  Defined in `internal/fs/fs.go:1912` as:
  `func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) (err error)`
- **`coreToDirentPlus`**:
  Defined in `internal/fs/fs.go:1700` as:
  `func (fs *fileSystem) coreToDirentPlus(ctx context.Context, fullName inode.Name, core inode.Core, parInodeCtx context.Context) (entryPlus *fuseutil.DirentPlus, err error)`
- **`lookUpOrCreateInodeIfNotStale`**:
  Defined in `internal/fs/fs.go:1089` as:
  `func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(parInodeCtx context.Context, ic inode.Core) (in inode.Inode, err error)`

## 2. Inode Lock Implementation (`locker.RWLocker`)

- **`locker.RWLocker`**:
  Defined in `internal/locker/rw_locker.go:26`:
  ```go
  type RWLocker interface {
      sync.Locker
      RLock()
      RUnlock()
  }
  ```
  This is a standard reader-writer mutex interface that extends `sync.Locker` with `RLock()` and `RUnlock()`.

- **Directory Inodes**:
  Directory inodes (`dirInode` in `internal/fs/inode/dir.go` and `baseDirInode` in `internal/fs/inode/base_dir.go`) implement `locker.RWLocker` using a `locker.RWLocker` field:
  - `dirInode.mu` is a `locker.RWLocker` (created via `locker.NewRW(...)`).
  - They expose `Lock()`, `Unlock()`, `RLock()`, and `RUnlock()` methods on the inode itself.
  - Since `DirInode` interface embeds `Inode` and exposes `RLock()` and `RUnlock()`, any directory inode can be checked/cast to `locker.RWLocker`.

- **File Inodes**:
  File inodes (`FileInode` in `internal/fs/inode/file.go`) do NOT implement `locker.RWLocker`.
  - They use `f.mu syncutil.InvariantMutex` which only supports exclusive locking.
  - They only implement `Lock()` and `Unlock()` of `sync.Locker`.
  - Type asserting a `FileInode` to `locker.RWLocker` will fail (`ok == false`).

## 3. Locking Logic Analysis

### Lookup Path
- `LookUpInode` calls `lookUpOrCreateChildInode`.
- `lookUpOrCreateChildInode` calls `lookUpOrCreateInodeIfNotStale` to obtain the locked child inode.
- Currently, `lookUpOrCreateInodeIfNotStale` always acquires a write-lock (`in.Lock()`) on the child inode before returning it.
- After obtaining the locked child, `LookUpInode` retrieves its attributes via `fs.getAttributes(ctx, child)`.
- Finally, it calls `defer fs.unlockAndMaybeDisposeOfInode(child, &err)` to release the child's lock.

### Directory Listing Path
- `ReadDirPlus` fetches cores for directory entries and loops over them, calling `coreToDirentPlus`.
- `coreToDirentPlus` calls `lookUpOrCreateInodeIfNotStale` which returns the child write-locked.
- It then retrieves attributes using `child.Attributes(...)` and unlocks the child via `defer child.Unlock()`.

### Problem
Because lookups and listings acquire write locks on child directory inodes (even though they only perform read/attribute operations), they can cause writer starvation and serialisation delays.

---

## 4. Proposed Design for Read-Locking and Lock Upgrading

### R1. Introduce Read-Only Inode Locking Mode in Lookup
We can introduce a boolean parameter `readLock` to `lookUpOrCreateInodeIfNotStale` and `lookUpOrCreateChildInode`.
- If `readLock` is true:
  - Check if the child inode implements `locker.RWLocker`.
  - If it does, acquire a read-lock (`RLock()`).
  - Otherwise, fallback to a write-lock (`Lock()`).
- The locking helper functions will return a boolean `isReadLocked` along with the inode.
- When unlocking the inode (e.g. in `unlockAndMaybeDisposeOfInode` or in `coreToDirentPlus` defer), we check `isReadLocked`.
  - If true, call `RUnlock()`.
  - If false, call `Unlock()`.

### R2. Use Read-Locking for Lookup and Attribute Retrieval
- Modify `LookUpInode` to call `lookUpOrCreateChildInode(..., true)`.
- Modify `coreToDirentPlus` to call `lookUpOrCreateInodeIfNotStale(..., true)`.

### R3. Upgrade Lock Safely on Remote Size Changes
When `cmp == 2` in `lookUpOrCreateInodeIfNotStale`, the child size has changed remotely. We must update the size of the inode, which requires a write lock.
- If we hold a read lock on the inode (`isReadLocked == true`), we cannot call `UpdateSize` directly.
- We must upgrade the lock safely to avoid deadlocks and race conditions:
  1. Release the read lock on the inode.
  2. Release `fs.mu` lock.
  3. Loop/retry the lookup with a write lock (`localReadLock = false`).
  4. On the next iteration, the inode will be acquired with a write lock.
  5. The size change can then be safely updated under the write lock.

---

## 5. Proposed Code Changes (Diff Patch)

The exact code changes to `internal/fs/fs.go` can be defined in a patch file: `proposed_changes.patch`.
- `lookUpOrCreateInodeIfNotStale` signature and loop implementation updated.
- `createDirInode` updated to accept `readLock` and return `isReadLocked`.
- `lookUpOrCreateChildInode` updated to accept `readLock` and return `isReadLocked`.
- `unlockAndMaybeDisposeOfInode` and `unlockAndDecrementLookupCount` updated to accept `isReadLocked`.

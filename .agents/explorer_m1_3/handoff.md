# Handoff Report: Read-Locking Inode Lookup Optimizations

This report details the findings and proposals for implementing read-locking optimizations in inode lookup and listing paths, along with safe lock-upgrading logic.

## 1. Observation

- **Core Lookup Functions**:
  - `LookUpInode`: `internal/fs/fs.go:1912`
  - `coreToDirentPlus`: `internal/fs/fs.go:1700`
  - `lookUpOrCreateInodeIfNotStale`: `internal/fs/fs.go:1089` (prior to edit)
- **Lock Definition**:
  - `locker.RWLocker`: `internal/locker/rw_locker.go:26`
- **Inode Lock Implementations**:
  - Directory Inodes (`dirInode` in `internal/fs/inode/dir.go` and `baseDirInode` in `internal/fs/inode/base_dir.go`) implement `locker.RWLocker` and provide reader-writer locking methods (`Lock()`, `Unlock()`, `RLock()`, `RUnlock()`).
  - File Inodes (`FileInode` in `internal/fs/inode/file.go`) only implement `sync.Locker` via `syncutil.InvariantMutex` (exclusive lock only).
- **Existing locking behavior**:
  - `lookUpOrCreateInodeIfNotStale` returns all inodes locked with an exclusive write lock (`in.Lock()`).
  - `LookUpInode` and `coreToDirentPlus` retrieve attributes from the child inode while it is locked exclusively, then unlock it using exclusive unlock (`Unlock()`).
- **Remote Size changes (`cmp == 2`)**:
  - In `lookUpOrCreateInodeIfNotStale` (lines 1166-1171):
    ```go
    if cmp == 2 {
        logger.Warnf("The size of object has changed remotely at the same generation. Updating the existing inode to reflect the size change.\n")
        existingInode.UpdateSize(oGen.Size)
        in = existingInode
        return
    }
    ```
    This updates the existing inode's size remotely, which is a mutation requiring a write lock.

## 2. Logic Chain

1. **R1 (Read-Only Locking Support)**:
   - Since directories implement `locker.RWLocker` but files only implement `sync.Locker`, we can type-assert the target inode using `rwLocker, ok := in.(locker.RWLocker)`.
   - If `ok == true` and the caller requested read-locking, we can call `rwLocker.RLock()`. Otherwise, we fallback to `in.Lock()`.
   - We must return the status `isReadLocked` back to the caller so they can release the lock using the appropriate method (`RUnlock()` vs `Unlock()`).
2. **R2 (Read-Locking in Lookup & Listing)**:
   - We update `LookUpInode` and `coreToDirentPlus` to request a read lock when obtaining child inodes (passing `readLock = true` to `lookUpOrCreateChildInode` and `lookUpOrCreateInodeIfNotStale`).
   - The returned status `isReadLocked` is propagated and used to conditionally release the lock via `RUnlock()` if it was read-locked.
   - Other filesystem modification operations (e.g. `MkDir`, `CreateFile`, `CreateSymlink`, `RmDir`, `Rename`) continue to request a write lock (`readLock = false`).
3. **R3 (Safe Lock Upgrading)**:
   - If a lookup finds that the child size has changed remotely (`cmp == 2`), we must call `existingInode.UpdateSize(...)`.
   - If the child was read-locked, calling `UpdateSize` would be a thread-safety violation.
   - Since atomic lock upgrading is not supported by Go's `sync.RWMutex`, we must:
     1. Release the read lock (`rwLocker.RUnlock()`).
     2. Release the filesystem lock (`fs.mu.Unlock()`).
     3. Restart the lookup loop, but this time requesting a write lock (`localReadLock = false`).
     4. On retry, the inode will be write-locked, and the size change can be safely applied.

## 3. Caveats

- Atomic lock upgrading is not possible. The release-and-retry approach means another goroutine can modify the mapping between releasing the read lock and acquiring the write lock. However, the retry loop in `lookUpOrCreateInodeIfNotStale` naturally handles this by re-verifying the inode mapping (`fs.generationBackedInodes[ic.FullName] == existingInode`) after re-acquiring the lock.
- No other paths (e.g., file reads/writes) were modified as they were out of scope.

## 4. Conclusion

The proposed read-locking design successfully optimizes inode lookup and directory listing operations by acquiring read locks on directories, preventing writer starvation. The lock upgrading logic correctly transitions a read lock to a write lock by releasing and retrying under the existing retry loops, avoiding deadlocks while ensuring thread safety.

## 5. Verification Method

- The proposed patch file is saved at: `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/explorer_m1_3/proposed_changes.patch`.
- Apply the patch: `git apply .agents/explorer_m1_3/proposed_changes.patch`
- Compile the package and its tests:
  - `go build ./internal/fs/...`
  - `go test -run=None ./internal/fs/...`
- Clean up: `git checkout internal/fs/fs.go`

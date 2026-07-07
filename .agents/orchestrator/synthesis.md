# Synthesis of Milestone 1 Exploration and Design

All three Explorer subagents (explorer_m1_1, explorer_m1_2, explorer_m1_3) have completed their analyses and are in complete consensus on the findings, architecture, and design for read-locking inode lookups.

## Consensus

### 1. Target Symbols and Paths
- **LookUpInode**: `internal/fs/fs.go` (around line 1912/1986 depending on line counting context).
- **coreToDirentPlus**: `internal/fs/fs.go` (around line 1700/1774).
- **lookUpOrCreateInodeIfNotStale**: `internal/fs/fs.go` (around line 1089/1113).
- **locker.RWLocker**: `internal/locker/rw_locker.go` (line 26).
- **lookupCount**: `internal/fs/inode/lookup_count.go`.

### 2. Inode Locking Capabilities
- Directory Inodes (`dirInode` and `baseDirInode`) implement `locker.RWLocker` and support read-locking via `RLock()` and `RUnlock()`.
- File and Symlink Inodes do not implement `locker.RWLocker` (they only implement `sync.Locker` via exclusive locks).

### 3. Core Problems & Solutions
- **R1: Read-Lock Option in Lookup**: Introduce a `readLock bool` parameter to `lookUpOrCreateInodeIfNotStale` and helper functions. If `readLock` is true and the inode implements `locker.RWLocker`, acquire a read lock; otherwise acquire a write lock. The return value should also convey whether a read lock or write lock was acquired so callers can unlock appropriately.
- **R2: Update Lookup & Listing Paths**: Update `LookUpInode` and `coreToDirentPlus` to request a read lock when obtaining child inodes, and release the lock with the appropriate unlock operation. Other operations (like mkdir, create, rmdir, rename) must request a write lock.
- **Race on lookupCount**: Since multiple goroutines can now lock and return a directory inode under a read lock concurrently, they will mutate the lookup count concurrently. Thus, the lookup count must be updated to use thread-safe atomic operations (`sync/atomic`).
- **R3: Lock Upgrading on Remote Size Changes**: If a lookup finds that the child size has changed remotely (`cmp == 2` in `lookUpOrCreateInodeIfNotStale`), size state must be updated, which requires a write lock. If the child was read-locked, we must release the read lock, release `fs.mu` filesystem lock, acquire the write lock, re-acquire `fs.mu`, and retry the staleness checks. If a read lock was originally requested, we downgrade back to a read lock before returning to preserve consistency.

## Divergence / Nuances
None. All three explorers arrived at the identical design. They also generated compilation patches which verified correctly.

## Path to Implementation (Milestone 2)
The implementation will apply these changes:
1. `internal/fs/inode/lookup_count.go`: Update `lookupCount` struct and methods (`Inc`, `Dec`) to use `sync/atomic` operations.
2. `internal/fs/fs.go`:
   - Modify `createDirInode`, `lookUpOrCreateInodeIfNotStale` to accept `readLock bool`.
   - Update lock/unlock wrappers and helper signatures (`unlockAndMaybeDisposeOfInode`, `unlockAndDecrementLookupCount`) to support read locks.
   - Implement the lock upgrade flow on `cmp == 2`.
   - Update call-sites to pass the correct `readLock` value.

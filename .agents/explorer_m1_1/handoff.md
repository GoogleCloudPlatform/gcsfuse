# Handoff Report: Read-Locking Exploration and Design (M1)

This handoff report summarizes the exploration of lookups, listing paths, and locking logic, and describes the proposed design to support read-locking and lock upgrades on size changes.

## 1. Observation

*   **LookUpInode** is implemented in `internal/fs/fs.go` at line 1912:
    ```go
    func (fs *fileSystem) LookUpInode(
        ctx context.Context,
        op *fuseops.LookUpInodeOp) (err error) {
    ```
*   **coreToDirentPlus** is implemented in `internal/fs/fs.go` at line 1700:
    ```go
    func (fs *fileSystem) coreToDirentPlus(ctx context.Context, fullName inode.Name, core inode.Core, parInodeCtx context.Context) (entryPlus *fuseutil.DirentPlus, err error) {
    ```
*   **lookUpOrCreateInodeIfNotStale** is implemented in `internal/fs/fs.go` at line 1089:
    ```go
    func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(parInodeCtx context.Context, ic inode.Core) (in inode.Inode, err error) {
    ```
*   **locker.RWLocker** is defined in `internal/locker/rw_locker.go` at line 26:
    ```go
    type RWLocker interface {
        sync.Locker
        RLock()
        RUnlock()
    }
    ```
*   **Directory Inodes** (`dirInode` in `internal/fs/inode/dir.go` and `baseDirInode` in `internal/fs/inode/base_dir.go`) define `RLock` and `RUnlock` methods delegating to the internal `mu locker.RWLocker`.
*   **File and Symlink Inodes** use exclusive-only locks (`syncutil.InvariantMutex` and `sync.Mutex` respectively) and do not support `locker.RWLocker`.
*   **lookupCount** (`internal/fs/inode/lookup_count.go`) is not thread-safe: it uses simple `uint64` and `bool` fields without atomic operations or internal synchronization.

---

## 2. Logic Chain

1. **R1 (Read-Only Lock Mode)**: Since directory inodes support read-locking through `locker.RWLocker` and file/symlink inodes do not, we can support read-locking at lookup by checking if the inode implements `locker.RWLocker` via type assertion, and acquiring the read lock if requested and supported.
2. **R2 (Read-Locking in Lookup/Listing)**: Directory listing (`coreToDirentPlus`) and child lookup (`LookUpInode`) are read-only operations on the child inode. Passing a `readLock = true` parameter down to `lookUpOrCreateInodeIfNotStale` allows these paths to acquire read locks on directories.
3. **Lookup Count Race**: Because multiple lookups or listings can return a directory inode in a read-locked state concurrently, the lookup count of the directory inode will be incremented/decremented concurrently. Since the fields in `lookupCount` are not atomic, this would lead to a data race. Therefore, `lookupCount` must be modified to use `sync/atomic` operations.
4. **R3 (Lock Upgrading)**: In `lookUpOrCreateInodeIfNotStale`, if the lookup finds that the child size has changed remotely (`cmp == 2`), the size must be updated, which requires a write lock. If the child was read-locked, we can safely upgrade the lock by releasing the read lock and the filesystem lock (`fs.mu`), acquiring the write lock, and re-acquiring `fs.mu`. We then re-validate the inode's presence in the map and the generation match. After updating the size, if a read lock was originally requested, we downgrade the lock back to a read lock (by releasing the write lock and acquiring the read lock) while holding `fs.mu` to preserve the return contract.

---

## 3. Caveats

*   **File Inode Locking**: Since `FileInode` and `SymlinkInode` only support exclusive locking, they will continue to be write-locked even if a read lock is requested. This is safe, but means parallel read-operations on files at the FUSE level still serialize on their individual locks.
*   **No Integration/FS Tests Run**: Per the constraints, no fs/integration tests have been executed. Validation is strictly restricted to code reviews and static verification (`go build ./internal/fs/...`).

---

## 4. Conclusion

The design successfully allows concurrent read-locking on directory inodes during lookups and listings, preventing writer starvation. The lock upgrade path safely transitions from read-lock to write-lock to handle size updates on remote changes, and downgrades back to read-lock afterward.

The complete code implementation changes are documented in `analysis.md` at `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/explorer_m1_1/analysis.md`.

---

## 5. Verification Method

1. Apply the code changes described in `analysis.md`.
2. Compile the package:
   ```bash
   go build ./internal/fs/...
   ```
3. Run the unit tests of the affected packages to ensure no regressions in basic FUSE fs/inode functionality:
   ```bash
   go test -v ./internal/fs/...
   ```

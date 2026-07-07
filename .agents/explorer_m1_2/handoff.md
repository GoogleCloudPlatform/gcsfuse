# Handoff Report: Read-Locking Inode Lookups (Milestone 1)

## 1. Observation
- **Symbols and Paths**:
  - `LookUpInode`: `internal/fs/fs.go:1986`
  - `coreToDirentPlus`: `internal/fs/fs.go:1774`
  - `lookUpOrCreateInodeIfNotStale`: `internal/fs/fs.go:1113`
  - `locker.RWLocker`: `internal/locker/rw_locker.go:26`
  - `lookupCount`: `internal/fs/inode/lookup_count.go`
- **Current Locks**:
  - `dirInode` implements `locker.RWLocker` via a private `mu locker.RWLocker` field.
  - `FileInode` does not support read-locking, only implements exclusive `sync.Locker`.
  - Lookups and listings lock returning child inodes exclusively using `Lock()`.
- **Race Condition on Lookup Count**:
  - Inodes' lookup count field `lc.count` is modified via non-atomic increments/decrements in `IncrementLookupCount()` and `DecrementLookupCount()`. If inodes are read-locked concurrently, this state will be corrupted due to race conditions.

---

## 2. Logic Chain
1. To prevent reader starvation on directory concurrent lookups, directory inodes (which support `locker.RWLocker`) should be read-locked (`RLock()`) instead of write-locked (`Lock()`).
2. Concurrent read lock holders must not race when updating lookup counts. Therefore, `lookupCount` must use atomic primitives (`sync/atomic` package) to ensure thread-safety without holding write locks.
3. Callers performing read-only operations (`LookUpInode`, `coreToDirentPlus`) should pass `readLock = true` down the lookup chain.
4. If a lookup path discovers that the remote size has changed (`cmp == 2` in `lookUpOrCreateInodeIfNotStale`), size state must be updated. This requires an exclusive write lock.
5. To upgrade a read lock to a write lock without violating lock ordering rules (which require acquiring child lock before `fs.mu` lock), we must release the child read lock, drop `fs.mu`, re-acquire the child write lock, re-acquire `fs.mu`, and retry the staleness checks.

---

## 3. Caveats
- Checked static compilation of `internal/fs/...` packages only. Did not run integration or mount tests.
- Assumed `sync/atomic` atomic operations do not introduce noticeable performance overhead.
- Lock upgrading requires releasing the read lock and filesystem lock before retrying. This implies that during the retry window, other threads may modify the inode state. The staleness retry loop correctly detects changes (using generation/comparisons) and retries cleanly.

---

## 4. Conclusion
The proposed design is correct, deadlock-free, and compiles successfully under static checks.
A patch file `read_locking.patch` containing the complete implementation code has been generated in the workspace folder (`.agents/explorer_m1_2/read_locking.patch`).

---

## 5. Verification Method
1. Apply the patch file:
   ```bash
   git apply .agents/explorer_m1_2/read_locking.patch
   ```
2. Verify static compilation:
   ```bash
   go build ./internal/fs/...
   ```
3. Run tests in `internal/fs` package to confirm no regression:
   ```bash
   go test ./internal/fs/...
   ```
4. Revert the patch:
   ```bash
   git restore internal/fs/fs.go internal/fs/inode/lookup_count.go
   ```

---

## 6. Remaining Work
- Apply the patch in Milestone 2.
- Perform code review and static verification in Milestone 3.

# Review Report and Handoff

## Review Summary

**Verdict**: REQUEST_CHANGES (FAIL)

## Findings

### [Critical] Finding 1: Race Condition in `unlockAndDecrementLookupCount` with Read Locks

- **What**: A concurrent lookup operation can retrieve, lock, and verify an inode whose lookup count has just been decremented to 0 by another thread, but has not yet been deleted from the filesystem indexes. This causes the lookup to return a destroyed inode.
- **Where**: `internal/fs/fs.go` (specifically in `unlockAndDecrementLookupCount` and `lookUpOrCreateInodeIfNotStale`).
- **Why**: 
  When an inode is read-locked, multiple threads can acquire the read lock concurrently.
  If Thread A calls `unlockAndDecrementLookupCount` on a read-locked inode, the lookup count is decremented to 0, setting `shouldDestroy` to `true`.
  Before Thread A acquires `fs.mu` to delete the inode from indexes, Thread B initiates a lookup, finds the inode in the index, unlocks `fs.mu`, and acquires a read lock on the inode. Since Thread A only holds a read lock, Thread B does not block.
  Thread B re-acquires `fs.mu` and verifies that the index still points to the inode (which it does, because Thread A hasn't deleted it yet). Thread B then returns the inode and increments its lookup count back to 1.
  Finally, Thread A acquires `fs.mu`, deletes the inode from indexes, and calls `in.Destroy()`. This leaves Thread B with a reference to a destroyed inode, leading to potential use-after-free, panic, or data corruption.
- **Suggestion**: 
  Perform both the lookup count decrement and index deletion/destruction atomically under `fs.mu` when `shouldDestroy` is evaluated. Because `fs.mu` is acquired after the inode lock (preserving lock hierarchy), locking `fs.mu` before decrementing and holding it through index deletion/destruction will block concurrent lookups at `fs.mu.Lock()` until the deletion is complete. Once `fs.mu.Lock()` is released, the concurrent lookup will fail its index verification check and retry safely.

---

## 1. Observation

In `internal/fs/fs.go`:
```go
func (fs *fileSystem) unlockAndDecrementLookupCount(in inode.Inode, N uint64, readLocked bool) {
	name := in.Name()

	// Decrement the lookup count.
	shouldDestroy := in.DecrementLookupCount(N)

	// Update file system state, orphaning the inode if we're going to destroy it
	// below.
	if shouldDestroy {
		fs.mu.Lock()
		delete(fs.inodes, in.ID())
        ...
		fs.mu.Unlock()
	}
    ...
```

In `lookUpOrCreateInodeIfNotStale`:
```go
		// Otherwise we need to read the inode's source generation below, which
		// requires the inode's lock. We must not hold the inode lock while
		// acquiring the file system lock, so drop it while acquiring the inode's
		// lock, then reacquire.
		fs.mu.Unlock()
		readLocked = lockInode(existingInode, readLock)
		fs.mu.Lock()

		// Check that the index still points at this inode. If not, it's possible
		// that the inode is in the process of being destroyed and is unsafe to
		// use. Go around and try again.
		if fs.generationBackedInodes[ic.FullName] != existingInode {
			unlockInode(existingInode, readLocked)
			continue
		}
```

In `internal/fs/inode/lookup_count.go`:
```go
func (lc *lookupCount) Inc() {
	if lc.destroyed {
		panic(fmt.Sprintf("Inode %v has already been destroyed", lc.id))
	}

	atomic.AddUint64(&lc.count, 1)
}

func (lc *lookupCount) Dec(n uint64) (destroy bool) {
    ...
	for {
		current := atomic.LoadUint64(&lc.count)
        ...
		newCount := current - n
		if atomic.CompareAndSwapUint64(&lc.count, current, newCount) {
			destroy = newCount == 0
			break
		}
	}
	return
}
```

---

## 2. Logic Chain

1. **Locking Mechanism Change**: The read-locking optimization allows directory and metadata lookup operations to acquire a read lock (`RLock`) on the inode instead of a write lock (`Lock`).
2. **Concurrent Read Lock Sharing**: Under read-locking, `lockInode(existingInode, true)` permits multiple concurrent threads to hold read locks on the same inode simultaneously.
3. **Dec to 0 Gap**: If Thread A decrements the lookup count to 0 in `DecrementLookupCount(N)`, it sets `shouldDestroy` to `true`.
4. **FS Lock Release / Lack of Atomicity**: The decrement operation occurs outside of the filesystem lock `fs.mu`.
5. **Interleaving Lookup**: While Thread A is in the gap before acquiring `fs.mu` to delete the index entry, Thread B performs a lookup for the same inode:
   - Thread B retrieves `existingInode` from the index under `fs.mu`.
   - Thread B unlocks `fs.mu`.
   - Thread B calls `lockInode(existingInode, true)`. Because Thread A holds a read lock, Thread B's read lock request succeeds instantly without blocking.
   - Thread B locks `fs.mu` and confirms that the index entry still matches `existingInode`.
   - Thread B increments the lookup count of `existingInode` to 1 and returns it as a valid, live inode.
6. **Use-After-Free**: Thread A then acquires `fs.mu`, deletes `existingInode` from the indexes, and calls `in.Destroy()`. This destroys the inode context/prefetchers, leaving Thread B with a reference to a destroyed inode.

---

## 3. Caveats

- We assumed lookup count decrement to 0 is the primary trigger for inode destruction, which is standard in this codebase.
- No other unexplored areas were found; we focused on correctness of synchronization, thread safety, deadlock avoidance, lock upgrade/downgrade logic, and unit/compilation tests as per objectives.

---

## 4. Conclusion

The read-locking optimization introduces a critical concurrency race condition where an inode can be destroyed after a concurrent thread has successfully looked it up and retrieved it from the index. Therefore, the implementation in its current form fails verification.

---

## 5. Verification Method

To verify compilation:
```bash
go build ./internal/fs/...
```
To verify unit tests:
```bash
go test -v ./internal/fs/inode/...
```

To demonstrate the race condition:
Under high concurrent lookup and forget load, or by placing a sleep in `unlockAndDecrementLookupCount` right before `fs.mu.Lock()`, a test case can trigger a panic where `Destroy()` is called on a returned active inode.

---

## Verified Claims

- Code compiles successfully -> Verified via `go build ./internal/fs/...` -> PASS
- Unit tests pass -> Verified via `go test ./internal/fs/inode/...` -> PASS
- Deadlock avoidance logic -> Verified (lock hierarchy inode -> fs.mu is consistently followed) -> PASS
- Thread safety of lookup_count -> Verified via `lookup_count_test.go` and code inspection -> PASS
- Correctness of read-locking under concurrent lookups -> FAILED (due to race condition during decrement/deletion)

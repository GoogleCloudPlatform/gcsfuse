# Synthesis of Milestone 3 Verification & Review

Milestone 3 verification has been completed by 2 Reviewers, 1 Challenger, and 1 Forensic Auditor. The Forensic Auditor reported a CLEAN verdict with no integrity violations. However, both Reviewers and the Challenger identified two critical concurrency and deadlock defects in the implementation:

## 1. Deadlock in Lock Downgrade Flow (Reviewer 1 & Challenger 2 & Challenger 1)
- **Problem**: In `lookUpOrCreateInodeIfNotStale`, when downgrading a lock from write to read (lines 1218-1221 and 1234-1236 in the optimized code), the code calls `existingInode.(locker.RWLocker).RLock()` while holding the filesystem lock `fs.mu`. This violates the lock ordering rule (inode lock must be acquired before `fs.mu`).
- **Consequence**: This causes a circular deadlock with operations like `ForgetInode` or `RmDir` that lock the inode exclusively and then try to lock `fs.mu`. A stress test (`TestReadLockingUpgradeDeadlock`) was created by Challenger 1 and successfully reproduced the deadlock (hanging and panicking after 3 seconds).
- **Remediation**: Avoid calling `RLock()` while holding `fs.mu`. Instead of manually downgrading, the thread should release the write lock on the inode and `continue` the loop. The next iteration will find the inode, drop `fs.mu`, safely acquire the read lock (without holding `fs.mu`), re-acquire `fs.mu`, and return it.

## 2. Race Condition in `unlockAndDecrementLookupCount` (Reviewer 2)
- **Problem**: In `unlockAndDecrementLookupCount`, the lookup count is decremented outside of `fs.mu`.
- **Consequence**: A concurrent lookup operation can retrieve, lock, and verify an inode whose lookup count was decremented to 0 but has not yet been deleted from indexes. The lookup increments its count back to 1 and returns it, but the decrementing thread subsequently deletes the index entries and calls `in.Destroy()`, leading to a use-after-free or panic on the active inode.
- **Remediation**: Perform both the lookup count decrement and index deletion/destruction atomically under `fs.mu`. Lock `fs.mu` before calling `in.DecrementLookupCount(N)`. This blocks concurrent lookups until deletion is complete, causing them to fail verification and retry safely.

## Verdict
**REQUEST_CHANGES (FAIL)**. The implementation must be updated to address these two critical issues.

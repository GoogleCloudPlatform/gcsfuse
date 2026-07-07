# Original User Request

## Initial Request — 2026-07-07T22:12:46Z

# Teamwork Project Prompt

Implement read-locking optimizations for inode lookups in `gcsfuse` to prevent writer starvation and eliminate lookup delays on directories.

Working directory: /usr/local/google/home/kislayk/gitproj/gcsfuse
Integrity mode: development

## Requirements

### R1. Introduce Read-Only Inode Locking Mode in Lookup
Support looking up and returning child inodes in a read-locked state if they support read-write locking (i.e. directories implementing `locker.RWLocker`). This prevents writer lock requests (and subsequent reader starvation) during concurrent lookups.

### R2. Use Read-Locking for Lookup and Attribute Retrieval
Modify `LookUpInode` and listing paths (`coreToDirentPlus`) to request the child inode in a read-locked state.

### R3. Upgrade Lock Safely on Remote Size Changes
If a lookup finds that the child size has changed remotely (i.e., `cmp == 2` in `lookUpOrCreateInodeIfNotStale`), safely upgrade the lock to a write lock by releasing the read lock/fs lock and retrying, to allow updating the inode size.

## Acceptance Criteria

### Correctness and Thread Safety
- No regression in locking correctness under concurrent lookups and modifications.
- Inode locking rules (acquire inode lock before fs lock) are strictly preserved to avoid deadlocks.

### Verification (Static Only)
- Code compiles successfully using `go build ./internal/fs/...`.
- Implementation is verified by code review and static analysis.
- *Constraint: Do NOT run any fs or integration tests.*

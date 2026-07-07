## 2026-07-07T22:19:58Z

You are worker_m2_1. Your working directory is `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/worker_m2_1/`.
Objective: Implement the read-locking optimizations for inode lookups in gcsfuse.

Instructions:
1. Load the domain skill at `/google/src/files/head/depot/google3/research/omega/teamwork/playbooks/software_engineering/SKILL.md`.
2. Apply the design documented in the synthesis report at `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/orchestrator/synthesis.md`.
Specifically:
- Update `internal/fs/inode/lookup_count.go` to make it thread-safe using `sync/atomic`.
- Modify `internal/fs/fs.go` (and associated files/types if needed) to support read-locking for inodes implementing `locker.RWLocker`.
- Update `LookUpInode` and `coreToDirentPlus` to request read locks.
- Implement the lock upgrade from read to write lock on remote size change (`cmp == 2` in `lookUpOrCreateInodeIfNotStale`) with proper releasing/retrying and downgrading back to read lock if originally requested.
3. Verify that the code compiles successfully by running:
`go build ./internal/fs/...`
and that basic unit tests pass:
`go test ./internal/fs/...`
*Constraint: Do NOT run any integration or mount tests.*

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A Forensic Auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write a handoff/completion report describing the changes, compilation, and testing results to `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/worker_m2_1/handoff.md`. Notify the orchestrator (conversation ID: 16ecf609-e12c-496f-a57e-823517e4cde8) when done.

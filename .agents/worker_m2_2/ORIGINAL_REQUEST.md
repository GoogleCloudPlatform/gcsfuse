## 2026-07-07T22:54:38Z
You are worker_m2_2. Your working directory is `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/worker_m2_2/`.
Objective: Fix the concurrency defects identified during verification.

Instructions:
1. Load the domain skill at `/google/src/files/head/depot/google3/research/omega/teamwork/playbooks/software_engineering/SKILL.md`.
2. Read the synthesis report at `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/orchestrator/synthesis_m3.md` detailing the deadlock in lock downgrade flow and the race condition in `unlockAndDecrementLookupCount`.
3. Apply the fixes:
- Fix the deadlock: In `lookUpOrCreateInodeIfNotStale` (in `internal/fs/fs.go`), remove calling `RLock()` while holding `fs.mu`. Instead, release the write lock (or read lock/whatever is held), drop `fs.mu` (if held), and `continue` the loop. The next iteration will re-acquire the correct lock safely.
- Fix the race: In `unlockAndDecrementLookupCount` (in `internal/fs/fs.go`), call `fs.mu.Lock()` before calling `in.DecrementLookupCount(N)`. Perform both decrement and deletion from `fs.inodes`, `fs.generationBackedInodes`, etc. under `fs.mu.Lock()`. Make sure `fs.mu` is unlocked before calling `in.Destroy()`.
4. Compile the packages:
`go build ./internal/fs/...`
5. Run the deadlock unit test to verify that the deadlock is resolved:
`go test -v -run "TestReadLockingUpgradeDeadlock" ./internal/fs/...`
6. Run the inode unit tests:
`go test -v -count=1 ./internal/fs/inode/...`
*Constraint: Do NOT run integration or mount tests.*

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A Forensic Auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write a handoff/completion report detailing your changes and test outcomes to `/usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/worker_m2_2/handoff.md`. Notify the orchestrator (conversation ID: 16ecf609-e12c-496f-a57e-823517e4cde8) when done.

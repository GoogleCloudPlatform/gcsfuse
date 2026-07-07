# Plan - Read-locking Optimizations in gcsfuse

## Objective
Implement read-locking optimizations for inode lookups in gcsfuse to prevent writer starvation and eliminate lookup delays on directories.

## Milestones

### Milestone 1: Exploration and Analysis
- Spawn `teamwork_preview_explorer` to:
  - Locate `LookUpInode`, `coreToDirentPlus`, and `lookUpOrCreateInodeIfNotStale`.
  - Identify how directory/inode locking is currently implemented.
  - Formulate a precise design for introducing read-only locks, modifying lookup paths, and upgrading locks on remote size changes.
- Output: Explorer analysis report.

### Milestone 2: Implementation & Compilation
- Spawn `teamwork_preview_worker` to:
  - Implement read-only inode locking mode in Lookup.
  - Modify `LookUpInode` and listing paths (`coreToDirentPlus`) to request the child inode in a read-locked state.
  - Implement lock upgrade when remote size changes (release read lock, get write lock, retry).
  - Verify that compilation passes (`go build ./internal/fs/...`).
- Output: Implementation report.

### Milestone 3: Review & Verification
- Spawn `teamwork_preview_reviewer` to review code correctness, thread safety, deadlock avoidance, and API sanity.
- Spawn `teamwork_preview_challenger` to verify behavior if needed (via compile/static analysis checks).
- Spawn `teamwork_preview_auditor` to run integrity checks.
- Output: Verification reports.

### Milestone 4: Final Handover
- Collect and synthesize all outputs, verify against constraints, and send completion report to the Sentinel.

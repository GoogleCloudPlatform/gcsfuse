# Project: gcsfuse Read-Locking Optimizations

## Architecture
- `gcsfuse` exposes Google Cloud Storage as a FUSE file system.
- Inode management and lookups are key operations under `internal/fs/`.
- Inodes support read-write locking via `locker.RWLocker` if they are directories or support concurrent lookups.

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|---|---|---|---|
| 1 | M1: Exploration | Locate code and design the lock transition | None | DONE |
| 2 | M2: Implementation | Modify locks in lookup and listing paths, handle upgrade | M1 | DONE |
| 3 | M3: Review & Audit | Code reviews, static verification, and integrity audit | M2 | IN_PROGRESS |
| 4 | M4: Final Synthesis | Synthesize results and report to Sentinel | M3 | PLANNED |

## Interface Contracts
- To be updated by Explorer/Worker as implementation starts.

## Code Layout
- `internal/fs/` contains FUSE file system implementation.

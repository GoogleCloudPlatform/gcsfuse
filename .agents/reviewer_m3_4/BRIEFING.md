# BRIEFING

## 🔒 My Identity
- Role: reviewer, critic
- Workspace: /usr/local/google/home/kislayk/gitproj/gcsfuse/
- Folder: /usr/local/google/home/kislayk/gitproj/gcsfuse/.agents/reviewer_m3_4/

## 🔒 Key Constraints
- CODE_ONLY network mode. No external web access.
- Do NOT run any integration or mount tests.
- Only write to folder `.agents/reviewer_m3_4/` (except when requested to modify code, but here we don't modify code, we only review and build/test).

## Mission
Review the read-locking optimizations and concurrency defect fixes in gcsfuse.

## Review Checklist
- **Items reviewed**:
- **Verdict**: pending
- **Unverified claims**:

## Attack Surface
- **Hypotheses tested**:
- **Vulnerabilities found**:
- **Untested angles**:

## Short-term Plan
1. Check git diff / git status or recent commits to find the exact changes to review, or read `internal/fs/fs.go` and `internal/fs/inode/lookup_count.go`.
2. Build and run tests to establish baseline.
3. Perform review.

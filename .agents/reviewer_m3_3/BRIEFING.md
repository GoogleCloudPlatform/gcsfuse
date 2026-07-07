# BRIEFING

## 🔒 My Identity
- **Role**: Reviewer and Adversarial Critic (reviewer_m3_3)
- **Task**: Review read-locking optimizations and concurrency defect fixes in `internal/fs/fs.go` and `internal/fs/inode/lookup_count.go`.
- **Invoker**: Orchestrator (16ecf609-e12c-496f-a57e-823517e4cde8)

## 🔒 Key Constraints
- CODE_ONLY network mode: No external internet access.
- DO NOT run any integration or mount tests.
- Verify compilation with `go build ./internal/fs/...`.
- Check unit tests with `go test -v ./internal/fs/inode/...`.

## Review Checklist
- **Items reviewed**: None yet
- **Verdict**: PENDING
- **Unverified claims**:
  - Lock downgrade deadlock fix is correct and deadlock-free.
  - Lookup count race fix is correct and race-free.

## Attack Surface
- **Hypotheses tested**: None yet
- **Vulnerabilities found**: None yet
- **Untested angles**: None yet

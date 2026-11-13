---
name: gcsfuse-minor-release-branch
description: End-to-end runbook for cutting and pushing a new persistent GCSFuse minor release branch (v<major>.<minor>_release) from master after pre-cut govulncheck and unit test gating (raising master fix PRs first if any vulnerabilities or unit test failures exist).
---

# GCSFuse Minor Release Branch Creation Skill

This skill provides a standardized, end-to-end workflow for cutting a new persistent minor release branch (`v<major>.<minor>_release`, e.g., `v3.13_release`) for GCSFuse from `master` (or a specified commit hash). Before cutting the release branch, it enforces strict pre-cut health gates on `master`:
1. **Zero-CVE Gate (`govulncheck`)**: If `govulncheck` detects any vulnerabilities on `master`, do **not** cut the release branch; instead, raise a PR against `master` to remediate the vulnerabilities first.
2. **Unit Test Gate (`go test`)**: If any unit test fails (unless the user explicitly specifies otherwise), do **not** cut the release branch; instead, raise a PR against `master` to fix the failing unit test(s) first.
3. **Cut & Push Release Branch**: Only when `master` has zero vulnerabilities and zero unit test failures, cut `v<major>.<minor>_release` and push it to `origin`.

## Persistent Minor Release Branch & 1:1 GKE Minor Locking Model

### 1. Single Persistent Minor Release Branch (`v<major>.<minor>_release`)
- `v<major>.<minor>_release` is cut **once** from `master` (or a specified `<commit_sha>`) for the entire `<major>.<minor>` release series (e.g., `v3.13_release`).
- The initial release tag created by the CD pipeline from this branch is `v<major>.<minor>.0` (e.g., `v3.13.0`), and all subsequent patch releases (`v<major>.<minor>.1`, `v<major>.<minor>.2`, ...) land directly on `v<major>.<minor>_release`.
- A `v<major>.<minor>_release` branch reaches End of Life (EOL) 2 years after the release of the last GKE minor version that depends on it.

### 2. 1:1 GKE Minor Version Locking & 30-Day Release Handshake
- **Latest GKE Minor**: Dynamically tracks top-of-tree GCSFuse minor releases (`v<major>.<minor>.x` -> `v<major>.<minor+1>.x`).
- **Older GKE Minors**: Each older GKE minor version is locked **1:1** to a dedicated `v<major>.<minor>_release` branch. Two GKE minor versions must **never** share the same GCSFuse minor stream.
- **30-Day GKE Release Notification Handshake**: When GKE gives 30-day notice for an upcoming GKE minor release:
  1. Lock the outgoing latest GKE minor to its active `v<major>.<minor>_release` branch (enforcing the 4–5 week soak-gated merge rule for functional bug fixes on that branch going forward).
  2. If no newer GCSFuse minor has been cut since the previous GKE minor launched, cut a new `v<major>.<minor+1>_release` branch (for initial release `v<major>.<minor+1>.0`) from `master` for the incoming latest GKE minor.

## Related Skills
- **[GCSFuse Dependency Upgrading Skill](../dep-upgrade/SKILL.md)**: Upgrading Go module dependencies and Go toolchain versions when `govulncheck` detects vulnerabilities on `master`.
- **[GCSFuse Build and Style Verification Skill](../build/SKILL.md)**: Local compilation, formatting, and static analysis verification via `make build`.
- **[GCSFuse Pull Request Creation Skill](../pull-request/SKILL.md)**: Raising a PR targeting `master` to fix vulnerabilities or unit test failures discovered during pre-cut verification.
- **[GCSFuse Release Branch CVE Patching & Backport Skill](../cve-patching/SKILL.md)**: Applying zero-soak CVE remediations and soak-gated functional bug-fix backports onto persistent `v<major>.<minor>_release` branches (for patch releases `v<major>.<minor>.<N+1>`).

```mermaid
flowchart TD
    A["1. Fetch Tags & Remote Release Branches (origin/v*_release)"] --> B["2. Determine Next Minor Branch (v<major>.<minor>_release) & Target Commit"]
    B --> C["3. Verify No Existing Tag/Branch Conflict (v<major>.<minor>_release, v<major>.<minor>.0)"]
    C --> D["4. Run govulncheck on Target Commit (Binary + Module Scan)"]
    D --> E{"Vulnerabilities Found?"}
    E -- "Yes" --> F["Stop Release Cut: Raise PR to Fix CVEs on master"]
    E -- "No (0 CVEs)" --> G["5. Run Build & Unit Tests"]
    G --> H{"Unit Test Failures?"}
    H -- "Yes (Unless User Overrides)" --> I["Stop Release Cut: Raise PR to Fix Unit Tests on master"]
    H -- "No (All Pass)" --> J["6. Cut v<major>.<minor>_release & Push to origin"]
```

---

## Phase 1: Version & Source Commit Discovery

### 1.1 Fetch Latest Tags and Remote Branches
Always start by synchronizing local refs with `origin`:
```bash
git fetch origin --tags --prune
```

### 1.2 Determine the Next Minor Release Branch (`v<major>.<minor>_release`)
1. **If the user explicitly specifies a target minor version** (e.g., `v3.13`, `3.13`, `v3.13.0`, or `v3.13_release`), normalize it to:
   - **Initial release version / tag**: `v<major>.<minor>.0` (e.g., `v3.13.0`)
   - **Persistent minor release branch**: `v<major>.<minor>_release` (e.g., `v3.13_release`)

2. **If the user does not specify a version**, discover the highest existing `<major>.<minor>` release stream from tags and remote `v*_release` branches:
   ```bash
   git tag -l "v*.*.*" --sort=-v:refname | head -n 20
   git branch -r --list "origin/v*_release" --sort=-v:refname | head -n 20
   ```
   Increment `<minor>` by `1` and set the initial patch version to `0`:
   - Latest minor stream: `v<major>.<M>_release` / `v<major>.<M>.*` (e.g., `v3.12_release` / `v3.12.1`)
   - Next minor initial release version: `v<major>.<M+1>.0` (e.g., `v3.13.0`)
   - Next minor release branch: `v<major>.<M+1>_release` (e.g., `v3.13_release`)

3. **Check for Existing Tag or Branch Conflicts**:
   Verify that neither `refs/heads/v<major>.<minor>_release` nor `refs/tags/v<major>.<minor>.0` already exists on `origin` (or locally):
   ```bash
   git ls-remote origin \
     "refs/heads/v<major>.<minor>_release" \
     "refs/tags/v<major>.<minor>.0"
   git tag -l "v<major>.<minor>.0"
   git branch -a --list "*v<major>.<minor>_release"
   ```
   > [!IMPORTANT]
   > If `origin/v<major>.<minor>_release` or `refs/tags/v<major>.<minor>.0` already exists on `origin`, **stop immediately** and ask the user whether they meant to cut the next minor branch (`v<major>.<minor+1>_release`) or apply a patch/backport onto the existing `v<major>.<minor>_release` branch.

### 1.3 Identify and Checkout the Target Source Commit
1. Determine the source commit on `master`:
   - **User-Specified Commit**: If the user provides a specific commit hash `<commit_sha>` (for example, the most recent commit on `master` that passed the periodic continuous E2E test build at code freeze), resolve its full SHA (`git rev-parse <commit_sha>^{commit}`).
   - **Default (`origin/master`)**: If no commit hash is specified, use the tip of `origin/master` (`git rev-parse origin/master`).

2. Verify that the target commit is a valid ancestor of `origin/master`, inspect its metadata, and check it out for pre-cut verification:
   ```bash
   git merge-base --is-ancestor <target_commit> origin/master
   git show -s --format="Commit: %H%nDate:   %ci%nAuthor: %an <%ae>%nTitle:  %s" <target_commit>
   git checkout --detach <target_commit>
   ```

---

## Phase 2: Pre-Cut Vulnerability Gate (`govulncheck`)

Before cutting the minor release branch, verify that the target commit on `master` has **0 known security vulnerabilities**.

### 2.1 Ensure `govulncheck` Is Available
```bash
if ! command -v govulncheck &> /dev/null; then
  go install golang.org/x/vuln/cmd/govulncheck@latest || true
  export PATH=$PATH:$(go env GOPATH)/bin
fi
```

### 2.2 Binary Scan with Target Go Toolchain (`.go-version`)
Running `govulncheck` in source mode uses the host environment's default `go` binary, which can cause false-positive `stdlib` reports if the host Go version differs from `.go-version`. Always verify the compiled binary built with the branch's exact target Go toolchain:
1. Read the target Go version from `[.go-version](../../../.go-version)` and build a temporary `gcsfuse` binary in a temporary directory using that exact toolchain:
   ```bash
   TMP_DIR=$(mktemp -d)
   TARGET_GO=$(tr -d '[:space:]' < .go-version)
   GOTOOLCHAIN=go${TARGET_GO} go build -o "${TMP_DIR}/gcsfuse-release-vulncheck" .
   ```
2. Run `govulncheck` in binary mode:
   ```bash
   govulncheck -mode=binary "${TMP_DIR}/gcsfuse-release-vulncheck"
   ```
3. Remove the temporary directory:
   ```bash
   rm -rf "${TMP_DIR}"
   ```

### 2.3 Source / Module Dependency Scan
Also run `govulncheck` in module/source mode across the repository to check for any non-stdlib module advisories:
```bash
govulncheck ./...
```

### 2.4 Gate Action: If Vulnerabilities Are Found, Fix `master` First
> [!IMPORTANT]
> **Do NOT cut `v<major>.<minor>_release` if `govulncheck` reports any fixable `stdlib` or module vulnerabilities (`Fixed in: vX.Y.Z`).**
> Because a new minor release branch is cut from `master`, `master` itself must be remediated first.

If vulnerabilities are detected:
1. **Stop the release branch creation workflow.** Do not create or push `v<major>.<minor>_release`.
2. Create a remediation branch from `origin/master` (e.g., `git checkout -b fix-cve-master origin/master`).
3. Follow the [GCSFuse Dependency Upgrading Skill](../dep-upgrade/SKILL.md) to upgrade the affected Go modules and/or Go toolchain across the repository (remembering to ask the user for confirmation before upgrading the Go compiler version or performing a major module version upgrade).
4. Re-verify that both `govulncheck -mode=binary` (built with the updated `.go-version` toolchain) and `govulncheck ./...` report `0` vulnerabilities, and verify compilation and unit tests pass.
5. Push the remediation branch and **raise a Pull Request targeting `master`** following the [GCSFuse Pull Request Creation Skill](../pull-request/SKILL.md) (including the `execute-integration-tests` and `execute-perf-test` labels).
6. Report the PR link to the user and stop; the `v<major>.<minor>_release` branch should be cut after the `master` remediation PR is merged.

---

## Phase 3: Pre-Cut Build & Unit Test Gate

Once `govulncheck` passes with `0` vulnerabilities, verify that the target commit compiles cleanly and passes all unit tests.

### 3.1 Run Build and Formatting Checks
Follow the [GCSFuse Build and Style Verification Skill](../build/SKILL.md) to verify compilation and code hygiene:
```bash
make build
```
> [!NOTE]
> In sandboxed or offline environments where `golangci-lint` or network-dependent `go generate` tools cannot run, verify compilation and module tidiness directly via:
> ```bash
> go mod tidy
> go build ./...
> ```

### 3.2 Run Unit Tests
Run the unit test suite across the repository:
```bash
CGO_ENABLED=0 go test ./...
```

### 3.3 Gate Action: If Unit Tests Fail, Raise a PR to Fix `master` First
> [!IMPORTANT]
> **Do NOT cut `v<major>.<minor>_release` if unit test failures are detected**, unless the user has explicitly instructed to ignore or bypass specific test failures.

If one or more unit tests fail (and no user override was given):
1. **Stop the release branch creation workflow.** Do not create or push `v<major>.<minor>_release`.
2. Create a bug-fix branch from `origin/master` (e.g., `git checkout -b fix-unit-tests-master origin/master`).
3. Diagnose and fix the failing unit test(s) or underlying regression, and re-run `make build` and the unit tests until they pass cleanly.
4. Push the fix branch and **raise a Pull Request targeting `master`** following the [GCSFuse Pull Request Creation Skill](../pull-request/SKILL.md).
5. Report the failing test details and PR link to the user and stop; the `v<major>.<minor>_release` branch should be cut once the fix PR is merged into `master`.

---

## Phase 4: Cut and Push the Minor Release Branch

Proceed to this phase **only when both Phase 2 (`govulncheck` = 0 vulnerabilities) and Phase 3 (`go test` = 0 failures) pass cleanly**:

1. **Verify Working Tree Hygiene**:
   Confirm that building and testing produced no untracked files or unstaged diffs:
   ```bash
   git status --porcelain
   ```

2. **Create the Minor Release Branch (`v<major>.<minor>_release`)**:
   Create and check out `v<major>.<minor>_release` pointing to `<target_commit>`:
   ```bash
   git checkout -b v<major>.<minor>_release <target_commit>
   ```
   Confirm that `HEAD` matches `<target_commit>` exactly:
   ```bash
   git rev-parse HEAD
   ```

3. **Push `v<major>.<minor>_release` to `origin`**:
   Push the new persistent minor release branch to `origin` and verify the remote ref:
   ```bash
   git push -u origin v<major>.<minor>_release
   git ls-remote --heads origin v<major>.<minor>_release
   ```

---

## Verification Checklist

Before completing the task, verify the following:
- [ ] Latest tags and `origin/v*_release` branches were fetched and inspected to determine `v<major>.<minor>_release`, and verified that neither `refs/heads/v<major>.<minor>_release` nor `refs/tags/v<major>.<minor>.0` already existed on `origin`.
- [ ] Target commit `<target_commit>` was verified as a valid commit on `origin/master`.
- [ ] `govulncheck -mode=binary` (built with the branch's `.go-version` Go toolchain) and `govulncheck ./...` were executed:
  - If vulnerabilities were found: `v<major>.<minor>_release` was **not** created, and a PR fixing `master` was raised instead.
- [ ] Build (`make build` or `go build ./...`) and unit tests (`go test`) were executed:
  - If unit tests failed (without user override): `v<major>.<minor>_release` was **not** created, and a PR fixing `master` was raised instead.
- [ ] Only if `0` vulnerabilities and `0` unit test failures were found: `v<major>.<minor>_release` was created from `<target_commit>` and pushed to `origin` (unless the user explicitly requested local-only).

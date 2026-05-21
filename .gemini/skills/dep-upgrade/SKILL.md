# GCSFuse Dependency Upgrading Skill

This skill provides a comprehensive, step-by-step workflow for Antigravity agents to identify, upgrade, and verify external dependencies and Go toolchain versions in the GCSFuse repository, specifically targeting the remediation of CVEs and security vulnerabilities.

---

## 1. Parsing Input & Discovering Vulnerabilities

The user initiates this skill by providing one of the following inputs:

### Input Option A: Specific Dependencies to Upgrade
* The user explicitly specifies the module(s) and/or Go version to upgrade (e.g., "Upgrade `google.golang.org/protobuf` to latest", "Upgrade Go to 1.27.0").
* Identify the exact module path(s) and target version(s).

### Input Option B: Bug / Vulnerability Reference
* The user specifies a bug reference, issue ID, or CVE ID (e.g., "Fix security issues mentioned in b/123456" or "Fix CVE-2026-12345").
* **Workflow for Option B**:
  1. Retrieve the full details of the issue, bug description, or external CVE details.
  2. Analyze the issue content or logs to find the vulnerable dependencies (e.g., `golang.org/x/net` before `v0.17.0`) or Go standard library vulnerabilities.
  3. If the bug mentions Go standard library vulnerabilities (e.g., `Found in: net/http@go1.26.2`, `Fixed in: net/http@go1.26.3`), identify the new target Go compiler version.
  4. Produce a list of target packages and their minimum safe versions.

---

## 2. Querying Versions & Major Upgrade Safety Checks

Before executing any dependency upgrades, identify the target versions and evaluate them for major breaking changes.

### Core Directive
* **Default Target**: Always aim to upgrade dependencies to their **latest stable version**.
* **Upgrade Limit**: You may automatically upgrade dependencies to their latest minor or patch versions (e.g., `v1.2.0` -> `v1.5.3`). However, if upgrading to the latest stable version requires a **major version change**, you must halt and ask the user for permission (as detailed in Step 2).

### Step 1: Query Available Versions
List all available stable versions of the target Go module to find the latest stable release:
```bash
go list -m -versions <module_path>
```
*Example:*
```bash
go list -m -versions go.opentelemetry.io/otel/sdk
```

### Step 2: Version Comparison & Safety Checks
Compare the current version of the dependency in `go.mod` against the proposed target/latest version.

* **Major Version Check Rule**:
  If the proposed upgrade involves a **major version change** (e.g., upgrading `github.com/pkg/errors` from `v0.9.1` to `v2.0.0`, or any package whose import path or major version number increments):
  1. **Stop immediately.**
  2. **Alert the user**: Notify them that a major version upgrade is breaking, will require changing module import paths in `.go` files, and could introduce API incompatibilities.
  3. **Ask for permission** before proceeding.
  4. Do NOT proceed with the upgrade unless the user explicitly approves.

* **No Downgrades Rule**:
  If the proposed target version is lower/older than the current version in `go.mod`:
  1. **Stop immediately.**
  2. **Alert the user**: Notify them that the target version represents a downgrade (e.g., going from `v1.5.0` to `v1.3.0`).
  3. **Ask for permission**: Do NOT proceed with a downgrade unless the user explicitly instructs or approves it.

* If the upgrade to the latest version is a minor or patch version change (e.g., `v1.2.0` to `v1.3.5`) and is a strictly higher version, proceed automatically.

---

## 3. Dependency Upgrade Workflow

To upgrade standard Go modules (non-toolchain libraries):

### Guidelines
1. **Focus on Direct Dependencies**: Only upgrade direct dependencies directly. Do not manually upgrade indirect dependencies unless the corresponding direct dependency is also being upgraded. Let `go mod tidy` resolve and update indirect dependencies automatically based on the upgraded direct dependencies.
2. **Lock Version Precisely**: Do not use wildcard upgrades if a specific fixed version is known.
3. **Upgrade Co-dependencies (Peer Packages)**: Upgrading a core library (like `go.opentelemetry.io/otel/sdk`) often requires upgrading peer packages (like `go.opentelemetry.io/otel` or contribution/instrumentation packages like `otelhttp`) to the same version to prevent Skew/API conflicts.

### Commands

#### Case A: Upgrading All/General Dependencies
If you are asked to upgrade all or general dependencies, run the following command to upgrade **only the direct dependencies** to their latest minor/patch versions:
```bash
go get $(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
```
Then, run `go mod tidy` to let Go resolve and update the indirect dependencies and clean up `go.mod` and `go.sum`:
```bash
go mod tidy
```

#### Case B: Upgrading Specific Dependencies
If you are asked to upgrade specific dependencies:
1. Run the `go get` command for the target package(s) and its direct dependencies:
   ```bash
   go get <module_path>@<target_version> [peer_package]@[target_version] ...
   ```
2. Tidy up the module definition files (`go.mod` and `go.sum`):
   ```bash
   go mod tidy
   ```

---

## 4. Go Toolchain & Standard Library Upgrades

Standard library vulnerabilities (e.g., in `net/http` or `html/template`) cannot be patched via `go get`. They require upgrading the Go toolchain/compiler version itself across the entire repository.

### Core Directives for Go Version Upgrades

1. **Target Go Version Rule**:
   * **Default Target**: When upgrading the Go compiler/toolchain (whether for standard library CVE fixes, general feature upgrades, or dependency requirements), **always upgrade to the latest stable available version of Golang**, unless a specific target version is explicitly requested by the user or hardcoded in the bug description.
   * > [!IMPORTANT]
   * > If the upgrade involves a major/minor version change (e.g., moving the repository from Go `1.23.x` to a newer series like `1.26.x`), **upgrade to the latest available stable release of that new series** (e.g., the absolute latest stable release `1.26.3` or newer), instead of just targetting the baseline first release of the series (e.g., `1.26.0`), unless explicitly specified otherwise by the user or the bug.
   * Verify the latest stable version of Go via the web search tool or Go's official download site before finalizing the target version.

2. **Absolute Go Version Consistency**:
   * **Strict Consistency Requirement**: The **exact same** three-part (patch-level) Go version string (e.g., `1.26.3`) must be applied globally across the entire repository. There must be absolutely no version mismatch or inconsistency.
    * **Consistency in [go.mod](../../../go.mod)**: The `go` directive in [go.mod](../../../go.mod) **MUST be updated to match the exact patch-level version string** (e.g., `go 1.26.3`), aligning it completely with [.go-version](../../../.go-version). Do NOT use minor-only shorthand (e.g., `go 1.26`) in [go.mod](../../../go.mod) while using a three-part version string elsewhere.
    * Every single reference to the Go compiler in main builder images, packaging Dockerfiles, presubmit checks, performance suites, and CD scripts must be kept strictly uniform.

### Step 1: Retrieve Current Go Version
Look at the [.go-version](../../../.go-version) file in the root directory, or the `go` directive in [go.mod](../../../go.mod) (e.g., `go 1.26.3`) to identify the old Go version string.

### Step 2: Determine and Query the Target Go Version
1. Identify the minimum safe version required (e.g., if remediating a CVE).
2. Query the web or official resources to find the **latest stable available version** of the Go compiler (within the target major/minor release, or a newer release series if upgrading Go major/minor versions).
3. Set this **latest available version** as the exact target Go version string (e.g., `1.26.3`).

### Step 3: Search the Entire Repository
Search the codebase using `grep` to find all instances of the old Go version string (e.g., `1.26.3`):
```bash
# Example grep query using the search tool:
Query: "1.26.3"
SearchPath: "<workspace_root>"
```

*Key Go Version Reference Locations that MUST match exactly:*
1. **[.go-version](../../../.go-version)**:
   *Located at the root of the repository.*
   * **Mandatory Action**: Ensure this file contains the exact target Go version (e.g., `1.26.3`).
2. **[go.mod](../../../go.mod)**:
   * **Mandatory Action**: Ensure the `go` directive matches the exact patch-level version string (e.g., `go 1.26.3`). Do NOT use two-part versions here.
   ```diff
   -go 1.26.3
   +go 1.27.1
   ```
3. **[Dockerfile](../../../Dockerfile)** (Main Builder Image at root)
4. **Packaging Dockerfiles**:
   * **[Dockerfile](../../../tools/containerize_gcsfuse_docker/Dockerfile)** (Containerization base image)
   * **[Dockerfile](../../../tools/package_gcsfuse_docker/Dockerfile)** (Packaging base image. Make sure to align *both* the builder baseline tag `FROM golang:X.Y.Z` and the download command `goX.Y.Z.linux` to match the exact same version string).
5. **E2E & Integration Test Scripts**:
   * **[e2e_test.sh](../../../tools/cd_scripts/e2e_test.sh)**
   * **[run_e2e_tests.sh](../../../tools/integration_tests/run_e2e_tests.sh)**
   * **[improved_run_e2e_tests.sh](../../../tools/integration_tests/improved_run_e2e_tests.sh)**
6. **Performance & Presubmit Test Scripts**:
   * **[build.sh](../../../perfmetrics/scripts/presubmit_test/pr_perf_test/build.sh)**
   * **[setup.sh](../../../perfmetrics/scripts/read_cache/setup.sh)**
   * **[run_checkpoints.sh](../../../perfmetrics/scripts/ml_tests/checkpoint/Jax/run_checkpoints.sh)**

### Step 4: Replace All References
Update every file identified in Step 3, substituting the old Go version with the new exact target Go version. Ensure that no minor-only (e.g. `1.26` instead of `1.26.3`) or mismatched version strings exist.

### Step 5: Sync and Tidy
Run `go mod tidy` to ensure `go.mod` and `go.sum` are consistent and fully synchronized.

---

## 5. Compilation & Verification

After upgrading dependencies and/or the Go toolchain, verify the integrity and safety of the changes.

### Compile Check
Verify that the entire codebase builds cleanly with no compile-time regressions:
```bash
make build
```

### Run Vulnerability Scan
Verify that all active CVEs are completely cleared. First, ensure `govulncheck` is installed on the system, and install it if it is missing:
```bash
# Check if govulncheck is installed, if not, attempt to install it and expose $(go env GOPATH)/bin in PATH
if ! command -v govulncheck &> /dev/null; then
  go install golang.org/x/vuln/cmd/govulncheck@latest || true
  export PATH=$PATH:$(go env GOPATH)/bin
fi
```
Then attempt to run the vulnerability scan.
> [!IMPORTANT]
> If `govulncheck` fails to install or run (e.g., due to offline environments, missing access to external proxies, or toolchain configuration limits), **do NOT treat this failure as a critical blocker**. Log a warning to the user, note the issue in your summary, and proceed to running the test suite to complete the dependency upgrade.

If the utility is available, run the scan:
```bash
govulncheck ./...
```
*Result:* The scan should report `Your code is affected by 0 vulnerabilities`.

### Run Tests
Verify that the test suite passes successfully with the upgraded dependencies:
```bash
go test ./...
```


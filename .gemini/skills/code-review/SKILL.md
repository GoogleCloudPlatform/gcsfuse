# GCSFuse Code Review & Quality Assurance Skill

This skill provides a highly rigorous, multi-dimensional code-review framework and quality runbook for verifying, optimizing, and generating code in the GCSFuse repository. It is designed to be well-versed with **Effective Go (Golang)**, **Standard Python PEP 8 / telemetries**, and the complex, locking-sensitive multi-threaded **GCSFuse Architectural Anatomy**.

---

## 1. Scope & Core Objectives

Agents must apply this skill when reviewing pull requests, inspecting local diffs, or performing **proactive self-review during code generation**. The objective of this skill is to guarantee:
* **Style & Form Compliance**: Seamless alignment with Effective Go, PEP 8, import organizers, and layout rules.
* **Architectural Hygiene**: Strict package boundaries; ensuring structural operations map precisely onto the correct system layer.
* **Extreme Concurrency Safety**: Verification of GCSFuse’s highly specialized locking invariants (avoiding stalls and deadlocks).
* **Code-Gen & Telemetry Alignment**: Ensuring YAML definitions match auto-generated source code fields.
* **Robust Verification & Coverage**: Broad test coverage, complete error mapping, and correct context propagation.

---

## 2. GCSFuse Architectural Anatomy & Boundaries

Understanding and maintaining the core decoupling in GCSFuse is necessary to prevent structural erosion. The text diagram below traces the path of a typical system-level FUSE request down to the Google Cloud Storage API:

```text
+-------------------------------------------------------------+
|                  Client Application Layer                   |
|         [Operating System (Kernel FUSE Driver)]             |
+-------------------------------------------------------------+
                              |
                              v  (FUSE syscall requests)
+-------------------------------------------------------------+
|                 Filesystem Interface (FS)                   |
|                  [internal/fs package]                      |
|                            |                                |
|                            v  (Dispatches calls)            |
|               [inode.go / dir.go / file.go]                 |
+-------------------------------------------------------------+
                              |
                              v  (Delegates to bucket cache/syncer)
+-------------------------------------------------------------+
|             Extension & Coordination Layer (GCSX)           |
|         [internal/gcsx (BucketManager / Syncer)]            |
|                            |                                |
|                            v  (Interacts with local store)  |
|         [internal/cache (File/Stat/Metadata caches)]        |
+-------------------------------------------------------------+
                              |
                              v  (Resolves remote references)
+-------------------------------------------------------------+
|             Low-level Storage Client Wrapper                |
|       [internal/storage (GCS Storage Client)]               |
|                            |                                |
|                            v  (Performs authenticated net)  |
|               [internal/auth (OAuth / LibAuth)]             |
+-------------------------------------------------------------+
                              |
                              v  (Executes API calls)
+-------------------------------------------------------------+
|                     External Services                       |
|           [Google Cloud Storage JSON/gRPC APIs]             |
+-------------------------------------------------------------+
```

### Architectural Directory Map & Responsibilities

| Directory / Package | Primary Engineering Responsibility | Code Review Warning Signs |
| :--- | :--- | :--- |
| **`cfg/`** | Mount parameter configuration storage and YAML config validation (`validate.go` & `rationalize.go`). | Hardcoded default values; validation logic skipped for new parameters. |
| **`cmd/`** | Entry-points for mounting and running the FUSE background daemon (`cmd/mount.go` & `main.go`). | Bypassing flag parser; initializing global variables directly. |
| **`internal/fs/`** | Core filesystem interface mapping Go FUSE APIs to inode structures. Inodes (`inode.go`), handles (`dir.go`, `file.go`), permissions (`perms/`). | Direct GCS Client/JSON API invocations. Bypassing metadata/stat caches. |
| **`internal/gcsx/`** | Storage extension layer. Coordinates caching, synchronization (`syncer.go`), and handles remote/local file reconciliation. | Placing POSIX specific inode/directory formatting inside the storage layer. |
| **`internal/storage/`** | Wrapper for GCP Storage client. Manages gRPC/HTTP protocol preferences, connection tracing, and retry budgets. | Holding file system locks across slow API calls. Mixing retry loops in FS. |
| **`internal/cache/`** | Houses local stat cache, metadata cache, and local content cache. | Non-thread-safe map accesses; missing thread synchronization; stale cache TTL updates. |
| **`internal/locker/`** | Customized `sync.Mutex` and `sync.RWMutex` wrappers featuring deadlock warning traces at a 5-second interval threshold. | Using raw `sync.Mutex` directly in filesystem handles; holding locks past function exits. |
| **`internal/logger/`** | Unified logger wrapping structured `slog`. Emits traces (`Tracef`), warnings (`Warnf`), errors (`Errorf`), and fatals (`Fatal`). | Using native `fmt.Printf` or Go standard `log` instead of structural gcsfuse logger. |
| **`metrics/`** | Central OTel telemetry management. Metadatas are declared in `metrics.yaml` and auto-generated to `otel_metrics.go`. | Manually adding metric structs to `.go` files instead of editing the generator YAML. |
| **`perfmetrics/`** | Contains continuous, load, and benchmark regression test harnesses written in Go and Python. | Bypassing pre-check cleanups; missing validation assertions; unparameterized variables. |
| **`tools/`** | Packaging pipelines, metrics generators, test suites, container builders, and automation frameworks. | Hardcoded environmental paths; manual compilation dependencies. |

---

## 3. GCSFuse Specific Core Architectural Concepts

Reviewers must have a deep understanding of these custom GCSFuse subsystems and functional constraints:

### A. Flat Bucket Directory Emulation (Implicit vs. Explicit Directories)
Google Cloud Storage is a flat key-value object store; directories are virtual abstractions emulated by GCSFuse. Differentiate between:
* **Implicit Directories**: Activated via the mount configuration option `--implicit-dirs`. GCSFuse emulates directory structures dynamically by searching the namespace for matching sub-path objects (e.g. mapping virtual parent folders on the fly without zero-byte GCS directory files ending in a trailing slash `/`).
* **Explicit Directories**: Rely on real zero-byte objects ending with a trailing slash `/` as explicit directory directory indicators in backing buckets.
* **Review Concerns**:
  * Ensure directory creation/deletion methods correctly add or delete directory placeholders.
  * Ensure folder renames (`Rename` operations) handle Hierarchical Namespace (HNS) enabled buckets atomically via direct layout endpoints, whereas flat buckets require concurrently copy-recreate every child object (which introduces high egress transfer and eventual consistency latency).

### B. Append Composing Optimization & Garbage Collection
To optimize high-volume sequential write appends:
* **Sequential Append Composing**: If a file append is launched that exceeds the `AppendThreshold`, instead of rewriting the entire object to GCS, GCSFuse writes intermediate logs as individual temporary fragments (prefixed with `TmpObjectPrefix`). It then triggers a compose operation (`ComposeObjects`) on top of the original GCS target, before dropping temporary objects.
* **Residual Temp Objects & GC**: If a compose-append loop is interrupted or aborted by a client exception, raw temporary objects will leak.
* **Review Concerns**:
  * Verify that mutations to file upload systems do not disrupt compose chains.
  * Understand that setup workflows call a background task (`garbageCollect`) that periodically sweeps backing buckets to purge obsolete temporary objects (prefixed with `TmpObjectPrefix`).

### C. MRD Connections & Short Reads on Concurrently Appended Files
For kernel-optimized readers like the [KernelMRDReader](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/gcsx/kernel_readers/kernel_mrd_reader.go) which run concurrent Multi-Range Downloader (MRD) streams:
* **Concurrently Appended Files**: When a file is appended to from the same local mount, the OS kernel dynamically registers the expanded file size, letting subsequent client threads issue reads past the *old* file boundaries.
* **The gRPC OutOfRange Gotcha**: The active MRD connection, created prior to the append, holds stale metadata pointing to the old size. Issuing a read past the old size limits thus returns a gRPC `OutOfRange` error status.
* **Review Concerns**:
  * Verify that gRPC errors are filtered via the dedicated short-read classifier (`isShortRead`). If the read returns less than the requested buffer size and wraps a gRPC `OutOfRange` error, it must trigger an active MRD connection recreation (`RecreateMRD`) and launch a read retry loop from the target offset to safely pull the expanded data block.

---

## 4. Go Engineering Standards (Effective Go Alignment)

Review all Golang contributions according to the following strict guidelines:

### A. Core Style & Receiver Conventions
* **Automatic Layout Formatting**: All modifications **MUST** align with `goimports` and `go fmt`. The orchestrating linter checker is verified via `make build`.
* **Receiver Naming Standards**:
  * Receiver names must be short (usually one or two letters) and matching the type name.
  * Receiver names must be strictly uniform across every single method of that structure.
  * > [!WARNING]
  * > Do **NOT** use general receiver terms such as `me`, `this`, or `self`.
  * **Correct**: `func (bm *bucketManager) GetBucketName(...)`
  * **Incorrect**: `func (this *bucketManager) GetBucketName(...)`
* **Variable Scoping**: Minimize scope blocks for logical clarity and compiler safety. Use short-form initialization `if val, err := lookup(); err == nil { ... }` where appropriate.

### B. Critical Concurrency & Locking Safety
GCSFuse filesystem operations are concurrently accessed by multiple user threads via FUSE hooks. Lock hygiene is crucial to prevent deadlocks and network stalls.

* **Unified Locker Wrapper**:
  * All structure-protecting locks **MUST** utilize `locker.New` (for mutual exclusion) or `locker.NewRW` (for read-write) from the `github.com/googlecloudplatform/gcsfuse/v3/internal/locker` package.
  * This is critical because the custom locker tracks invariant checking and triggers a debug trace if a writer lock is held for more than **5 seconds**.
* **Narrow Lock Scope Rule**:
  * > [!IMPORTANT]
  * > **NEVER hold a mutex lock while executing slow operations, disk I/O, channel transmissions, or remote GCS API calls.**
  * All network connections, reads, writes, and bucket lookups are high-latency operations. Holding a filesystem lock over these delays suspends other OS requests, leading to severe stalls.
  * **Remedy**: Cache state locally, release the lock, perform the high-latency I/O operation, re-acquire the lock, and verify that variables have not been modified by concurrent operations in the interim (Double-checked locking).
* **Parallel High-QPS Read Locking Optimization (ML Training Checkpoints)**:
  * High QPS training (such as concurrent PyTorch/JAX checkpointers accessing the same file handle in parallel) is highly sensitive to lock contention.
  * *Concurrency Design Check*: Review how [random_reader.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/gcsx/random_reader.go) prevents bottleneck serialization. 
  * Ensure that local cache read hits use shared read locks (`fileCacheMu.RLock()`), seeks/classification parameters utilize lock-free atomic tracking registers (`atomic.Uint64`, `atomic.Int64`), and multi-range downloader (MRD) streams bypass range reader locking entirely to operate concurrently on joint handles.
* **Safe Locker Declarations**:
  ```go
  import "github.com/googlecloudplatform/gcsfuse/v3/internal/locker"

  type CacheManager struct {
      mu   locker.Locker
      data map[string]string
  }

  func NewCacheManager() *CacheManager {
      cm := &CacheManager{
          data: make(map[string]string),
      }
      // Initialize locker with customized name and optional invariant check callback
      cm.mu = locker.New("CacheManager.mu", cm.checkInvariants)
      return cm
  }
  ```
* **Thread-Safe Maps**: Standard Go maps are not concurrent-safe. Always encapsulate map mutations and lookups inside a lock sequence or use the specialized `sync.Map` when appropriate.
* **Pre-allocating Slices**: When creating a slice whose final capacity is known in advance, pre-allocate space to avoid continuous array re-allocations and garbage collection pressure:
  * **Correct**: `items := make([]FileInfo, 0, count)`
  * **Incorrect**: `var items []FileInfo` followed by iterative `append`.

### C. Robust Context-Propagation
* **Pass Contexts Verbally**: Do **NOT** store contexts as variables inside long-lived structures. Instead, pass `context.Context` explicitly as the very first argument of all internal API routines.
* **Trace Cancellation Integrity**: Ensure that contexts are correctly monitored during heavy loops (e.g. streaming file reads). Call `ctx.Err()` to verify active viability, gracefully closing stale or cancelled connections.
* **Avoid Context Stash**:
  * **Correct**: `func (s *StorageClient) Read(ctx context.Context, req *Request) ...`
  * **Incorrect**: Storing a context inside a long-lived struct that outlives the associated caller thread's execution.

### D. Errors as Values & Boundaries
* **Strict Error Analysis**: Always verify the returned errors from every functional execution. Bypassing verification via `_ = call()` is strictly forbidden unless there is a well-documented reason.
* **Explicit Wrapper Contexts**: Utilize `%w` syntax formatting to wrap intermediate error stack layers, maintaining root visibility:
  * `return fmt.Errorf("stat object metadata: %w", err)`
* **POSIX System Map Errors**: In the filesystem layer (`internal/fs/`), errors are converted into kernel errnos (e.g., `syscall.ENOENT`, `syscall.EEXIST`, `syscall.EIO`, `syscall.ENOTDIR`). Review the error boundaries to ensure correct POSIX code integration, ensuring standard operations return exact matching failure conditions to GCSFuse mounting handlers.

---

## 5. Python Quality Standards

Review Python files (particularly performance regression tests under `perfmetrics/scripts/` and tools) to ensure they adhere to modern standards:

### A. PEP 8 & Design Hygiene
* **Code Formatting**: Format code strictly according to PEP 8 standards. Verify line length stays within **80** or **100** characters max.
* **No Wildcard Imports**: Bypassing namespace separation via `from module import *` is strictly forbidden. Explicitly name the required imports or import the module scope.
* **Clean Resource Hooks**: Always open file pointers and network client channels using the `with` block pattern, guaranteeing cleanup upon scope exit:
  * **Correct**:
    ```python
    with open("metrics_output.txt", "w") as f:
        f.write(report_payload)
    ```

### B. Typing Coverage
* **Type Annotations**: Proactively add Python type hints to signatures, parameter blocks, and structural elements. Type hints prevent logical errors and make test scripts much easier to read:
  ```python
  from typing import List, Dict

  def calculate_percentile(latencies: List[float], target: float) -> Dict[str, float]:
      # Logic block...
  ```

### C. Robust Exception Guarding
* **No Bare Exceptions**: Blanket trapping via `except:` is strictly prohibited because it intercepts system signals (such as SIGINT/SIGTERM), making tests difficult to terminate.
* **Specify & Log Exceptions**: Always target explicit exceptions (e.g. `except ValueError:`, `except subprocess.CalledProcessError:`) and preserve backtrace contexts in error streams via `logging.exception(...)`:
  ```python
  try:
      execute_shell_run()
  except subprocess.CalledProcessError as e:
      logging.exception("Subprocess execution failed with status: %d", e.returncode)
      raise
  ```

### D. Mock Safety in Testing
* **Prevent State Leakage**: When writing Python test scripts, verify that any mock objects (`unittest.mock.patch` or pytest fixtures) are properly terminated upon test completion.
* **Prefer Context Mocking**: Prefer context managers for temporary patches rather than global class annotations, protecting subsequent benchmark tests from picking up stub values:
  ```python
  with patch('os.environ.get', return_value='1024'):
      # Targeted system state assertion...
  ```

---

## 6. Telemetry & Configuration Generators

GCSFuse uses a code generation pattern for adding and validating parameters or telemetries. **Do NOT directly hardcode new config parameters or metric structs in `.go` source files.**

### A. Configuration Updates Workflow
If a feature requires new CLI flags or YAML parameters:
1. Define the parameter settings directly inside the central metadata schema: [cfg/params.yaml](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/cfg/params.yaml).
2. Write appropriate parsing constraints, types, defaults within the schema definitions.
3. Validate and map the logic in [cfg/validate.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/cfg/validate.go) and [cfg/rationalize.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/cfg/rationalize.go).
4. Run the dynamic configuration generator (or trigger via Makefile):
   ```bash
   make build # Internally calls go generate to build cfg/config.go and cfg/config_test.go
   ```
5. Confirm that the generated outputs compile correctly without manual adjustments.

### B. Telemetry & Custom Metrics Workflow
If a feature tracks new operations, latencies, or counts:
1. Declare the metric definition inside the metric tracking schema: [metrics/metrics.yaml](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/metrics/metrics.yaml).
2. Configure attributes, descriptions, histograms boundaries, and units.
3. > [!CAUTION]
   > **Mandatory Metric Prefixing Gotcha**:
   > OpenTelemetry metrics are filtered under [otelexporters.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/monitor/otelexporters.go#L45).
   > Every single metric name **MUST** start with one of the approved namespace prefixes: `fs/`, `gcs/`, `file_cache/`, `buffered_read/`, `grpc.`, or `read/`.
   > If you define a metric that is NOT prefixed by one of these namespaces, the OTel exporter **will silently drop the metric** during stream aggregation!
4. Run the generator to output the OpenTelemetry struct bindings:
   ```bash
   make build # Internally triggers the generator to output metrics/otel_metrics.go
   ```
5. Update logical blocks using the newly populated metrics under the autogenerated metric wrapper class:
   ```go
   import "github.com/googlecloudplatform/gcsfuse/v3/metrics"

   // Example incremental call
   metrics.RecordFileCacheReadBytesCount(ctx, bytesRead, cacheHit, readType)
   ```

---

## 7. Localized Integration Testing Map

Verify that changes to file system, caching, or mounting operations have corresponding E2E and integration validation suites. Integration tests are self-contained packages placed under:

📁 **[tools/integration_tests/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests)**

Verify that functional changes are tested under the matching suite directory:

| Component Under Review | Target Integration Directory | Test Verification Objective |
| :--- | :--- | :--- |
| **Flat Bucket Directory Emulation** | [implicit_dir/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/implicit_dir) & [explicit_dir/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/explicit_dir) | Ensure parent paths and recursive listings emulate directories correctly on the fly. |
| **Stat/File/Metadata Caches** | [read_cache/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/read_cache) & [negative_stat_cache/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/negative_stat_cache) | Validate stat TTL expiries and read caching hits/misses. |
| **Mounting Controls & CLI parameters** | [mounting/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/mounting) & [flag_optimizations/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/flag_optimizations) | Verify that CLI arguments map onto backend behaviors under direct operating system mounts. |
| **Parallel Concurrent Reads/Writes** | [concurrent_operations/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/concurrent_operations) & [streaming_writes/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/streaming_writes) | Ensure file handles perform concurrent operations safely with zero data loss or synchronization deadlocks. |
| **Telemetry / Monitoring Exports** | [monitoring/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/monitoring) | Ensure Prometheus and Cloud Monitoring correctly export all custom metric streams. |
| **Permissions, Attributes, Read-Only** | [readonly/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/readonly) & [readonly_creds/](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/tools/integration_tests/readonly_creds) | Validate authentication scopes, token source bounds, and file access modes. |

---

## 8. Step-by-Step Code Review Workflow

When performing a code review, follow this systematic evaluation process:

```text
  +-------------------------------------------------+
  |            1. Run Static Validation             |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |          2. Review Module Architecture          |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |      3. Trace Concurrency and Locking Safety    |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |          4. Verify Contexts and Errors          |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |            5. Audit Code Generators             |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |           6. Evaluate Test Coverage             |
  +-------------------------------------------------+
                           |
                           v
  +-------------------------------------------------+
  |          7. Output Code Review Report           |
  +-------------------------------------------------+
```

### Step 1: Run Static Validation
* Ensure the local environment aligns with [.go-version](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/.go-version).
* Execute the validation targets from the root of the workspace:
  ```bash
  make build
  ```
* Ensure formatting checks, static tests (`go vet`), and linters pass cleanly.

### Step 2: Review Module Architecture
* Verify that structural dependencies stay within correct package boundaries.
* Ensure files under filesystem structures (`internal/fs/`) do not make direct GCS REST/gRPC API calls. Ensure low-level details are kept cleanly abstracted behind storage hooks in `internal/storage/`.

### Step 3: Trace Concurrency and Locking Safety
* Audit all locker additions. Verify that every lock initialization maps to `locker.New` or `locker.NewRW`.
* Trace the critical paths of newly introduced locks:
  * Check that lock acquisition scope is minimized.
  * Confirm that a lock is **NEVER held** during I/O block transfers, remote calls, channel operations, or wait condition blocks.
  * Check for locking hazards in high-volume, multi-threaded parallel read sequences (e.g. ensuring locks do not serialise concurrent streams).
  * Verify lock releases under defer lines: `defer mu.Unlock()`.

### Step 4: Verify Contexts and Errors
* Ensure contexts are passed downstream as function arguments, never stashed within internal struct state.
* Confirm that errors are wrapped contextually (`%w`) and that filesystems-visible failures resolve to valid POSIX return codes (e.g. `syscall.ENOENT` or `syscall.EEXIST`).

### Step 5: Audit Code Generators & Telemetry Boundaries
* If configuration parameters or custom telemetry structures are modified, ensure they are specified in `cfg/params.yaml` or `metrics/metrics.yaml`, respectively.
* **Audit Metric Names**: Double check that any new telemetry matches prefix filters (`fs/`, `gcs/`, `file_cache/`, `buffered_read/`, `grpc.`, `read/`) so that they are not drop-aggregated.
* Run `make build` and verify that generated structs (`cfg/config.go`, `metrics/otel_metrics.go`) compile correctly.

### Step 6: Evaluate Test Coverage
* Verify that new code is covered by appropriate unit tests.
* Ensure that cache operations are validated inside isolated execution targets under the `internal/cache/` subdirectory.
* Trace and locate matching E2E suites under `tools/integration_tests/` and verify validation.
* Run unit tests to verify:
  ```bash
  make test
  ```

### Step 7: Output Code Review Report
Construct a structured, actionable review summary using the template in Section 10.

---

## 9. Proactive Self-Review for Code Generation

When generating code, designing a feature, or drafting refactor scripts, apply this skill proactively. Prevent bugs before they are committed by holding a self-review loop:

1. **Architecture Target Verification**: Define the target package boundaries before writing code.
2. **Lock Contour Outlining**: Map lock contours on paper/scratchpads to prove zero deadlocks will occur. Keep lock boundaries narrow, especially for ML checkpointing/training access paths.
3. **Draft Context Flow**: Verify that context values flow downstream uninterrupted.
4. **Define Config & Metrics Upfront**: Write changes to `cfg/params.yaml` or `metrics/metrics.yaml` *first* (checking prefix formatting boundaries), and then let `make build` generate the boilerplate code automatically.
5. **Simultaneous Test Creation**: Write standard testing patterns side-by-side with feature development, preserving isolated, reliable, and multi-threaded checks under `tools/integration_tests/`.

---

## 10. Unified Code Review Report Template

When publishing feedback, use this structure. Place file reference links using `file://` formatting (e.g., [fs.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/fs.go#L123-L135)).

````markdown
# GCSFuse Code Review Report

## 1. Executive Summary & Verdict

* **Verdict**: `[LGTM | Needs Revision | Blocked]`
* **Overview**: A high-level, 2-3 sentence technical summary of the proposed changes and overall quality.

---

## 2. Key Logical & Safety Findings

This section highlights critical issues related to concurrency, directory emulations, appends logic, context leaks, or performance.

### A. Critical Concurrency & Locking Hazards
* **Issue**: *[e.g., Mutex lock held over remote GCS API call in file_handle.go]*
* **Reference**: [file_handle.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/file_handle.go#L145)
* **Impact**: *[e.g., Causes severe stalls and threads deadlock if network delays occur]*
* **Recommended Remedy**:
  ```go
  // Recommended pattern change:
  - h.mu.Lock()
  - defer h.mu.Unlock()
  - resp, err := h.gcsClient.Read(ctx, request)
  + h.mu.Lock()
  + cacheState := h.state
  + h.mu.Unlock()
  +
  + resp, err := h.gcsClient.Read(ctx, request)
  +
  + h.mu.Lock()
  + defer h.mu.Unlock()
  + // Re-validate state and update...
  ```

### B. Directory Emulation Boundaries & File Systems Decoupling
* **Issue**: *[e.g., Virtual directory listing returns stale listings under implicit directories]*
* **Reference**: [dir.go](file:///usr/local/google/home/kislayk/gitproj/gcsfuse/internal/fs/dir.go#L80)
* **Impact**: *[e.g., Inodes cache entries mismatch under heavy multi-threading and dynamic parent updates]*
* **Recommended Remedy**: *[e.g. Ensure stat cache expiries invalidate emulated virtual folders]*

---

## 3. Style, Telemetry & Code-Gen Integrity

Verify style compliance, prefix validation, type hints, and that configuration / telemetry updates follow the proper generation pipelines.

* **Formatting Checks (`make build`)**: `[Pass | Fail]` - *[Specify if auto-formatting, imports sorting, or dependency hygiene changes require a stage commit]*
* **Code-Gen Schema Matching**:
  * Config Parameters: `[Aligned | Needs Generation]` - *[Verify if modifications to params.yaml require building config.go]*
  * Telemetry OTel: `[Aligned | Needs Generation]` - *[Verify metrics.yaml and otel_metrics.go alignment]*
  * OTel Prefixes Check: `[Passed | Failed - SILENT DROP RISK DETECTED]` - *[Verify that custom metrics start with fs/, gcs/, file_cache/, buffered_read/, grpc., or read/]*
* **Minor Code Cleanliness Remarks**:
  * [x] **Receiver Naming**: Uniform receiver naming conventions followed? `[Yes | No]`
  * [x] **Type Hints (Python)**: Subprocess/automation scripts fully typed? `[Yes | No]`
  * [x] **Error Wrapping**: Used `%w` correctly in nesting? `[Yes | No]`

---

## 4. Test Verification & Verification Checks

Review testing coverage, isolated runs, and validation checks.

- [ ] **Unit Tests Coverage**: *[Check if new scenarios have corresponding coverage in testing files (e.g. file_test.go)]*
- [ ] **Integration Suites Coverage**: *[Specify if integration tests under tools/integration_tests/ are created/updated]*
- [ ] **Static Verification**: *[Verify that the test suite was executed locally via `go test ./...` and `make test` without errors]*
- [ ] **Mock Sandboxing**: *[Confirm that python/unit tests correctly scope patches and mock environments to prevent state leakages]*
````

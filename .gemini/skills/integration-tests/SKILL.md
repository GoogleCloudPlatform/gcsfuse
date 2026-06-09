---
name: gcsfuse-integration-testing
description: Step-by-step runbook for creating, extending, and verifying integration tests in the GCSFuse repository.
---

# GCSFuse Integration Testing Skill

This skill provides a comprehensive, step-by-step runbook for Antigravity agents to create, extend, structure, and verify integration tests within the GCSFuse repository's [tools/integration_tests](../../../tools/integration_tests) directory. Following this framework ensures that all tests adhere to uniform setup configurations, GKE compatibility paths, safe flag override systems, and clean teardowns.

---

## Phase 1: Pre-Implementation Planning & Design

Before writing any new integration tests or modifying files, walk through the clarification, research, and technical design steps to prepare your testing components.

### 1.1 Pre-Implementation Clarifications
Engage the developer to confirm basic requirements:
- [ ] **Package Scope**: Is this a brand-new test package being added, or are we extending an existing suite?
- [ ] **GKE Compatibility**: Should this suite run on GKE? If yes, it must support GKE container environments by handling the `GKEMountedDirectory` execution path correctly.
- [ ] **Bucket Type Support**: Which Google Cloud Storage bucket types are compatible with the test cases? (Choose all that apply: `flat`, `hns`, `zonal`). Note that different configs can be run on different sets of compatible bucket types.
- [ ] **Mount Configurations**: What kind of mounts are expected for GCSFuse? (e.g., Standard Static Mounting, Dynamic Mounting, Only-Dir Mounting, etc.). While static mounting is the standard baseline, clarify if the test package needs to execute across a multi-mount lifecycle.
- [ ] **Config Struct Update**: If this is a new package, clarify if you need to add a new slice configuration item to the global `Config` struct in [tools/integration_tests/util/test_suite/config.go](../../../tools/integration_tests/util/test_suite/config.go) to map the new YAML properties correctly.

### 1.2 Proactive Research & Cross-Referencing
> [!TIP]
> **Proactive Reference Cross-Checking**:
> Do not limit yourself strictly to basic dummy templates! You are highly encouraged to explore and inspect other active packages under [tools/integration_tests](../../../tools/integration_tests) (such as [read_cache](../../../tools/integration_tests/read_cache), [operations](../../../tools/integration_tests/operations), or [negative_stat_cache](../../../tools/integration_tests/negative_stat_cache)) to review real-world design, study current code styles, and cross-reference helper functions. Proactively reiterate your design choices and ensure your target test suite aligns perfectly with established repository standards before proposing changes!

### 1.3 Technical Action Plan Guidelines
When creating a new integration test suite, it is highly recommended to draft a structured technical action plan to scope your work before implementation:
- **Test Scenario Design**: Summarize the behavior being tested, the preconditions, and the assertions required.
- **Target Files Identification**: List the exact target paths to be created or modified (e.g., package files, the configuration entry structure in [tools/integration_tests/util/test_suite/config.go](../../../tools/integration_tests/util/test_suite/config.go), and the shell registration list in [tools/integration_tests/improved_run_e2e_tests.sh](../../../tools/integration_tests/improved_run_e2e_tests.sh)).
- **Compatibility & Mount Mapping**: Map out the explicit list of compatible bucket types (`flat`, `hns`, `zonal` / `rapid`) and mount behaviors (static, dynamic, only-dir) your test suite must cover to ensure clean compliance checks.

---

## Phase 2: Core Architecture & Environment Mechanics

All integration tests are driven by central configurations and dynamic environmental mappings that resolve variables according to where they are run.

### 2.1 YAML Configuration & Config Flow
All modern integration tests in GCSFuse rely on a centralized, shared configuration file: [tools/integration_tests/test_config.yaml](../../../tools/integration_tests/test_config.yaml).

> [!IMPORTANT]
> **No Backward-Compatibility Support Policy**:
> Do NOT build fallback logic to parse command flags directly if the YAML configuration is absent. The repository is migrating fully to the configuration file model. Enforce parsing of the YAML config via the `test_suite.ReadConfigFile` flow in all cases.

### 2.2 Dynamic Bucket Discovery & Config Filtering
During execution, the test suite dynamically determines GCS environment capabilities to filter what configurations should be run:
1. **Bucket Metadata Query**: At startup, `setup.TestEnvironment(ctx, cfg)` queries the target bucket attributes via GCS APIs.
2. **Bucket Type Identification**: Returns a string type representing:
   - `zonal` (if `attrs.LocationType == "zone"`)
   - `hns` (if Hierarchical Namespace is enabled: `attrs.HierarchicalNamespace.Enabled`)
   - `flat` (if standard bucket architecture)
3. **Compatibility Matching**: When preparing the flag options, `setup.BuildFlagSets(*cfg, bucketType, t.Name())` loops through the `configs` list in the YAML file and filters them based on their compatibility maps.
4. **Target Execution**: Only configurations where `compatible[bucketType]` is marked as `true` (and whose `run` properties match the active Go test suite/run filters) are scheduled to execute.

### 2.3 Environment Placeholders Index
Central GKE and runner variables are written as text placeholders within [tools/integration_tests/test_config.yaml](../../../tools/integration_tests/test_config.yaml). They are dynamically resolved at runtime depending on GKE pod mounts or local flag rewrites:

| Placeholder | Meaning & Role | Resolution Flow |
| :--- | :--- | :--- |
| `${MOUNTED_DIR}` | The GKE/container pre-mounted point target directory. | Parsed directly via `setup.MountedDirectory()` on GKE. |
| `${BUCKET_NAME}` | The active target GCS bucket name. | Injected from CLI flags or `setup.TestBucket()`. |
| `${ONLY_DIR}` | Sub-directory inside the bucket for static Only-Dir subpath tests. | Populated via the `setup.OnlyDirMounted()` environment helpers. |
| `${PROFILE_LABEL}` | GCP Cloud Profiler label/version to differentiate profiling runs. | Extracted and populated via `setup.ExtractServiceVersionFromFlags()`. |
| `${PROFILE_SERVICE_NAME}` | GCP Cloud Profiler service identifier tag. | Extracted and populated via `setup.CloudProfilerServiceNameFromFlags()`. |
| `${BILLING_PROJECT}` | The GCP Billing Project ID for Requester Pays validation. | Parsed and set dynamically via `setup.BillingProject()`. |
| `${KEY_FILE}` | The Service Account credentials JSON filepath. | Loaded and set dynamically via `setup.KeyFile()`. |

### 2.4 GKE vs. Local/GCE Path Overwrites
Integration tests configure temporary workspaces, cache directories, and log targets via flag sets (e.g., `--cache-dir=/gcsfuse-tmp/...` and `--log-file=/gcsfuse-tmp/...`):
- **On GKE Environments**: GCSFuse runs within standard Kubernetes volume containers where storage mounts are static and already configured. No path rewriting is necessary.
- **On Local/GCE Developer Environments**: Raw static paths could cause write permission conflicts or target root violations.
- **Path Overwrites**: To support local environments safely, the test runner invokes `setup.OverrideFilePathsInFlagSet(cfg, setup.TestDir())`. This recursively replaces all occurrences of the `/gcsfuse-tmp` flag paths with safe, dynamically generated workspace directories under the host system's sandboxed environment (`setup.TestDir()`).

### 2.5 Only-Dir Path Translations
In Only-Dir mounting configurations, GCSFuse mounts only a single directory prefix inside the GCS bucket rather than the entire bucket root:
- The path is saved in the environment using `setup.SetOnlyDirMounted(onlyDirName + "/")`.
- In individual test cases and storage API checks, ALWAYS use `setup.GetBucketAndObjectBasedOnTypeOfMount(object)` instead of directly targeting the raw bucket name. This dynamically prefixes the target directory key and translates paths correctly so assertions do not fail.

---

## Phase 3: Package Conventions & TestMain Flow

To keep GCSFuse integration tests clean and uniform, all folders and files must adhere to standard splitting conventions, lifecycles, and safe utility routines.

### 3.1 Directory Splits & Layout
* **`setup_test.go` or `<package_name>_test.go`**: Contains the main `TestMain(m *testing.M)` function, the shared package `env` struct, common package constants (e.g., `testDirName`), and shared helper functions used by more than one test file in the package.
* **`<scenario>_test.go`**: Separate scenario files should hold specific testing suites using `testify/suite` (`suite.Suite`).
* **Package-Scope Environment Struct**: To prevent global variable pollution across concurrent runs, declare a localized private `env` struct containing pointers to the clients, target configs, and state strings:
  ```go
  type env struct {
      mountFunc            func(*test_suite.TestConfig, []string) error
      mountDir             string
      rootDir              string
      storageClient        *storage.Client
      storageControlClient *control.StorageControlClient
      ctx                  context.Context
      testDirPath          string
      cfg                  *test_suite.TestConfig
      bucketType           string
  }
  var testEnv env
  ```
* **Header Convention**: Every single Go file **MUST** be prefixed with the appropriate Google LLC copyright header, using the **current calendar year (2026)**.

### 3.2 Unified TestMain Execution Flow
When executing, the integration package's `TestMain(m *testing.M)` function MUST sequentially handle:
1. **Flag Parsing**: Parse initial setup flags using `setup.ParseSetUpFlags()`.
2. **Read YAML Configurations**: Load and unmarshal the centralized config block:
   ```go
   config := test_suite.ReadConfigFile(setup.ConfigFile())
   testEnv.cfg = &config.<PackageName>[0]
   ```
3. **Environment Discovery**: Detect context and discover bucket attributes:
   ```go
   testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)
   ```
4. **Client Setup**: Construct storage and control plane clients (e.g., `client.CreateStorageClientWithCancel(...)`, `client.CreateControlClientWithCancel(...)`).
5. **GKE Check & Short-Circuit (FIRST STEP)**: GKE environments run containerized tests against pre-mounted directories. Before doing any local folder creation or local mounts, check if GKE is active and exit directly:
   ```go
   if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
       testEnv.mountDir, testEnv.rootDir = testEnv.cfg.GKEMountedDirectory, testEnv.cfg.GKEMountedDirectory
       os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
   }
   ```
6. **Local GCSFuse Preparation**: If not on GKE, construct the target workspace:
   ```go
   setup.SetUpTestDirForTestBucket(testEnv.cfg)
   setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())
   testEnv.mountDir, testEnv.rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory
   ```
7. **Sequential Multi-Mount Execution Loop**: Run the standard Go test suite (`m.Run()`) sequentially across target mount models (static, dynamic, or only-dir mounting) depending on compatibility, keeping track of failure success codes.
8. **Safe Bucket Cleanup**: When all tests finish, perform systematic, deep bucket-level cleanup:
   ```go
   setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
   ```

### 3.3 Safe Flag & Regex Parsing
If your test needs to parse parameters from complex list structures or flag strings:
- Flag inputs often contain space and comma separations when multiple flags are joined together in command lines (e.g. `--cloud-profiler-service-name=foo,--log-severity=trace`).
- Your regular expression **MUST** explicitly support both assignment modes (`=` or whitespace) and exclude trailing items using comma and whitespace breakers.

#### Safe Parser Regex Pattern:
```go
func CloudProfilerServiceNameFromFlags(flags []string) string {
    // Regex matches --cloud-profiler-service-name=value or --cloud-profiler-service-name value
    // [=\s] permits both equal sign and space breakers.
    // [^,\s]+ terminates the match safely at a comma, space, or string boundary.
    re := regexp.MustCompile(`--cloud-profiler-service-name[=\s]([^,\s]+)`)
    for _, flagSet := range flags {
        matches := re.FindStringSubmatch(flagSet)
        if len(matches) > 1 {
            return matches[1]
        }
    }
    return "gcsfuse" // Fallback default value
}
```

---

## Phase 4: Writing Test Scenarios & Anti-Flakiness Rules

Designing integration test cases requires a disciplined, robust approach to assert structures, async wait methods, unique naming keys, and central helpers reuse.

### 4.1 The Arrange-Act-Assert (AAA) Model
While the **Arrange-Act-Assert (AAA)** pattern is traditionally a strict best practice for unit testing, it is highly recommended to follow this style in integration tests as well wherever possible:
- **Arrange**: Set up the state, directories, target buckets, test files, and options necessary for this specific case. Proactively invoke folder helpers like `setup.SetupTestDirectory(...)` here.
- **Act**: Execute GCSFuse actions, mount operations, read-write procedures, or call the specific methods under test.
- **Assert**: Validate results against expected outcomes. Use descriptive Assertions from the `testify/assert` framework (e.g., `assert.NoError`, `assert.Contains`, `assert.ErrorContains`).

### 4.2 Anti-Flakiness Rule 1: No Raw Sleeps (Use `RetryUntil` Polling)
Using raw sleep calls (e.g., `time.Sleep(30 * time.Second)`) assuming an asynchronous operation, metadata flush, file write, or size check "should have finished by now" is a major source of flaky tests and execution slowdowns.
- **Strict Requirement**: Instead of a hardcoded sleep, implement active polling with an explicit timeout.
- **Unified Framework Helper**: Use the built-in, generic [RetryUntil](../../../tools/integration_tests/util/operations/validation_helper.go#L100-L157) retry harness located in [tools/integration_tests/util/operations/validation_helper.go](../../../tools/integration_tests/util/operations/validation_helper.go). It executes an evaluation block at a safe frequency until a successful (nil error) state is returned or the timeout deadline is reached.

#### Polling Example (Waiting for File Size Update):
```go
// Correct pattern: Poll the file state actively with RetryUntil instead of using time.Sleep()
targetFile := path.Join(testEnv.testDirPath, "file.txt")
expectedSize := int64(1024)

sizeVal := operations.RetryUntil(testEnv.ctx, s.T(), 500*time.Millisecond, 10*time.Second, func() (int64, error) {
    fi, err := os.Stat(targetFile)
    if err != nil {
        return 0, err // Retry on stat errors
    }
    if fi.Size() != expectedSize {
        return 0, fmt.Errorf("expected size %d, got %d", expectedSize, fi.Size()) // Retry if size is not correct yet
    }
    return fi.Size(), nil // Success! Returns target value and exits loop
})
```

### 4.3 Anti-Flakiness Rule 2: Enforce Unique Resource Naming
GCSFuse integration packages are scheduled in parallel streams and share physical testing buckets:
- Hardcoding static object names or directories (e.g., `targetFile := "data.txt"`) will cause immediate collision issues, resource state contamination, and test failures.
- **Strict Requirement**: ALWAYS dynamically generate folder and file suffixes by appending a secure random identifier: `setup.GenerateRandomString(5)`.
- **Usage**:
  ```go
  // Generate distinct directory names per suite
  s.testDir = testDirName + setup.GenerateRandomString(5)
  testEnv.testDirPath = setup.SetupTestDirectory(s.testDir)

  // Generate distinct file paths inside the folder
  uniqueFile := path.Join(testEnv.testDirPath, "item_" + setup.GenerateRandomString(5) + ".txt")
  ```

### 4.4 Correct Error Handling & Avoiding False Negatives
To guarantee test suite integrity and prevent silent failures (**false negatives**), all test scenarios MUST enforce strict, robust error handling:

- **Strictly Check Every Error**: NEVER discard or swallow returned errors using the blank identifier `_` (e.g., writing `_ = file.Close()` or `_ = os.Remove(path)` is strictly forbidden). If an operation returns an error, you MUST verify/assert it.
- **Fail Fast on Fatal Preconditions (Assert vs. Require)**:
  - **Require (`require.NoError`, `require.FailNow`)**: Use for fatal preconditions, suite arrangements, setups, and mount commands. If GCSFuse fails to mount or target directories cannot be created, the scenario should fail immediately to avoid downstream clutter and false test statuses.
  - **Assert (`assert.NoError`, `assert.True`)**: Use for target validation steps and assertions where standard validation tracking is sufficient.
- **Validate Side Effects Explicitly**: Do not assume operations succeeded just because they returned a nil error. Proactively verify the side effects (e.g., when asserting file writes, call `os.Stat` to confirm the file actually exists and verify that its contents, sizes, and permissions match the expectations).
- **Cleanup and Close Safety**: In teardown functions and test helpers, check and report failures from storage bucket cleanup APIs or OS file closing methods so that errors are not silently ignored during test completion.

### 4.5 Code Reuse & Core Helpers Reference
Do not duplicate filesystem operations, mounting boilerplate, or client construction functions. Before implementing a new test utility, inspect the central helper modules:
- **[tools/integration_tests/util/setup/setup.go](../../../tools/integration_tests/util/setup/setup.go)**: Holds common setup routines (`SetupTestDirectory`, `SetUpTestDirForTestBucket`, `BuildFlagSets`, `CleanupDirectoryOnGCS`, `ReplaceOrAppendFlag`).
- **[tools/integration_tests/util/operations](../../../tools/integration_tests/util/operations)**: Prefer filesystem wrappers over raw `os` methods to get automatic failure hooks (`operations.CreateDirectory`, `operations.CreateFileWithContent`, `operations.WriteToFile`, `operations.ReadFile`).
- **[tools/integration_tests/util/client](../../../tools/integration_tests/util/client)**: Central storage API client construction, control APIs, and mock object creators.

> [!WARNING]
> **Avoid Deprecated Methods**:
> Avoid utilizing obsolete legacy functions still left over from the flag-to-config migration process (e.g., `SetUpTestDirForTestBucketFlag`, `RunTestsForMountedDirectoryFlag`). Proactively identify and skip outdated methods.

---

## Phase 5: Formatting, Verification & PR Creation

Once you have completed the test package coding steps, you **MUST** follow these specific quality control and validation workflows before completing your turn.

### 5.1 Code Formatting Checklist
Execute standard Go mod/formatting checkers from the root of the workspace. Ensure all files are cleanly formatted and the imports tree resolves perfectly without raising formatting differences:
```bash
goimports -w .
go fmt ./...
go mod tidy
git diff --exit-code --name-only
```

### 5.2 CI/CD Pipeline Package Registration
When introducing a new integration test suite, you **MUST** register the package in the central test orchestrator shell script:
- **Target File**: [tools/integration_tests/improved_run_e2e_tests.sh](../../../tools/integration_tests/improved_run_e2e_tests.sh)
- **Action**: Append the name of the new test package to the `TEST_PACKAGES_COMMON` array. This step is critical to guarantee that the new suite is automatically picked up by Kokoro presubmits and scheduled across parallel splits.

### 5.3 Developer Test Execution & Cloudtop Alerts
1. **Manual Testing Command**: Invite the user to manually compile and verify the new tests across compatible GCS environments (e.g., `flat`, `hns`, `zonal`):
   ```bash
   BUCKET_NAME=<compatible_bucket_name> GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/<package_name> -p 1 --integrationTest -v --config-file="<workspace_root>/tools/integration_tests/test_config.yaml"
   ```
   *(Note: Remind the user to substitute `<compatible_bucket_name>` with their GCS-configured test bucket matching the specific compatible type they are testing).*
2. **Dynamic Cloudtop Warning Block**: Include a mandatory warning alert to advise the user on environment limits:
   > [!WARNING]
   > **Cloudtop/Jetski Local Environment Limits**:
   > Executing full integration tests directly inside your local Cloudtop terminal or a Jetski virtual session environment might fail or hang. This is caused by Cloudtop-specific network proxy intercept blocks, system FUSE mounting constraints, or credential scope restrictions. If failures occur locally on your Cloudtop environment, please transfer your branch/run the tests inside your designated GCP Linux VM/sandbox instead.

### 5.4 PR Creation & Skill Delegation
After formatting verification passes successfully:
1. Generate a concise, high-quality summary of changes and pull request fields (Title with Conventional Commits structure, e.g., `test(operations): add directory operations test`).
2. **Invoke GCSFuse PR Skill**: Delegate the PR publishing task to GCSFuse's central **Pull Request Creation Skill** by loading and strictly executing the steps laid out in [.gemini/skills/pull-request/SKILL.md](../../../.gemini/skills/pull-request/SKILL.md).

---

## Phase 6: Reference Dummy Templates

To keep this runbook compact and highly readable, the standard boilerplate code templates have been split into standalone, syntax-highlighted reference Go files. Refer to these targets when designing your new testing components:

* **Unified Lifecycle & TestMain Setup**: [templates/setup_test.go](./templates/setup_test.go) (Lifecycle management, GKE pod environment discovery, dynamic client setups, sequential multi-mount testing loop, and full post-suite bucket cleanup)
* **Individual Scenario Suite Runner**: [templates/dummy_feature_test.go](./templates/dummy_feature_test.go) (Testify framework suite patterns, active AAA test case designs, automatic fail-safe GCSFuse logs preservation, and loop execution across config flags)

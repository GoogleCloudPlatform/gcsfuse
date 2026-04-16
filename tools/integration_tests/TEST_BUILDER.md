# GCSFuse Integration Test Command Builder

Use this guide to construct the command to run end-to-end (E2E) integration tests for GCSFuse.

## 1. Select a Package

Select the package you want to test by checking the box (or just noting the name).

- [ ] `benchmarking`: Tests for benchmarking performance.
- [ ] `concurrent_operations`: Tests for operations happening concurrently.
- [ ] `emulator_tests`: Tests that run against an emulator.
- [ ] `explicit_dir`: Tests for explicit directory handling.
- [ ] `gzip`: Tests for gzip compression support.
- [ ] `implicit_dir`: Tests for implicit directory handling.
- [ ] `interrupt`: Tests for interrupt handling.
- [ ] `kernel_list_cache`: Tests for kernel list cache behavior.
- [ ] `list_large_dir`: Tests for listing large directories.
- [ ] `local_file`: Tests for local file operations.
- [ ] `log_content`: Tests for log content verification.
- [ ] `log_rotation`: Tests for log rotation behavior.
- [ ] `managed_folders`: Tests for managed folders feature.
- [ ] `monitoring`: Tests for monitoring and metrics.
- [ ] `mount_timeout`: Tests for mount timeout scenarios.
- [ ] `mounting`: Tests for basic mounting functionality.
- [ ] `negative_stat_cache`: Tests for negative stat cache.
- [ ] `operations`: Tests for general file system operations.
- [ ] `read_cache`: Tests for read cache behavior.
- [ ] `read_large_files`: Tests for reading large files.
- [ ] `readonly`: Tests for read-only mounts.
- [ ] `readonly_creds`: Tests with read-only credentials.
- [ ] `rename_dir_limit`: Tests for rename directory limits.
- [ ] `stale_handle`: Tests for stale file handles.
- [ ] `streaming_writes`: Tests for streaming writes.
- [ ] `write_large_files`: Tests for writing large files.

*Note: `util` is excluded as it is a helper package.*

## 2. Select Options

Select additional options for the test run.

- [ ] **Verbose Output**: Add `-v` to see detailed logs.
- [ ] **Short Mode**: Add `-short` to skip long-running tests.
- [ ] **Specific Test**: Run a specific test function by name by adding `-run <test_name>`.
- [ ] **Timeout**: Adjust the timeout (default is 20m).
- [ ] **Mounted Directory**: Add `--mountedDirectory=<path>` to run tests against a pre-mounted directory.
- [ ] **Test Installed Package**: Add `--testInstalledPackage` to run tests on the package pre-installed on the host machine (instead of building a new one).
- [ ] **Presubmit Run**: Add `--presubmit` to indicate a presubmit run (skips some tests).
- [ ] **TPC Endpoint**: Add `--testOnTPCEndPoint` to run tests on TPC endpoint.

## 3. Construct the Command

Fill in the variables in the template below based on your selections.

### Template

```bash
export TEST_PACKAGE_NAME=<selected_package>
export TEST_BUCKET_NAME=<your_gcs_bucket_name>

GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/$TEST_PACKAGE_NAME/... -p 1 --integrationTest --testbucket=$TEST_BUCKET_NAME --timeout=20m <additional_options>
```

### Examples

#### Run all tests in `operations` package
```bash
export TEST_PACKAGE_NAME=operations
export TEST_BUCKET_NAME=my-test-bucket

GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/$TEST_PACKAGE_NAME/... -p 1 -short --integrationTest -v --testbucket=$TEST_BUCKET_NAME --timeout=20m
```

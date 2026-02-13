# Dev Guide

## Contributing to GCSFuse

**If you're interested in contributing a new feature, enhancement, or bug fix,
please open a GitHub issue to discuss your proposal with the GCSFuse team first
before making the contribution.** Make sure to describe the use case (what are
you trying to solve and why?), and the proposed solution in the GitHub issue.

**Note:** We make no guarantees that contribution requests are accepted, or will
be rolled out as a Generally Available (GA) feature

Rollout of any new feature should be done in 2 phases:

**1. Experimental Phase:**

* New features should initially be introduced behind
  an [experimental flag](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/dev_guide.md#adding-a-new-cli-flagconfig-to-gcsfuse).
  These flags should be prefixed with `--experimental-` and marked as hidden in
  the help output.
* This signals to users that the feature is in development and may change or be
  removed.
* **Testing Requirements**:
    * Thorough testing is a crucial requirement for any new features added to
      GCSFuse even during the experimental phase. This helps ensure the
      feature's stability, identify potential issues early, and prevent
      regressions.
    * [Add unit tests](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/dev_guide.md#testing-of-new-features#how-to-write-unit-tests)
      for all new code written for the experimental feature. This includes
      testing edge cases, boundary conditions, and error handling.
    * [Add composite tests](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/dev_guide.md#testing-of-new-features#how-to-write-composite-tests)
      to validate GCSFuse's end user functionality while isolating the testing
      to the GCSFuse code itself. This allows for faster
      and more controlled testing without relying on a real Google Cloud Storage
      bucket.
    * [Add end-to-end tests](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/dev_guide.md#testing-of-new-features#how-to-write-end-to-end-tests)
      for complete feature flows, simulating real-world use
      with an actual Google Cloud Storage bucket.
    * For more details on testing best practices, please refer to
      the [testing guide](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/dev_guide.md#testing-of-new-features)
      below.
      After the feature development is complete and the code is merged to the
      mainline, the feature will roll out as an experimental feature in the next
      GCSFuse release with the feature disabled by default.

* After the feature development is complete and the code is merged to the
  mainline, the feature will roll out as an experimental feature in the next
  GCSFuse release with the feature disabled by default.

* Developers should open a new GitHub issue (
  see [example](https://github.com/GoogleCloudPlatform/gcsfuse/issues/793))
  to track the feature's progress towards GA readiness. This issue should
  outline the motivation for the feature, its design, and any known
  limitations.

**2. General Availability (GA) Rollout:**

* Once the feature has been thoroughly vetted in test environments and approved
  by the GCSFuse team, it will be rolled out for general availability. **Note
  that experimental features are not guaranteed to become generally available.**

* The GCSFuse team will manage the GA rollout. This involves:
    * Removing the experimental prefix from the flag.
    * Updating
      the [Google Cloud documentation](https://cloud.google.com/storage/docs/gcsfuse-cli)
      to include the new feature.

## Adding a new CLI Flag/Config to GCSFuse

Each param in GCSFuse should be supported as both as a CLI flag as well as via
config file; the only exception being *--config-file* which is supported only as
a CLI flag for obvious reasons. Please follow the steps below in order to make a
new param available in both the modes:

1. Declare the new param in
   [params.yaml](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/params.yaml#L4).
   Refer to the documentation at the top of the file for guidance.
2. Run `make build` from the project root to generate the required code.
3. Add validations on the param value in
   [validate.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/validate.go)
4. If there is any requirement to tweak the value of this param based on other
   param values or other param values based on this one, such a logic should be
   added in
   [rationalize.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/rationalize.go).
   If we want that when an int param is set to -1 it should be replaced with
   `math.MaxInt64`, rationalize.go is the appropriate candidate for such a
   logic.
5. Add unit tests in
   [validate_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/validate_test.go)
   and
   [rationalize_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/rationalize_test.go)
6. Add one test-case in
   [root_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/root_test.go)
   to verify that the flag works - no need to test for different scenarios; a
   single test for a happy-case is sufficient.

## Testing of new features

GCSFuse maintains 3 types of tests for each feature - unit tests, composite
tests and end-to-end tests.

### How to write unit tests

When adding new features to GCSFuse, make sure to write thorough unit tests.
Here's how:

1. Use
   the [stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify/assert)
   library: This makes your tests more readable and easier to write.
2. *Keep tests clear and focused:* Each test should focus on a single behavior.
   Don't be afraid of some repetition to improve readability.
3. **Structure tests with Arrange-Act-Assert (AAA):** This helps organize your
   tests and makes them easier to understand.
    - **Arrange:** Set up the test data and environment.
    - **Act:** Run the code you're testing.
    - **Assert:** Check that the results are what you expect.
4. Call setup code directly in each test: This prevents tests from interfering
   with each other.
5. Write specific assertions: Avoid generic, parameterized assertions.
6. Keep tests short: Long tests are harder to understand and debug.

### How to write composite tests

1. Composite tests are written
   in [fs_test](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/fs/fs_test.go)
   package and use
   the [fake bucket](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/storage/fake/bucket.go)
   implementation to emulate GCS.
2. When introducing a new file system level feature, composite tests must be
   added. (
   Ref: [Sample test file](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/fs/hns_bucket_test.go))
3. **Structure tests with Arrange-Act-Assert (AAA):** This helps organize your
   tests and makes them easier to understand.
    - **Arrange:** Set up the test data and environment.
    - **Act:** Run the code you're testing.
    - **Assert:** Check that the results are what you expect.

### How to write end-to-end tests

End-to-end (e2e) tests are crucial for ensuring the correctness and reliability
of GCSFuse. These tests verify the complete functionality of GCSFuse by
interacting with a real Google Cloud Storage bucket. This guide explains how to
write end-to-end tests for GCSFuse.

1. **Adding tests:**
    - Navigate to
      the ["integration_tests"](https://github.com/GoogleCloudPlatform/gcsfuse/tree/master/tools/integration_tests)
      directory which contains all GCSFuse end-to-end tests.
    - If you are adding a new feature (e.g., a new flag, a new file operation),
      create a new package directory for your tests. Add corresponding package
      name entry to
      our [test script](https://github.com/GoogleCloudPlatform/gcsfuse/blob/a4fb11ad4d424fcc07727dc18e06561ffd8e2467/tools/integration_tests/run_e2e_tests.sh#L66)
      so that these tests can be run as part of our CI pipeline.
      Ref PR: https://github.com/GoogleCloudPlatform/gcsfuse/pull/2144
    - If you are modifying existing behavior, modify the corresponding tests
      within the relevant package.

2. **Structure tests with Arrange-Act-Assert (AAA):** This helps organize your
   tests and makes them easier to understand.
    - **Arrange:** Set up the test data and environment.
    - **Act:** Run the code you're testing.
    - **Assert:** Check that the results are what you expect.
3. **Run Tests:** GCSFuse end-to-end tests are run using the following command.
    - To run all tests in the package:
       ```shell
       export TEST_PACKAGE_NAME=<Enter package name here. For eg: operations>
       export TEST_BUCKET_NAME=<Enter your bucket name here.>
       GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/$TEST_PACKAGE_NAME/... -p 1 -short --integrationTest -v --testbucket=$TEST_BUCKET_NAME --timeout=60m 
       ```
    - To run a particular test add `-run` flag to the command:
       ```shell
       export TEST_NAME=<Enter particular test name you want to run>
       export TEST_PACKAGE_NAME=<Enter package name here. For eg: operations>
       export TEST_BUCKET_NAME=<Enter your bucket name here.>
       GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/$TEST_PACKAGE_NAME/... -p 1 -short --integrationTest -v --testbucket=$TEST_BUCKET_NAME --timeout=60m -run $TEST_NAME
       ```
4. **Run all tests as pre-submit:** Existing GCSFuse end-to-end tests can be run
   as a pre-submit check by adding the `execute-integration-tests` and `kokoro:run` label to your
   pull request. Ask one of your assigned code reviewers to apply this label,
   which will trigger the tests. Your reviewer will share any test failure
   details on the pull request.

5. **Discuss test scenarios:** If you are unsure about how to test a specific
   feature or have questions about scenarios to test, please feel free to open a
   [discussion thread](https://github.com/GoogleCloudPlatform/gcsfuse/discussions)
   with GCSFuse team. We're here to help!

## Dummy I/O Mode for Performance Testing

Dummy I/O mode simulates read operations without transferring data from Cloud Storage, allowing you to isolate and measure GCSFuse overhead independent of network latency.

**Note:** 
- Currently supports read operations only.
- Hidden feature for developers/performance engineers.
- Metadata operations (list, stat) remain real.
- Reads return zeros without fetching data.

### Use Cases

**✅ When to use:**
- Micro-optimizations in GCSFuse read flow
- Isolate performance from network latency
- Profile GCSFuse CPU/memory usage without network noise
- Benchmark different kernel configurations

**❌ When NOT to use:**
- Real-world performance testing (network latency is critical)
- Data correctness validation
- Production workloads

### Configuration

```bash
--enable-dummy-io
--dummy-io-reader-latency=150ms    # Simulates reader creation latency
--dummy-io-per-mb-latency=20ms     # Simulates per-MB read from stream latency
```


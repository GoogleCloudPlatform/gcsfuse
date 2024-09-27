# Dev Guide

## Adding a new param in GCSFuse

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
   and composite tests in
   [config_validation_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/config_validation_test.go),
   [config_rationalization_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/config_rationalization_test.go)
   and
   [root_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/root_test.go)

## Adding new features to GCSFuse

**If you're interested in adding a new feature, please open a GitHub issue to
discuss your proposal with the GCSFuse team first before making the
contribution.**

Rollout of any new feature should be done in 2 phases:

**1. Experimental Phase:**

* New features should initially be introduced behind an experimental flag. These
  flags should be prefixed with `--experimental-` and marked as hidden in the
  help output.
* This signals to users that the feature is in development and may change or be
  removed.
* After the feature development is complete, developers should open a GitHub
  issue (
  see [example](https://github.com/GoogleCloudPlatform/gcsfuse/issues/793)) to
  track the feature's progress towards production readiness. This issue should
  outline the motivation for the feature, its design, and any known limitations.

**2. Production Rollout:**

* Once the feature has been thoroughly vetted in test environments for at least
  a week and approved by the GCSFuse team, it will be rolled out for general
  availability. The GCSFuse team will own this part.
* This involves:
    * Removing the experimental prefix from the flag.
    * Updating the Google Cloud documentation to include the new feature.

## Testing of new features

GCSFuse maintains 3 types of tests for each feature - unit tests, composite
tests and e2e tests.

### How to write unit tests

When adding new features to GCSFuse, make sure to write thorough unit tests.
Here's how:

1. Use
   the [stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify/assert)
   library: This makes your tests more readable and easier to write.
2. *Keep tests clear and focused:* Each test should focus on a single behavior.
   Don't be afraid of some repetition to improve readability.
3. Structure tests with Arrange-Act-Assert (AAA): This helps organize your tests
   and makes them easier to understand.
   *Arrange:* Set up the test data and environment.
   *Act:* Run the code you're testing.
   *Assert:* Check that the results are what you expect.
4. Call setup code directly in each test: This prevents tests from interfering
   with each other.
5. Write specific assertions: Avoid generic, parameterized assertions.
6. Keep tests short: Long tests are harder to understand and debug.

### How to write composite tests

1. Composite tests are written
   in [fs_test](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/fs/fs_test.go)
   package and use a mock bucket.
2. When introducing a new file system level feature, composite tests must be
   added. (
   Ref: [Sample test file](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/fs/hns_bucket_test.go))
3. Structure tests with Arrange-Act-Assert (AAA): This helps organize your tests
   and makes them easier to understand.
   *Arrange:* Set up the test data and environment.
   *Act:* Run the code you're testing.
   *Assert:* Check that the results are what you expect.

### How to write e2e tests

E2e tests are crucial for verifying the end-to-end functionality of GCSFuse.
In GCSFuse, we maintain a dedicated directory for writing e2e tests. These tests
reside in
the ["integration_tests"](https://github.com/GoogleCloudPlatform/gcsfuse/tree/master/tools/integration_tests)
directory, but they are actually end-to-end (e2e) tests. This ensures that as
new features are introduced to GCSFuse, we rigorously test them by adding
corresponding e2e tests. E2E tests differ from composite tests as these tests
run against an actual GCSFuse bucket.






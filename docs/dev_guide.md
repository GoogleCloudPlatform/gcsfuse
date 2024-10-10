# Dev Guide

## Adding a new param in GCSFuse

Each param in GCSFuse should be supported as both as a CLI flag as well as via
config file; the only exception being *--config-file* which is supported only as
a CLI flag for obvious reasons. Please follow the steps below in order to make a
new param available in both the modes:

1.  Declare the new param in
    [params.yaml](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/params.yaml#L4).
    Refer to the documentation at the top of the file for guidance.
1.  Run `make build` from the project root to generate the required code.
1.  Add validations on the param value in
    [validate.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/validate.go)
1.  If there is any requirement to tweak the value of this param based on other
    param values or other param values based on this one, such a logic should be
    added in
    [rationalize.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/rationalize.go).
    If we want that when an int param is set to -1 it should be replaced with
    `math.MaxInt64`, rationalize.go is the appropriate candidate for such a
    logic.
1.  Add unit tests in
    [validate_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/validate_test.go)
    and
    [rationalize_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cfg/rationalize_test.go)
    and composite tests in
    [config_validation_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/config_validation_test.go),
    [config_rationalization_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/config_rationalization_test.go)
    and
    [root_test.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/cmd/root_test.go)
1.  Add the name of flag with underscores in
    [mount_gcsfuse/main.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/mount_gcsfuse/main.go)

> NOTE: Flags marked as `experimental` or private are subject to change at any time and should be used with caution.

# Known Issues

This document lists known issues and bugs in GCSFuse, their impact, and the releases in which they were fixed.

## Resolved Issues

### [v2.11.*] - metrics: Input/output error when metrics are enabled

*   **Issue:** Input/output error when metrics are enabled.
*   **Impact:** Applications may receive input/output errors from GCSFuse mounts when metrics are enabled.
*   **Fixed in:** v2.12.0
*   **Reference:** [#3870](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3870)

## Active Issues

### gRPC: GCSFuse hangs while repeatedly writing over a file

*   **Issue:** GCSFuse can hang when repeatedly writing to the same file using gRPC.
*   **Impact:** Applications that perform frequent writes to the same file may experience deadlocks.
*   **Reference:** [#2784](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2784)

### gRPC: Input/Output error while writing repeatedly over the existing file

*   **Issue:** An "Input/Output error" can occur when repeatedly writing to an existing file using gRPC.
*   **Impact:** This can lead to data corruption or incomplete writes.
*   **Reference:** [#2783](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2783)

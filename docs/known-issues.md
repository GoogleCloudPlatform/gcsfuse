# Known Issues

This document lists known issues and bugs in GCSFuse, their impact, and the releases in which they were fixed.

## Active Issues

| Issue | Impact | Reference |
| :--- | :--- | :--- |
| GCSFuse can hang when repeatedly writing to the same file using gRPC. | Applications that perform frequent writes to the same file may experience deadlocks. | [#2784](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2784) |
| An "Input/Output error" can occur when repeatedly writing to an existing file using gRPC. | This can lead to data corruption or incomplete writes. | [#2783](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2783) |
| `Error: GPG check FAILED` while installing GCSFuse on newer OS with strict cryptographic policies |This interrupts GCSFuse installation on certain OS (e.g., Rocky Linux 10, Red Hat Enterprise Linux 10). | [#3874](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3874) |


## Resolved Issues

| Issue | Affected Versions | Fixed in | Reference |
| :--- | :--- | :--- | :--- |
| Input/output error when metrics are enabled. Applications may receive input/output errors from GCSFuse mounts when metrics are enabled. | v2.11.* | v2.12.0 | [#3870](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3870) |
| Incorrect gcs/reader_count and gcs/download_bytes_count metrics. gcs/reader_count and gcs/download_bytes_count are unreliable before v3.5. | v3.4 and older | v3.5 | [#3895](https://github.com/GoogleCloudPlatform/gcsfuse/pull/3895)
| Points must be written in order. One or more of the points specified had an older start time than the most recent point | v3.4 and older | v3.5 | [#3895](https://github.com/GoogleCloudPlatform/gcsfuse/pull/3923)
| One or more points were written more frequently than the maximum sampling period configured for the metric. | v3.4 and older | v3.5 | [#3895](https://github.com/GoogleCloudPlatform/gcsfuse/pull/3923)
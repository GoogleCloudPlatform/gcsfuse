# Known Issues

This document lists known issues and bugs in GCSFuse, their impact, and the releases in which they were fixed.

## Active Issues

| Issue | Impact | Reference |
| :--- | :--- | :--- |
| GCSFuse can hang when repeatedly writing to the same file using gRPC. | Applications that perform frequent writes to the same file may experience deadlocks. | [#2784](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2784) |
| An "Input/Output error" can occur when repeatedly writing to an existing file using gRPC. | This can lead to data corruption or incomplete writes. | [#2783](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2783) |

## Resolved Issues

| Issue | Affected Versions | Fixed in | Reference |
| :--- | :--- | :--- | :--- |
| Input/output error when metrics are enabled. | Applications may receive input/output errors from GCSFuse mounts when metrics are enabled. | v2.12.0 | [#3870](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3870) |

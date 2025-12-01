# Known Issues

This document lists known issues and bugs in GCSFuse, their impact, and the releases in which they were fixed.

## Active Issues

| Issue | Impact | Reference |
| :--- | :--- | :--- |
| GCSFuse can hang when repeatedly writing to the same file using gRPC. | Applications that perform frequent writes to the same file may experience deadlocks. | [#2784](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2784) |
| An "Input/Output error" can occur when repeatedly writing to an existing file using gRPC. | This can lead to data corruption or incomplete writes. | [#2783](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2783) |
| `Error: GPG check FAILED` while installing GCSFuse on newer OS with strict cryptographic policies |This interrupts GCSFuse installation on certain OS (e.g., Rocky Linux 10, Red Hat Enterprise Linux 10). | [#3874](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3874) |
| GCSFuse does not use machine-type passed from GKE CSI Driver (from v3.4.0 onwards) | Optimized GCSFuse configs for high-performance machines will not be applied when using the GCSFuse GKE CSI driver.<br>**Workaround**: When using the GKE CSI driver, pass the machine type explicitly through gcsfuse mount-options: `machine-type=<MACHINE_TYPE>`. | [#3799](https://github.com/GoogleCloudPlatform/gcsfuse/pull/3799) [#4083](https://github.com/GoogleCloudPlatform/gcsfuse/issues/4083) |


## Resolved Issues

| Issue | Affected Versions | Fixed in | Reference |
| :--- | :--- | :--- | :--- |
| Input/output error when metrics are enabled. | Applications may receive input/output errors from GCSFuse mounts when metrics are enabled. | v2.12.0 | [#3870](https://github.com/GoogleCloudPlatform/gcsfuse/issues/3870) |

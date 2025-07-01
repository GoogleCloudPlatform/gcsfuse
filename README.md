[![codecov](https://codecov.io/gh/GoogleCloudPlatform/gcsfuse/graph/badge.svg?token=vNsbSbeea2)](https://codecov.io/gh/GoogleCloudPlatform/gcsfuse)

# Current status

Cloud Storage FUSE continues to evolve with significant enhancements in v2 and v3, and is Generally Available and
supported by Google starting with v1.0, Cloud Storage FUSE is Generally Available and supported by Google, provided that
it is used within its documented supported applications, platforms, and limits. Support requests, feature requests, and
general questions should be submitted as a support request via Google Cloud support channels or via
GitHub[here](https://github.com/GoogleCloudPlatform/gcsfuse/issues).

Cloud Storage FUSE is open source software, released under the
[Apache license](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/LICENSE).

## Cloud Storage Fuse v3 features

### Streaming Writes

Streaming writes is the new default write path that uploads data directly to Google Cloud Storage (GCS) as it is
written.
The previous write path temporarily staged the entire write in a local file, uploading to GCS on close or fsync.
This reduces both latency and disk space usage, making it particularly beneficial for large, sequential writes, such as
checkpoint writes, which can be up to _**40% faster**_, as observed in training runs.
See [streaming writes](https://github.com/googlecloudplatform/gcsfuse/blob/master/docs/semantics.md#with-streaming-writes)
for more details.

### File Cache Parallel Downloads (Default)

Parallel downloads uses multiple workers to download a file in parallel using the file cache directory as a prefetch
buffer. We recommend using parallel downloads for single-threaded read scenarios that load large files such as model
serving and checkpoint restores, with up to _**9x faster model load times**_.
See [Using Parallel Downloads](https://cloud.google.com/storage/docs/cloud-storage-fuse/file-caching#configure-parallel-downloads)
for more details.

### Automatic Optimization for High-Performance Machine Types

GCSFuse now automatically optimizes its configuration when running on specific high-performance Google Cloud machine
types to maximize performance for demanding workloads and effectively utilize the machine's capability. Manually set
values at the time of mount will override these defaults.

## Cloud Storage FUSE v2 features

Cloud Storage FUSE V2 provides important stability, functionality, and performance enhancements.

### File Cache
The file cache allows repeat file reads to be served from a local, faster cache storage of choice, such as a Local SSD, Persistent Disk, or even in-memory /tmpfs. The Cloud Storage FUSE file cache makes AI/ML training faster and more cost-effective by reducing the time spent waiting for data, with up to _**2.3x faster training time and 3.4x higher throughput**_ observed in training runs. This is especially valuable for multi epoch training and can serve small and random I/O operations significantly faster. The file cache feature is disabled by default and is enabled by passing a directory to 'cache-dir'. See [overview of caching](https://cloud.google.com/storage/docs/gcsfuse-cache) for more details. 

# ABOUT
## What is Cloud Storage FUSE?

Cloud Storage FUSE is an open source FUSE adapter that lets you mount and access Cloud Storage buckets as local file systems. For a technical overview of Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse.

## Cloud Storage FUSE for machine learning

To learn about the benefits of using Cloud Storage FUSE for machine learning projects, see https://cloud.google.com/storage/docs/gcsfuse-integrations#machine-learning.

## Limitations and key differences from POSIX file systems

To learn about limitations and differences between Cloud Storage FUSE and POSIX file systems, see https://cloud.google.com/storage/docs/gcs-fuse#differences-and-limitations.

## Pricing for Cloud Storage FUSE

For information about pricing for Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse#charges.

# CSI Driver

Using the [Cloud Storage FUSE CSI driver](https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver), users get the declarative nature of Kubernetes
with all infrastructure fully managed by GKE in combination with Cloud Storage. This CSI
driver relies on Cloud Storage FUSE to mount Cloud storage buckets as file systems on the
GKE nodes, with the Cloud Storage FUSE deployment and management fully handled by GKE, 
providing a turn-key experience.

# Support

## Supported operating system and validated ML frameworks 

To see supported operating system and ML frameworks that have been validated with Cloud Storage FUSE, see [here](https://cloud.google.com/storage/docs/gcs-fuse#supported-frameworks-os).

## Getting support

You can get support, submit general questions, and request new features by [filing issues in GitHub](https://github.com/GoogleCloudPlatform/gcsfuse/issues). You can also get support by using one of [Google Cloud's official support channels](https://cloud.google.com/support-hub).

See [Troubleshooting](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/troubleshooting.md) for common issue handling.


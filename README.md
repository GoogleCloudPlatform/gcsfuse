# Current status

Cloud Storage FUSE is a stable release, supported as a Pre-GA offering by Google 
provided that it is used within its supported applications and platforms, 
and limits documented. Support requests, feature requests, and general questions
should be submitted as a support request via Google Cloud support channels or 
via GitHub [here](https://github.com/GoogleCloudPlatform/gcsfuse/issues). 
It is highly recommended that customers restrict their use to 
non-production workloads until Cloud Storage FUSE becomes Generally Available.


Cloud Storage FUSE is open source software, released under the 
[Apache license](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/LICENSE).

# ABOUT
## What is Cloud Storage FUSE?

Cloud Storage FUSE is a FUSE adapter that lets you mount and access Cloud 
Storage buckets as a local filesystem, so applications can read and write Cloud
Storage objects using standard file system semantics. Cloud Storage FUSE can be 
run anywhere with connectivity to Cloud Storage, including Google Kubernetes
Engine, Google Compute Engine VMs or on-premises systems.

With Cloud Storage FUSE, you can  take advantage of the scale, affordability, 
throughput, and simplicity that Google Cloud Storage provides, while maintaining
compatibility with applications that use filesystem semantics without having to 
refactor applications to use native Cloud Storage APIs

## Cloud Storage FUSE for machine learning

Cloud Storage is a common choice for users looking to store training data, 
models, checkpoints, and logs for machine learning projects in Cloud Storage
buckets. With Cloud Storage FUSE, objects in Cloud Storage buckets can be 
accessed as files mounted as a local filesystem. Cloud Storage FUSE is 
officially supported when used for workloads based on PyTorch and TensorFlow 
Machine Learning frameworks.

Cloud Storage FUSE provides the following benefits:
- Mount and access Cloud Storage buckets with standard file system semantics, 
providing a portable journey for ML workloads that eliminates application 
refactoring costs
- From training, to inference, to checkpointing, leverage the native high scale 
and performance of Cloud Storage to run your ML workloads at scale
- Start training jobs quickly by providing compute resources with direct access
to data in Cloud Storage, rather than having to copy it down to a local
filesystem instance.

Cloud Storage FUSE can be deployed as a regular Linux package, but is also
available as part of a managed turn-key offering with Google Kubernetes Engine 
(GKE). In addition, Cloud Storage FUSE is integrated with [Vertex AI] including 
Google pre-built Machine Learning images such as a [Deep Learning Virtual Machine] 
image, or [Deep Learning Container image].

[Vertex AI]: https://cloud.google.com/vertex-ai/docs/training/code-requirements#fuse
[Deep Learning Virtual Machine]: https://cloud.google.com/deep-learning-vm
[Deep Learning Container image]: https://cloud.google.com/deep-learning-containers

## Technical Overview

Cloud Storage FUSE works by translating object storage names into a file and 
directory system, interpreting the “/” character in object names as a directory
separator so that objects with the same common prefix are treated as files in
the same directory. Applications can interact with the mounted bucket like a 
simple file system, providing virtually limitless file storage running in the cloud.

While Cloud Storage FUSE has a file system interface, it is not like a true 
POSIX, NFS or CIFS file system on the backend. Cloud Storage FUSE retains the 
same fundamental characteristics of Cloud Storage, preserving the scalability of
Cloud Storage in terms of size and aggregate performance while maintaining the 
same latency and single object performance. As with the other access methods, 
Cloud Storage does not support concurrency and locking. For example, if multiple
Cloud Storage FUSE clients are writing to the same file, the last flush wins.

## Key Differences from a POSIX file system:

Cloud Storage FUSE does not provide full POSIX support.  While Cloud Storage 
FUSE has a file system interface, it is not like a true POSIX, NFS or CIFS file 
system on the backend. It is ideal for use cases where Cloud Storage has the right 
performance and scalability characteristics for an application, and only basic file
system semantics are missing. When deciding if Cloud Storage FUSE is an appropriate 
solution, there are some additional differences compared to local file systems that 
should be taken into account:

**Performance**: Cloud Storage FUSE is a client interface to Cloud Storage and 
therefore has the same performance characteristics of Cloud Storage. See 
performance best practices [here](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/performance.md)

**Metadata**: Cloud Storage FUSE does not transfer metadata along with the file when 
uploading to Cloud Storage. This means that if you wish to use Cloud Storage FUSE 
as an uploading tool, you will not be able to set [metadata] such as content type 
and ACLs as you would with other uploading methods. If metadata properties are 
critical, considering using [gsutil], the [JSON API] or the [Google Cloud console]. 
The exception to this is that Cloud Storage FUSE does store mtime and symlink targets.

[metadata]: https://cloud.google.com/storage/docs/metadata#editable
[gsutil]: https://cloud.google.com/storage/docs/gsutil
[JSON API]: https://cloud.google.com/storage/docs/json_api
[Google Cloud console]: https://console.developers.google.com/

**Concurrency**: There is no concurrency control for multiple writers to a file. 
When multiple writers try to replace a file the last write wins and all previous
writes are lost - there is no merging, version control, or user notification of 
the subsequent overwrite.

**Linking**: Cloud Storage FUSE does not support hard links.

**Semantics**: Some semantics are not exactly what they would be in a traditional 
file system, and are documented [here](https://github.com/googlecloudplatform/gcsfuse/blob/master/docs/semantics.md#surprising-behaviors).

**Access**: Authorization for files is governed by Cloud Storage permissions. 
POSIX-style access control can only be specified at the time of mounting, 
works for users accessing the mount on that machine and does not affect ACLs 
of objects or buckets in Cloud Storage.

**Local storage**: Objects that are new or modified will be stored in their 
entirety in a local temporary file until they are closed or synced. 
When working with large files, be sure you have enough local storage capacity 
for temporary copies of the files, particularly if you are working with 
[Google Compute Engine instances]. 

[Google Compute Engine instances]:https://cloud.google.com/compute/docs/instances

**Directories**: By default, only directories that are explicitly defined 
(that is, they are their own object in Cloud Storage) will appear in the 
file system. A flag is available to change this behavior. Atomic directory 
rename is not supported.

**Error Handling**: Transient errors can occur in distributed systems like Cloud
Storage, such as network timeouts. Cloud Storage FUSE implements the Cloud 
Storage [retry best practices] with exponential backoff.

[retry best practices]:https://cloud.google.com/storage/docs/retry-strategy

For additional information, see the [semantics documentation].

[semantics documentation]: https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#files-and-dirs

For a fully supported enterprise grade filesystem, see Google Cloud [Filestore].

[Filestore]: https://cloud.google.com/filestore

## Charges incurred with Cloud Storage FUSE

Cloud Storage FUSE is available free of charge, but the storage, metadata, 
and network I/O it generates to and from Cloud Storage are charged like any 
other Cloud Storage interface. For information on how to estimate the costs 
from Cloud Storage FUSE operations, see [Cloud Storage FUSE pricing].

[Cloud Storage FUSE pricing]: https://cloud.google.com/storage/docs/gcs-fuse#charges

# CSI Driver

Using the [Cloud Storage FUSE CSI driver](https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver), users get the declarative nature of Kubernetes
with all infrastructure fully managed by GKE in combination with Cloud Storage. This CSI
driver relies on Cloud Storage FUSE to mount Cloud storage buckets as file systems on the
GKE nodes, with the Cloud Storage FUSE deployment and management fully handled by GKE, 
providing a turn-key experience.

# Support

## Supported applications and platforms

Cloud Storage FUSE officially supports  the latest two versions of the following 
Machine Learning frameworks:

- TensorFlow V2.x
- TensorFlow V1.x
- PyTorch V1.x

Support outside of the above Machine Learning use cases is non-guaranteed, and on 
a best effort basis.

## Supported operating systems

Cloud Storage FUSE supports usage with the following operating systems:
- Ubuntu 18.04 and above
- Debian 10 and above
- CentOS 7 and above
- RHEL 7 and above

## Getting support

Request support via your regular Google Cloud support channels, or by opening an
issue via GitHub [here](https://github.com/GoogleCloudPlatform/gcsfuse/issues)

See [Troubleshooting] for common issue handling

[Troubleshooting]:https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/troubleshooting.md

## Limitations and restrictions

Cloud Storage FUSE is not supported for usage with the following workloads or scenarios:

- Workloads that expect POSIX compliance.
     - Cloud Storage FUSE is not a traditional POSIX filesystem. For a fully supported
       enterprise grade filesystem, see [Cloud Filestore]. See [Key Differences from a 
       POSIX filesystem](https://cloud.google.com/storage/docs/gcs-fuse#expandable-1) and [Semantics doc](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md)
- Workloads that require file locking, or concurrent writes.
     - You should not write to the same file at the same time from different clients,
       as there is no concurrency control for multiple writers to a file. When multiple 
       writers try to replace a file the last write wins and all previous writes are 
       lost - there is no merging, version control, or user notification of the subsequent overwrite. 
       For example, if multiple Cloud Storage FUSE clients are writing to the same file, the last flush wins.
  
- Object versioning: Cloud Storage FUSE does not support buckets with object versioning enabled.
No guarantees are made about the behavior when used with a bucket with this feature enabled, which may
return non-current object versions.
- Transcoding: Reading or modifying a file backed by an object with a contentEncoding property set may
yield surprising results, and no guarantees are made about the behavior. It is recommended that you do not use 
Cloud Storage FUSE to interact with such objects.
- Retention policy: Cloud Storage FUSE does not support writing to a bucket with a Retention policy enabled - writes will fail.
Reading objects from a bucket with a Retention policy enabled is supported, but the bucket should be mounted as Read-Only 
by passing the ```-o RO``` flag during mount.
- Workloads that do file patching (overwrites in place).
     - Cloud Storage write semantics are for whole objects only. There is no mechanism for patching. Therefore, write patterns
       like these may result in poor performance and ballooning operations costs.
- Workloads that do directory renaming:
     - [Renaming] directories is by default not supported. A directory rename cannot be performed atomically in Cloud Storage 
       and would therefore be expensive(object copy, then delete) in terms of Cloud Storage operations, and for large directories
       would have high probability of failure, leaving the two directories in an inconsistent state. For example, Hadoop writes files to a 
       temporary directory and at the end re-names the directory to indicate completion.
- Databases
     - Databases typically use very small data sizes. Cloud Storage FUSE higher latency than a local file system, and as such, 
       latency and throughput will be reduced when reading or writing one small file at a time.
     - Traditional Databases on File Servers require overwriting in the middle of the file, which is not supported.
- Version control systems
     - Typically require file locking and file patching
- A filer replacement
     - Amongst many other things that users require from filers, Cloud Storage FUSE does not support file patching, file locking, 
       or support native directories. 

[Cloud Filestore]: https://cloud.google.com/filestore
[Renaming]:https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#missing-features

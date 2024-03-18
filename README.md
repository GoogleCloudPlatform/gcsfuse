# Current status

Starting with V1.0, Cloud Storage FUSE is Generally Available and supported by Google, provided that it is used within its documented supported applications, platforms, and limits. Support requests, feature requests, and general questions should be submitted as a support request via Google Cloud support channels or via GitHub [here](https://github.com/googlecloudplatform/gcsfuse/v2/issues).

Cloud Storage FUSE is open source software, released under the 
[Apache license](https://github.com/googlecloudplatform/gcsfuse/v2/blob/master/LICENSE).

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

## Supported frameworks and operating systems

To find out which frameworks and operating systems are supported by Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse#supported-frameworks-os.

## Getting support

You can get support, submit general questions, and request new features by [filing issues in GitHub](https://github.com/googlecloudplatform/gcsfuse/v2/issues). You can also get support by using one of [Google Cloud's official support channels](https://cloud.google.com/support-hub).

See [Troubleshooting](https://github.com/googlecloudplatform/gcsfuse/v2/blob/master/docs/troubleshooting.md) for common issue handling.


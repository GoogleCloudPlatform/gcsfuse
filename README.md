This is a user-space file system for interacting with [Google Cloud
Storage][gcs]. It is not yet ready for public consumption, so please do not use
it for anything of importance in its current state.

[gcs]: https://cloud.google.com/storage/


# Installing

See [installing.md][] for full installation instructions. The summary is that if
you already have Go, fuse, and Git installed, you need only run:

[installing.md]: https://github.com/googlecloudplatform/gcsfuse/blob/master/docs/installing.md

```
go get github.com/googlecloudplatform/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse` to `$GOPATH/bin`.


# Mounting

## Prerequisites

Before invoking gcsfuse, you must have a GCS bucket that you want to mount. If
your bucket doesn't yet exist, create one using the
[Google Developers Console][console].

In order to be permitted by GCS to access your bucket, gcsfuse requires
appropriate credentials in the form of a [service account][]. Follow the
instructions [here][create-key] to create a service account, generate a private
key, and download a JSON file containing the private key. Place the JSON file on
the machine that will be mounting the bucket.

[console]: console.developers.google.com
[service account]: https://cloud.google.com/storage/docs/authentication#service_accounts
[create-key]: https://cloud.google.com/storage/docs/authentication#generating-a-private-key

## Invoking gcsfuse

To mount a bucket using gcsfuse, invoke it like this:

```
gcsfuse --key_file /path/to/key.json --bucket my-bucket /path/to/mount/point
```

The directory onto which you are mounting the file system
(`/path/to/mount/point` in the above example) must already exist.


# Performance

Performance when copying data into GCS is comparable to gsutil (see
[issue #22][issue-22] for testing notes). There is some overhead due to staging
of data in temporary files before writing out to GCS, which is unavoidable given
the file system semantics.

[issue-22]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/22

If you notice unreasonable performance, please [file an issue][issues].

[issues]: https://github.com/googlecloudplatform/gcsfuse/issues

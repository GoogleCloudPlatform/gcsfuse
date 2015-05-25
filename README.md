gcsfuse is a user-space file system for interacting with [Google Cloud
Storage][gcs].

[gcs]: https://cloud.google.com/storage/


# Current status

Please treat gcsfuse as alpha-quality software. Use it for whatever you like,
but be aware that bugs may lurk, and that we reserve the right to make
backwards-incompatible changes.

The careful user should be sure to read [semantics.md][] for information on how
gcsfuse maps file system operations to GCS operations, and especially on
surprising behaviors. The list of [open issues][issues] may also be of interest.

[semantics.md]: docs/semantics.md
[issues]: https://github.com/GoogleCloudPlatform/gcsfuse/issues


# Installing

See [installing.md][] for full installation instructions. The summary is that if
you already have Go, fuse, and Git installed, you need only run:

[installing.md]: https://github.com/googlecloudplatform/gcsfuse/blob/master/docs/installing.md

```
go get github.com/googlecloudplatform/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse` to `$GOPATH/bin`. If you ever need to update to the latest version of
gcsfuse, you can do so with:

```
go get -u github.com/googlecloudplatform/gcsfuse
```


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

[console]: https://console.developers.google.com
[service account]: https://cloud.google.com/storage/docs/authentication#service_accounts
[create-key]: https://cloud.google.com/storage/docs/authentication#generating-a-private-key

## Invoking gcsfuse

To mount a bucket using gcsfuse, invoke it like this:

```
gcsfuse --bucket my-bucket --mount_point /path/to/mount/point
```

The directory onto which you are mounting the file system
(`/path/to/mount/point` in the above example) must already exist.

The gcsfuse tool will run until the file system is unmounted. By default little
is printed, but you can use the `--fuse.debug` flag to turn on debugging output
to stderr. If the tool should happen to crash, crash logs will also be written
to stderr.

If you are mounting a bucket that was populated with objects by some other means
besides gcsfuse, you may be interested in the `--implicit_dirs` flag. See the
notes in [semantics.md][semantics-implicit-dirs] for more information.

[semantics-implicit-dirs]: docs/semantics.md#implicit-directories

See [mounting.md][] for more detail, including notes on running as a daemon and
fstab compatiblity.

[mounting.md]: /docs/mounting.md


# Performance

## GCS round trips

By default, gcsfuse uses two forms of caching to save round trips to GCS, at the
cost of consistency guarantees. These caching behaviors can be controlled with
the flags `--stat_cache_ttl` and `--type_cache_ttl`. See
[semantics.md](docs/semantics.md#caching) for more information.

## Downloading file contents

Behind the scenes, when a newly-opened file is first read or modified, gcsfuse
downloads the entire backing object's contents from GCS. (Unless it is a
newly-created file, of course.) The contents are stored in a local temporary
file whose location is controlled by the flag `--temp_dir` and whose size is
controlled with `--temp_dir_limit`, which is used to serve reads and
modifications. Later, when the file is closed or fsync'd, gcsfuse writes the
contents of the local file back to GCS as a new object generation.

The consequence of this is that gcsfuse is relatively efficient when reading or
writing entire large files, but will not be particularly fast for small numbers
of random reads or writes within larger files. Performance when copying large
files into GCS is comparable to gsutil (see [issue #22][issue-22] for testing
notes). There is some overhead due to the staging of data in a local temporary
file, as discussed above.

[issue-22]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/22

## Other performance issues

If you notice otherwise unreasonable performance, please [file an
issue][issues].

[issues]: https://github.com/googlecloudplatform/gcsfuse/issues

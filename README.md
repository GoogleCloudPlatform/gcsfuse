gcsfuse is a user-space file system for interacting with [Google Cloud
Storage][gcs].

[gcs]: https://cloud.google.com/storage/


# Current status

Please treat gcsfuse as beta-quality software. Use it for whatever you like, but
be aware that bugs may lurk, and that we reserve the right to make small
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

[console]: https://console.developers.google.com

## Invoking gcsfuse

To mount a bucket using gcsfuse, invoke it like this:

```
gcsfuse my-bucket /path/to/mount/point
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

## Rate limiting

If you would like to rate limit traffic to/from GCS in order to set limits on
your GCS spending on behalf of gcsfuse, you can do so:

*   The flag `--op_rate_limit_hz` controls the rate at which gcsfuse will send
    requests to GCS.
*   The flag `--egress_bandwidth_limit_bytes_per_second` controls the egress
    bandwidth from gcsfuse to GCS.

All rate limiting is approximate, and is performed over a 30-second window. Rate
limiting is disabled by default.

## Other performance issues

If you notice otherwise unreasonable performance, please [file an
issue][issues].

[issues]: https://github.com/googlecloudplatform/gcsfuse/issues


<a name="support">
# Support

gcsfuse is open source software, released under the [Apache license](LICENSE).
It is distributed as-is, without warranties or conditions of any kind. Best
effort community support is available on [StackExchange][se] with the
`google-cloud-platform` and `google-cloud-storage` tags. Please be sure to look
at [previous questions and answers][qna] before asking a new one. For bugs and
feature requests, please [file an issue][issues].

[se]: http://serverfault.com/questions/ask?tags=google-cloud-platform+google-cloud-storage
[qna]: http://serverfault.com/questions/tagged/google-cloud-platform+google-cloud-storage

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

See [installing.md][] for full installation instructions for Linux and Mac OS X.

[installing.md]: docs/installing.md


# Mounting

## Prerequisites

* Before invoking gcsfuse, you must have a GCS bucket that you want to mount. If
your bucket doesn't yet exist, create one using the
[Google Developers Console][console].

[console]: https://console.cloud.google.com

* Make sure [the Google Cloud Storage JSON API is enabled][enableAPI].

[enableAPI]: https://cloud.google.com/storage/docs/json_api/#activating

* GCS credentials are automatically loaded using [Google application default
credentials][app-default-credentials], or a JSON key file can be specified
explicitly using `--key-file`. If you haven't already done so, the easiest way
to set up your credentials for testing is to run the [gcloud tool][]:

```
    gcloud auth login
```
  See [mounting.md][] for more information on credentials.

[gcloud tool]: https://cloud.google.com/sdk/gcloud/
[app-default-credentials]: https://developers.google.com/identity/protocols/application-default-credentials#howtheywork
[mounting.md]: /docs/mounting.md

## Invoking gcsfuse

To mount a bucket using gcsfuse over an existing directory `/path/to/mount`,
invoke it like this:

```
gcsfuse my-bucket /path/to/mount
```

**Important**: You should run gcsfuse as the user who will be using the file
system, not as root. Do not use `sudo`.

The gcsfuse tool will exit successfully after mounting the file system. Unmount
in the usual way for a fuse file system on your operating system:

    umount /path/to/mount         # OS X
    fusermount -u /path/to/mount  # Linux

If you are mounting a bucket that was populated with objects by some other means
besides gcsfuse, you may be interested in the `--implicit-dirs` flag. See the
notes in [semantics.md][semantics-implicit-dirs] for more information.

[semantics-implicit-dirs]: docs/semantics.md#implicit-directories

See [mounting.md][] for more detail, including notes on running in the
foreground and fstab compatiblity.

[mounting.md]: /docs/mounting.md


# Performance

## Latency and rsync

Writing files to and reading files from GCS has a much higher latency than using
a local file system. If you are reading or writing one small file at a time,
this may cause you to achieve a low throughput to or from GCS. If you want high
throughput, you will need to either use larger files to smooth across latency
hiccups or read/write multiple files at a time.

Note in particular that this heavily affects `rsync`, which reads and writes
only one file at a time. You might try using [`gsutil -m rsync`][gsutil rsync]
to transfer multiple files to or from your bucket in parallel instead of plain
`rsync` with gcsfuse.

[gsutil rsync]: https://cloud.google.com/storage/docs/gsutil/commands/rsync

## Rate limiting

If you would like to rate limit traffic to/from GCS in order to set limits on
your GCS spending on behalf of gcsfuse, you can do so:

*   The flag `--limit-ops-per-sec` controls the rate at which gcsfuse will send
    requests to GCS.
*   The flag `--limit-bytes-per-sec` controls the egress
    bandwidth from gcsfuse to GCS.

All rate limiting is approximate, and is performed over an 8-hour window. By
default, requests are limited to 5 per second. There is no limit applied to
bandwidth by default.

## GCS round trips

By default, gcsfuse uses two forms of caching to save round trips to GCS, at the
cost of consistency guarantees. These caching behaviors can be controlled with
the flags `--stat-cache-capacity`, `--stat-cache-ttl` and `--type-cache-ttl`. See
[semantics.md](docs/semantics.md#caching) for more information.

## Timeouts

If you are using [FUSE for macOS](https://osxfuse.github.io/), be aware that by
default it will give gcsfuse only 60 seconds to respond to each file system
operation. This means that if you write and then flush a large file and your
upstream bandwidth is insufficient to write it all to GCS within 60 seconds,
your gcsfuse file system may become unresponsive. This behavior can be tuned
using the [`daemon_timeout`][timeout] mount option. See [issue #196][] for
details.

[timeout]: https://github.com/osxfuse/osxfuse/wiki/Mount-options#daemon_timeout
[issue #196]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/196


## Downloading object contents

Behind the scenes, when a newly-opened file is first modified, gcsfuse downloads
the entire backing object's contents from GCS. The contents are stored in a
local temporary file whose location is controlled by the flag `--temp-dir`.
Later, when the file is closed or fsync'd, gcsfuse writes the contents of the
local file back to GCS as a new object generation.

Files that have not been modified are read portion by portion on demand. gcsfuse
uses a heuristic to detect when a file is being read sequentially, and will
issue fewer, larger read requests to GCS in this case.

The consequence of this is that gcsfuse is relatively efficient when reading or
writing entire large files, but will not be particularly fast for small numbers
of random writes within larger files, and to a lesser extent the same is true of
small random reads. Performance when copying large files into GCS is comparable
to gsutil (see [issue #22][issue-22] for testing notes). There is some overhead
due to the staging of data in a local temporary file, as discussed above.

[issue-22]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/22

Note that new and modified files are also fully staged in the local temporary
directory until they are written out to GCS due to being closed or fsync'd.
Therefore the user must ensure that there is enough free space available to
handle staged content when writing large files.

## Other performance issues

If you notice otherwise unreasonable performance, please [file an
issue][issues].

[issues]: https://github.com/googlecloudplatform/gcsfuse/issues

# Support

gcsfuse is open source software, released under the [Apache license](LICENSE).
It is distributed as-is, without warranties or conditions of any kind.

For support, visit [Server Fault][sf]. Tag your questions with `gcsfuse` and
`google-cloud-platform`, and make sure to look at
[previous questions and answers][previous] before asking a new one. For bugs and
feature requests, please [file an issue][issues].

[sf]: http://serverfault.com/
[previous]: http://serverfault.com/questions/tagged/gcsfuse


# Versioning

gcsfuse version numbers are assigned according to [Semantic
Versioning][semver]. Note that the current major version is `0`, which means
that we reserve the right to make backwards-incompatible changes.

[semver]: http://semver.org/

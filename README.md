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

Before invoking gcsfuse, you must have a GCS bucket that you want to mount. If
your bucket doesn't yet exist, create one using the
[Google Developers Console][console].

[console]: https://console.developers.google.com

GCS credentials are automatically loaded using [Google application default
credentials][app-default-credentials], or a JSON key file can be specified
explicitly using `--key-file`. If you haven't already done so, the easiest way
to set up your credentials for testing is to run the [gcloud tool][]:

    gcloud auth login

See [mounting.md][] for more information on credentials.

[gcloud tool]: https://cloud.google.com/sdk/gcloud/
[app-default-credentials]: https://developers.google.com/identity/protocols/application-default-credentials#howtheywork
[mounting.md]: /docs/mounting.md

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
besides gcsfuse, you may be interested in the `--implicit-dirs` flag. See the
notes in [semantics.md][semantics-implicit-dirs] for more information.

[semantics-implicit-dirs]: docs/semantics.md#implicit-directories

See [mounting.md][] for more detail, including notes on running as a daemon and
fstab compatiblity.

[mounting.md]: /docs/mounting.md


# Performance

## GCS round trips

By default, gcsfuse uses two forms of caching to save round trips to GCS, at the
cost of consistency guarantees. These caching behaviors can be controlled with
the flags `--stat-cache-ttl` and `--type-cache-ttl`. See
[semantics.md](docs/semantics.md#caching) for more information.

## Downloading file contents

Behind the scenes, when a newly-opened file is first modified, gcsfuse downloads
the entire backing object's contents from GCS. The contents are stored in a
local temporary file whose location is controlled by the flag `--temp-dir`.
Later, when the file is closed or fsync'd, gcsfuse writes the contents of the
local file back to GCS as a new object generation.

Files that are not modified are read chunk by chunk on demand. Such non-dirty
content is cached in the temporary directory, with a size limit defined by
`--temp-dir-bytes`. The chunk size is controlled by `--gcs-chunk-size`.

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

## Rate limiting

If you would like to rate limit traffic to/from GCS in order to set limits on
your GCS spending on behalf of gcsfuse, you can do so:

*   The flag `--limit-ops-per-sec` controls the rate at which gcsfuse will send
    requests to GCS.
*   The flag `--limit-bytes-per-sec` controls the egress
    bandwidth from gcsfuse to GCS.

All rate limiting is approximate, and is performed over a 30-second window. By
default, requests are limited to 5 per second. There is no limit applied to
bandwidth by default.

## Other performance issues

If you notice otherwise unreasonable performance, please [file an
issue][issues].

[issues]: https://github.com/googlecloudplatform/gcsfuse/issues


<a name="support">
# Support

gcsfuse is open source software, released under the [Apache license](LICENSE).
It is distributed as-is, without warranties or conditions of any kind.

Best effort community support is available on [Server Fault][sf]. Please be sure
to tag your questions with `gcsfuse` and `google-cloud-storage`, and to look at
[previous questions and answers][previous] before asking a new one. For bugs and
feature requests, please [file an issue][issues].

[sf]: http://serverfault.com/
[previous]: http://serverfault.com/questions/tagged/gcsfuse


# Versioning

gcsfuse version numbers are assigned according to [Semantic
Versioning][semver]. Note that the current major version is `0`, which means
that we reserve the right to make backwards-incompatible changes.

[semver]: http://semver.org/

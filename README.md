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

See [installing.md][] for full installation instructions for Linux and macOS.

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

    umount /path/to/mount         # macOS
    fusermount -u /path/to/mount  # Linux

If you are mounting a bucket that was populated with objects by some other means
besides gcsfuse, you may be interested in the `--implicit-dirs` flag. See the
notes in [semantics.md][semantics-implicit-dirs] for more information.

[semantics-implicit-dirs]: docs/semantics.md#implicit-directories

See [mounting.md][] for more detail, including notes on running in the
foreground and fstab compatibility.

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
default, there are no limits applied.

## Upload procedure control

An upload procedure is implemented as a retry loop with exponential backoff 
for failed requests to the GCS backend. Once the backoff duration exceeds this 
limit, the retry stops. Flag `--max-retry-sleep` controls such behavior.
The default is 1 minute. A value of 0 disables retries.

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

# Switching to Go Storage Client

This branch of the GCSFuse repository provides ability to use [Go Storage Client](https://pkg.go.dev/cloud.google.com/go/storage) as a client to interact with the GCS backend. Option to choose between Go Storage Client and currently used JSON API client is available via a flag mentioned below. All the methods present in the third party library, [jacobsa/gcloud](vendor/github.com/jacobsa/gcloud/gcs/bucket.go#L44) are changed to use the Go Storage Client.

## Extra flags added to access Go Client

* `--enable-storage-client-library` flag enables us to switch the client to the Go Storage Client for communicating with the GCS backend.
* `--max-idle-conns-per-host` flag allows us to set the max limit of idle connections when using the Go Storage Client in HTTP 1.1 mode.

## How to access different versions of clients

* **Default JSON API Client** 
```
go run . --implicit-dirs <bucket_name> <mount_dir_name>
```

* **JSON API Client (perf mode)** 
```
go run . --implicit-dirs --disable-http2 --max-conns-per-host 100 <bucket_name> <mount_dir_name>
```

* **Default Go Storage Client** \
Firstly set the `DisableKeepAlives` and `ForceAttemptHTTP2` parameters to False in the [bucket.go](vendor/github.com/jacobsa/gcloud/gcs/bucket.go#L574) file. Then run the following command.
```
go run . --implicit-dirs --enable-storage-client-library <bucket_name> <mount_dir_name>
```

* **Go Storage Client HTTP 2.0 (perf mode)**
```
go run . --implicit-dirs --enable-storage-client-library <bucket_name> <mount_dir_name>
```

* **Go Storage Client HTTP 1.1 (perf mode)**
```
go run . --implicit-dirs --enable-storage-client-library --disable-http2 --max-conns-per-host 100 <bucket_name> <mount_dir_name>
```

* **Go Storage Client HTTP 1.1 (ultra perf mode)**
```
go run . --implicit-dirs --enable-storage-client-library --disable-http2 --max-conns-per-host 100 --max-idle-conns-per-host 100 <bucket_name> <mount_dir_name>
```

## Extra points for using Go Storage Client

* [HTTP client timeout](vendor/github.com/jacobsa/gcloud/gcs/bucket.go#L585) needs to be always set when using Go Storage Client HTTP 1.1 for random reads. If no client timeout is set, then the HTTP client will be stuck indefinitely after making some calls. 

* Client timeout affect the performance in case of read flows. Lesser the client timeout more is the performance in random reads scenario. But for sequential reads it is just the opposite i.e. more the client timeout better the performance. So a optimal value of client timeout needs to mantained in order to get good results in both type of access patterns. Optimal value as per previous experiments was in the ball park of 800 ms.

* While the write flows are not affected by the client timeout but they highly depend on the [ChunkSize](vendor/github.com/jacobsa/gcloud/gcs/create_object.go#L268) parameter of the NewWriter. Currently the ChunkSize parameter is set to 0 to perform one-shot uploads because the current JSON API client also performs one-shot uploads. But it can be changed in the future as per needs. 

* The performance of the listing operation is dependent on the [MaxResults](vendor/cloud.google.com/go/storage/bucket.go#L1994) parameter. For now it is hardcoded to 5000 but we need to find a way to make it configurable.

## Performance of Go Storage Client

* **Reads**: In sequential reads, the Go Storage Client in HTTP 1.1 mode performs the best. Performs even better than the current JSON API client.
* **Writes**: In writes, be it random access or sequential access, all the clients perform equally well and there is not much of a difference.
* **List**: Go Storage Client in HTTP 1.1 mode and JSON API client in perf mode performs the best.

To conclude, according to our assessment Go Storage Client in HTTP 1.1 mode performs the best in an overall way.

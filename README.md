This is a user-space file system for interacting with [Google Cloud
Storage][gcs]. It is not yet ready for public consumption, so please do not use
it for anything of importance in its current state.

[gcs]: https://cloud.google.com/storage/


# Installing

Prerequisites:

*   gcsfuse is distributed as source code in the [Go][go] language. If you do
    not yet have Go installed, see [here][go-install] for instructions.

*   OS X only: Before using gcsfuse, you must have [FUSE for OS X][osxfuse].
    Installing FUSE is not necessary on Linux, since modern versions have kernel
    support built in.

*   The `go get` command bbelow will need to fetch source code from GitHub,
    which requires [Git][git]. If the `git` binary is not installed on your
    system, download it [here][git-download] or install it by some other means
    (for example on Google Compute Engine Debian instances you can run
    `sudo apt-get update && sudo apt-get install git-core`).

[go]: http://golang.org/
[go-install]: http://golang.org/doc/install
[osxfuse]: https://osxfuse.github.io/
[git]: http://git-scm.com/
[git-download]: http://git-scm.com/downloads

To install gcsfuse, run:

```
go get github.com/jacobsa/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse`.


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

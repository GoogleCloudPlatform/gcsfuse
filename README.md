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

[go]: http://golang.org/
[go-install]: http://golang.org/doc/install
[osxfuse]: https://osxfuse.github.io/

To install gcsfuse, run:

```
go get github.com/jacobsa/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse`.

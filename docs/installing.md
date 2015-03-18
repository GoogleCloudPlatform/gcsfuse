Prerequisites:

*   gcsfuse is distributed as source code in the [Go][go] language. If you do
    not yet have Go installed, see [here][go-install] for instructions. Be sure
    to follow the linked [setup instructions][go-setup], in particular setting
    the `GOPATH` environment variable and ensuring `$GOPATH/bin` is in your
    `PATH` environment variable.

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
[go-setup]: http://golang.org/doc/code.html
[osxfuse]: https://osxfuse.github.io/
[git]: http://git-scm.com/
[git-download]: http://git-scm.com/downloads

To install gcsfuse, run:

```
go get github.com/jacobsa/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse` to `$GOPATH/bin`.

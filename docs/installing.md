# Prerequisites

## Operating system

gcsfuse has been tested successfully with the following operating systems:

*   Linux (minimum kernel version 3.10)
*   OS X (minimum version 10.10.2)

It may or may not work correctly with other operating systems and older versions.


## Go

gcsfuse is distributed as source code in the [Go][go] language. If you already
have a working Go environment at the latest version, you can skip this section.

If you do not yet have Go installed, see [here][go-install] for instructions.
Be sure to follow the linked [setup instructions][go-setup], in particular
setting the `GOPATH` environment variable and ensuring both the `go` tool and
`$GOPATH/bin` are in your `PATH` environment variable. That probably involves
putting something that looks like this in your shell environment:

```
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$GOPATH/bin
```

[go]: http://golang.org/
[go-install]: http://golang.org/doc/install
[go-setup]: http://golang.org/doc/code.html


## Fuse

### Linux

Linux users must ensure that the `fusermount` binary is installed and is
executableby the user running gcsfuse, and that kernel support for fuse is
enabled and `/dev/fuse` has the appropriate permissions.

On Debian, fuse can be installed with:

```
sudo apt-get update && sudo apt-get install fuse
```

By default only those users in the `fuse` group can mount a fuse file system.
Add your user to this group with the following command, then log out and log
back in:

```
sudo adduser $USER fuse
```

On some versions of Debian, including the default Google Compute Engine image
as of 2015-03-18, `/dev/fuse` has incorrect permissions (cf.
[this][stackexchange] StackExchange answer). Fix this with the following
commands:

```
sudo chmod g+rw /dev/fuse
sudo chgrp fuse /dev/fuse
```

[stackexchange]: http://superuser.com/a/800016/429161

### OS X

OS X users must install [osxfuse][] before running gcsfuse.

[osxfuse]: https://osxfuse.github.io/


## Git

The `go get` command bbelow will need to fetch source code from GitHub, which
requires [Git][git]. If the `git` binary is not installed on your system,
download it [here][git-download] or install it by some other means (for example
on Google Compute Engine Debian instances you can run:

```
sudo apt-get update && sudo apt-get install git-core
```

[git]: http://git-scm.com/
[git-download]: http://git-scm.com/downloads



# Installation

To install gcsfuse, run:

```
go get github.com/googlecloudplatform/gcsfuse
```

This will fetch the gcsfuse sources, build them, and install a binary named
`gcsfuse` to `$GOPATH/bin`.

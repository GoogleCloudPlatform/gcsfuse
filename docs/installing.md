
gcsfuse has been tested successfully with the following operating systems:

*   Linux (minimum kernel version 3.10)
*   OS X (minimum version 10.10.2)

It may or may not work correctly with other operating systems and older versions.


# Linux

If you are running Linux on a 64-bit x86 machine and are happy to install
pre-built binaries (i.e. you don't want to build from source), you need only
ensure fuse is installed and then download and extract the latest release. The
instructions slightly vary by distribution.

## Debian and Ubuntu

    sudo apt-get install wget fuse
    wget https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.3.1/gcsfuse_v0.3.1_linux_amd64.tar.gz
    sudo tar -C /usr/local/bin -zxf gcsfuse_v0.3.0_linux_amd64.tar.gz

On old versions of Debian, including the one in the Google Compute Engine image
`debian-7` as of 2015-07-20, `/dev/fuse` has incorrect permissions (cf.
[this][stackexchange] StackExchange answer). Fix this with the following
commands:

```
sudo chmod g+rw /dev/fuse
sudo chgrp fuse /dev/fuse
```

Note that the operating system appears to periodically lose these changes, so
you may need to run the workaround above repeatedly.

[stackexchange]: http://superuser.com/a/800016/429161

## CentOS and Red Hat

    sudo yum install wget fuse
    wget https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.3.1/gcsfuse_v0.3.1_linux_amd64.tar.gz
    sudo tar -C /usr/local/bin -zxf gcsfuse_v0.3.0_linux_amd64.tar.gz

## SUSE

    sudo zypper install wget fuse
    wget https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.3.1/gcsfuse_v0.3.1_linux_amd64.tar.gz
    sudo tar -C /usr/local/bin -zxf gcsfuse_v0.3.0_linux_amd64.tar.gz


# OS X

If you already have the [Homebrew package manager][homebrew] installed, you can
do the following to install gcsfuse:

[homebrew]: http://brew.sh/

*   Download and install [osxfuse][]. (In modern versions of OS X you cannot do
    this via Homebrew because of Apple's requirements for kernel extension
    signatures.)

*   Run `brew tap homebrew/fuse && brew install gcsfuse`.

[osxfuse]: https://osxfuse.github.io/

Otherwise, or if you want to install a pre-release version of gcsfuse, read the
remainder of this document.


# Building from source

Prerequisites:

*   A working [Go][go] installation. See the Go [setup instructions][go-setup].
*   Fuse. See the instructions for the binary release above.
*   Git. This is probably available as `git` in your package manager.

[go]: http://golang.org/
[go-setup]: http://golang.org/doc/code.html

To install or update gcsfuse, run:

    go get -u github.com/googlecloudplatform/gcsfuse

This will fetch the gcsfuse sources to
`$GOPATH/src/github.com/googlecloudplatform/gcsfuse`, build them, and install a
binary named `gcsfuse` to `$GOPATH/bin`.

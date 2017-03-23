
gcsfuse has been tested successfully with the following operating systems:

*   Linux (minimum kernel version 3.10)
*   OS X (minimum version 10.10.2)

It may or may not work correctly with other operating systems and older versions.

If you are running on [Google Compute Engine][], it is recommended that you use
one of the following images with which it has been tested (preferring the
latest version when possible):

*   `ubuntu-1604-lts`, `ubuntu-1404-lts`
*   `debian-8`, `debian-7`
*   `centos-7`, `centos-6`
*   `rhel-7`, `rhel-6`
*   `sles-12`

[Google Compute Engine]: https://cloud.google.com/compute/


# Linux

If you are running Linux on a 64-bit x86 machine and are happy to install
pre-built binaries (i.e. you don't want to build from source), you need only
ensure fuse is installed, then download and install the latest release package.
The instructions vary by distribution.


## Ubuntu and Debian (latest releases)

The following instructions set up `apt-get` to see updates to gcsfuse, and are
supported for the **yakkety**, **xenial**, and **trusty**
[releases][ubuntu-releases] of Ubuntu, and the **jessie**
[release][debian-releases] of Debian. (Run `lsb_release -c` to find your
release codename.) Users of older releases should follow the instructions for
[other distributions](#other-distributions) below.

1.  Add the gcsfuse distribution URL as a package source and import its public
    key:

        export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
        echo "deb http://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
        curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -

2.  Update the list of packages available and install gcsfuse.

        sudo apt-get update
        sudo apt-get install gcsfuse

3.  (**Ubuntu before wily only**) Add yourself to the `fuse` group, then log
    out and back in:

        sudo usermod -a -G fuse $USER
        exit

Future updates to gcsfuse can be installed in the usual
way: `sudo apt-get update && sudo apt-get upgrade`.

[ubuntu-releases]: https://wiki.ubuntu.com/Releases
[debian-releases]: https://www.debian.org/releases/


## CentOS and Red Hat (latest releases)

The following instructions set up `yum` to see updates to gcsfuse, and work
for CentOS 7 and RHEL 7. Users of older releases should follow the instructions
for [other distributions](#other-distributions) below.

1.  Configure the gcsfuse repo:

        sudo tee /etc/yum.repos.d/gcsfuse.repo > /dev/null <<EOF
        [gcsfuse]
        name=gcsfuse (packages.cloud.google.com)
        baseurl=https://packages.cloud.google.com/yum/repos/gcsfuse-el7-x86_64
        enabled=1
        gpgcheck=1
        repo_gpgcheck=1
        gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
               https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
        EOF

2.  Install gcsfuse:

        sudo yum install gcsfuse

    Be sure to answer "yes" to any questions about adding the GPG signing key.

Future updates to gcsfuse will automatically show up when updating with `yum`.


## SUSE

Ensure that dependencies are present:

    sudo zypper install curl fuse

Download and install the latest release package:

    curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.21.0/gcsfuse-0.21.0-1.x86_64.rpm
    sudo rpm --install -p gcsfuse-0.21.0-1.x86_64.rpm

<a name="other-distributions"></a>
## Older releases and other distributions

Ensure that dependencies are present:

*   Install [fuse](http://fuse.sourceforge.net/).
*   Install [curl](http://curl.haxx.se/) (or use a different program for
    downloading below).

If you are on a distribution that uses `.rpm` files for package management:

    curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.21.0/gcsfuse-0.21.0-1.x86_64.rpm
    sudo rpm --install -p gcsfuse-0.21.0-1.x86_64.rpm

Or one that uses `.deb` files:

    curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.21.0/gcsfuse_0.21.0_amd64.deb
    sudo dpkg --install gcsfuse_0.21.0_amd64.deb

On some systems it may be necessary to add the your user account to the `fuse`
group in order to have permission to run `fusermount` (don't forget to log out
and back in afterward for the group membership change to take effect):

    sudo usermod -a -G fuse $USER
    exit

Old versions of Debian contain a [bug][debian-bug] that causes `/dev/fuse` to
repeatedly lose its permission settings. If you find that you receive
permissions errors when mounting, even after running the `usermod` instruction
above and logging out and back in, you may need to fix the permissions:

    sudo chmod g+rw /dev/fuse
    sudo chgrp fuse /dev/fuse

[fstab compatibility]: mounting.md#mount8-and-fstab-compatibility
[debian-bug]: http://superuser.com/a/800016/429161


# OS X

First, handle prerequisites:

*   Install [osxfuse](https://osxfuse.github.io/).
*   Install the [homebrew](http://brew.sh/) package manager.

Afterward, gcsfuse can be installed with `brew`:

    brew install homebrew/fuse/gcsfuse
    sudo ln -s /usr/local/sbin/mount_gcsfuse /sbin  # For mount(8) support

The second command is only necessary if you want to use gcsfuse with the
`mount` command or in your `/etc/fstab` file, as opposed to calling `gcsfuse`
directly.

In the future gcsfuse can be updated in the usual way for homebrew packages:

    brew update && brew upgrade


# Building from source

Prerequisites:

*   A working [Go][go] installation at least as new as [commit
    183cc0c][183cc0c]. See [Installing Go from source][go-setup].
*   Fuse. See the instructions for the binary release above.
*   Git. This is probably available as `git` in your package manager.

Because we use the [Go 1.5 vendoring support][183cc0c], you must ensure that
the appropriate variable is set in your environment:

    export GO15VENDOREXPERIMENT=1

To install or update gcsfuse, run:

    go get -u github.com/googlecloudplatform/gcsfuse

This will fetch the gcsfuse sources to
`$GOPATH/src/github.com/googlecloudplatform/gcsfuse`, build them, and install a
binary named `gcsfuse` to `$GOPATH/bin`.

[go]: http://tip.golang.org/doc/install/source
[183cc0c]: https://github.com/golang/go/commit/183cc0c
[go-setup]: http://golang.org/doc/code.html

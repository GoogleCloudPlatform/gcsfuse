# Credentials

Credentials for use with GCS will automatically be loaded using [Google
application default credentials][app-default-credentials], unless the flag
`--key-file` is set to a path to a JSON key file downloaded from the Google
Developers Console.

The easiest way to set up credentials when running on [Google Compute
Engine][gce] is to create your VM with a service account using the
`storage-full` access scope. (See [here][gce-service-accounts] for details on
VM service accounts.) When gcsfuse is run from such a VM, it automatically has
access to buckets owned by the same project as the VM.

When testing, especially on a developer machine, credentials can also be
configured using the [gcloud tool][]:

    gcloud auth login

Alternatively, you can set the `GOOGLE_APPLICATION_CREDENTIALS` environment
variable to the path to a JSON key file downloaded from the Google Developers
Console:

    GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]

[gce]: https://cloud.google.com/compute/
[gce-service-accounts]: https://cloud.google.com/compute/docs/authentication
[gcloud tool]: https://cloud.google.com/sdk/gcloud/
[app-default-credentials]: https://developers.google.com/identity/protocols/application-default-credentials#howtheywork


# Basic usage

## Mounting

As the user that will mount and use the file system, create the directory into
which you want to mount the gcsfuse bucket:

    mkdir /path/to/mount/point

In order to mount the bucket named `my-bucket`, invoke the gcsfuse binary
as follows:

    gcsfuse my-bucket /path/to/mount/point

You should be able to see your bucket contents if you run `ls
/path/to/mount/point`.

## Unmounting

On Linux, unmount using fuse's `fusermount` tool:

    fusermount -u /path/to/mount/point

On OS X, unmount like any other file system:

    umount /path/to/mount/point

On both systems, you can also unmount by sending `SIGINT` to the gcsfuse
process (usually by pressing Ctrl-C in the controlling terminal).


# Running as a daemon

gcsfuse runs as a foreground process, writing log messages to stderr. This
makes it easy to test out and terminate by pressing Ctrl-C, and to redirect its
output to where you like. However, it is common to want to put gcsfuse into the
background, detaching it from the terminal that started it. Advanced users also
want to manage gcsfuse log output, perhaps sending it to syslog.

In order to do this, use your preferred daemonization wrapper. Common choices
include [daemon][], [daemonize][], [daemontools][], [systemd][], and
[upstart][].

[daemon]: http://libslack.org/daemon/
[daemonize]: http://software.clapper.org/daemonize/
[daemontools]: http://cr.yp.to/daemontools.html
[systemd]: http://www.freedesktop.org/wiki/Software/systemd/
[upstart]: http://upstart.ubuntu.com/

For example, `daemon` can be installed using `sudo apt-get install daemon` on
Ubuntu or `brew install daemon` with [homebrew][] on OS X. Afterward, gcsfuse
can be run with:

    daemon -- gcsfuse my-bucket /path/to/mount/point

[homebrew]: http://brew.sh/


# mount(8) and fstab compatibility

The gcsfuse [installation process](installing.md) installed a helper understood
by the `mount` command to your system at one of these two paths, depending on
your operating system:

*   Linux: `/sbin/mount.gcsfuse`
*   OS X: `/sbin/mount_gcsfuse`

These scripts contain reasonable defaults, but you may need to edit them to
change details about your setup if you're doing anything special with regard to
e.g. credentials or daemonization. In particular, they require [daemon][] to be
installed.

These scripts allow you to mount buckets using the `mount` command:

    mount -t gcsfuse -o rw,user my-bucket /path/to/mount/point

Because the `mount` command works, you can also add entries to your
`/etc/fstab` file like the following:

    my-bucket /mount/point gcsfuse rw,noauto,user

Afterward, you can run `mount /mount/point`. The `noauto` option specifies that
the file system should not be mounted at boot time. If you want this, remove
the option and modify your mount helper to tell the daemonizing program to run
gcsfuse as your desired user.

[daemon]: http://libslack.org/daemon/

# Credentials

Credentials for use with GCS will automatically be loaded using [Google
application default credentials][app-default-credentials]. The easiest way to
set this up is using the [gcloud tool][]:

    gcloud auth login

Alternatively, you can set the `GOOGLE_APPLICATION_CREDENTIALS` environment
variable to the path to a JSON key file downloaded from the Google Developers
Console:

    GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]

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


# fstab compatibility

It is possible to set up entries for gcsfuse file systems in your `/etc/fstab`
file, such that file systems can be mounted at boot or on demand based on path
or name.

In order to do this, gcsfuse must be made compatible with the (underdocumented
and platform-specific) protocol spoken by [`mount`][mount] when calling its
external helpers. The gcsfuse repo contains a tool to help with this, which you
can install with:

    go install github.com/googlecloudplatform/gcsfuse/gcsfuse_mount_helper

[mount]: http://linux.die.net/man/8/mount

The helper accepts arguments in the form supplied by `mount`, but as discussed
above does not automatically daemonize, which is expected by `mount`. So the
final step is to install an external mount helper with a system-specific name
(e.g. `/sbin/mount_gcsfuse` on OS X, `/sbin/mount.gcsfuse` on Linux) that uses
a daemonizing wrapper program to start gcsfuse.
[gcsfuse_mount_helper/sample.sh][] contains an example that uses `daemon`.

[gcsfuse_mount_helper/sample.sh]: /gcsfuse_mount_helper/sample.sh

Once this helper is installed, you should be able to mount a bucket with a
command like the following:

    mount -t gcsfuse -o rw my-bucket /path/to/mount/point

Similarly, a line like the following can be added to `/etc/fstab` (on Linux you
may need to add the `user` option to allow non-root users):

    my-bucket /path/to/mount/point gcsfuse rw

Afterward, you can run simply `mount my-bucket`.

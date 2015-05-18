# Basic usage

First, ensure that you have downloaded a JSON file containing a private key for
your [service account][] from the [Google Developers Console][console]. Say for
the purposes of this document that it is located at `/path/to/key.json`.

[service account]: https://cloud.google.com/storage/docs/authentication#service_accounts
[console]: https://console.developers.google.com

Next, create the directory into which you want to mount the gcsfuse bucket:

    mkdir /path/to/mount/point

In order to mount the bucket named `my-bucket`, invoke the gcsfuse binary
as follows:

    gcsfuse --key_file /path/to/key.json --bucket my-bucket --mount_point /path/to/mount/point

You should be able to see your bucket contents if you run `ls
/path/to/mount/point`. To later unmount the bucket, either kill the gcsfuse
process with a SIGINT or run `umount /path/to/mount/point`. (On Linux, you may
need to replace `umount` with `fusermount -u`.)


# Daemonization

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

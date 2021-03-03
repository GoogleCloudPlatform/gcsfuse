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

    gcloud auth application-default login
    gcloud auth login

Alternatively, you can set the `GOOGLE_APPLICATION_CREDENTIALS` environment
variable to the path to a JSON key file downloaded from the Google Developers
Console:

    GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]

When mounting with an fstab entry, you can use the `key_file` option. For example:

    my-bucket /mount/point gcsfuse rw,noauto,user,key_file=/path/to/key.json

[gce]: https://cloud.google.com/compute/
[gce-service-accounts]: https://cloud.google.com/compute/docs/authentication
[gcloud tool]: https://cloud.google.com/sdk/gcloud/
[app-default-credentials]: https://developers.google.com/identity/protocols/application-default-credentials#howtheywork


# Basic usage

## Mounting

Say you want to mount the GCS bucket called `my-bucket`. First create the
directory into which you want to mount the gcsfuse bucket, then run gcsfuse:

    mkdir /path/to/mount/point
    gcsfuse my-bucket /path/to/mount/point

**Important**: You should run gcsfuse as the user who will be using the file
system, not as root. Similarly, the directory should be owned by that user. Do
not use `sudo` for either of the steps above or you will wind up with
permissions issues.

After the gcsfuse tool exits, you should be able to see your bucket contents if
you run `ls /path/to/mount/point`. If you would prefer the tool to stay in the
foreground (for example to see debug logging), run it with the `--foreground`
flag.

## Unmounting

On Linux, unmount using fuse's `fusermount` tool:

    fusermount -u /path/to/mount/point

On OS X, unmount like any other file system:

    umount /path/to/mount/point


## Logging

Use flags like `--debug_gcs`, `--debug_fuse` and `--debug_http` to get 
additional logs from GCS, Fuse, and HTTP requests.

When gcsfuse is run in the foreground, all the logs are printed to stdout and 
stderr. When it is in the background, only a few lines of logs indicating the 
mounting status would be printed to stdout or stderr.

If you need logs when running gcsfuse in the background, please use `--log-file`
to specify a log file. The directory of the log file must exist.

# Access permissions

As a security measure, fuse itself restricts file system access to the user who
mounted the file system (cf. [fuse.txt][fuse-security]). For this reason,
gcsfuse by default shows all files as owned by the invoking user. Therefore you
should invoke gcsfuse as the user that will be using the file system, not as
root.

If you know what you are doing, you can override these behaviors with the
[`allow_other`][allow_other] mount option supported by fuse and with the
`--uid` and `--gid` flags supported by gcsfuse. Be careful, this may have
security implications!

[fuse-security]: https://github.com/torvalds/linux/blob/a33f32244/Documentation/filesystems/fuse.txt#L253-L300
[allow_other]: https://github.com/torvalds/linux/blob/a33f32244/Documentation/filesystems/fuse.txt#L100-L105


# mount(8) and fstab compatibility

The gcsfuse [installation process](installing.md) installed a helper understood
by the `mount` command to your system at one of these two paths, depending on
your operating system:

*   Linux: `/sbin/mount.gcsfuse`
*   OS X: `/sbin/mount_gcsfuse`

On OS X, this program allows you to mount buckets using the `mount` command.
(On Linux, only root can do this.) For example:

    mount -t gcsfuse -o rw,user my-bucket /path/to/mount/point

The following mount options are supported, in addition to the standard ones for
your system, matching the semantics of the corresponding `gcsfuse` flags named
with dashes instead of underscores:

*   `implicit_dirs`
*   `dir_mode`
*   `file_mode`
*   `key_file`
*   `temp_dir`
*   `uid`
*   `gid`
*   `only_dir`
*   `limit_ops_per_sec`
*   `limit_bytes_per_sec`
*   `stat_cache_ttl`
*   `type_cache_ttl`
*   `billing_project`

On both OS X and Linux, you can also add entries to your `/etc/fstab` file like
the following:

    my-bucket /mount/point gcsfuse rw,noauto,user

Afterward, you can run `mount /mount/point` as a non-root user.

The `noauto` option above specifies that the file system should not be mounted
at boot time. 

If you would prefer to mount the file system automatically, you may need to pass 
the `x-systemd.requires=network-online.target` or `_netdev` option to ensure that gcsfuse waits
for the network system to be ready prior to mounting.

    my-bucket /mount/point gcsfuse rw,x-systemd.requires=network-online.target,user

You can also mount the file system automatically as a non-root user by
specifying the options `uid` and/or `gid`:

    my-bucket /mount/point gcsfuse rw,_netdev,allow_other,uid=1001,gid=1001

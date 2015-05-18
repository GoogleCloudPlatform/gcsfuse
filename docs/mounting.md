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

# Prerequisites

1. Before invoking Cloud Storage FUSE, you must have a Cloud Storage bucket that you want to mount. If you haven't yet, [create](https://cloud.google.com/storage/docs/creating-buckets#storage-create-bucket-console) a storage bucket. 
2. Provide [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials#howtheywork) to authenticate Cloud Storage FUSE requests to Cloud Storage. By default, Cloud Storage FUSE automatically loads the Application Default Credentials without any further configuration if they exist. You can use the gcloud auth login command to easily generate Application Default Credentials.

        gcloud auth application-default login
        gcloud auth login
        gcloud auth list

   Alternatively, you can authenticate Cloud Storage FUSE by setting the --key-file flag to the path of a JSON key file, which you can download from the Google Cloud console. You can also set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of the JSON key.

        GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]
When mounting with an fstab entry, use the key_file option:

    my-bucket /mount/point gcsfuse rw,noauto,user,key_file=/path/to/key.json

When you create a Compute Engine VM, its service account can also be used to authenticate access to Cloud Storage FUSE. When Cloud Storage FUSE is run from such a VM, it automatically has access to buckets owned by the same project as the VM.

# Basic syntax for mounting

The base syntax for using Cloud Storage FUSE is:

    gcsfuse [global options] [bucket] mountpoint

Where [global options] are optional specific flags you can pass (use gcsfuse --help), [bucket] is the optional name of your bucket, and ‘mountpoint’ is the directory on your machine that you are mounting the bucket to. For example:
    
    gcsfuse my-bucket /usr/me/myself

# Static Mounting

Static mounting means mounting a specific bucket. For example, say I want to mount the bucket “my-bucket” to the directory /usr/me/myself

    mkdir /usr/me/myself
    gcsfuse my-bucket /usr/me/myself
Note: Avoid using the name of the bucket as the local directory mount point name.

# Dynamic Mounting

Dynamic mounting dynamically mounts all buckets a user has access to as subdirectories, without passing a specific bucket name.
   
    mkdir /usr/me/myself
    gcsfuse /usr/me/myself

As an example, let’s say a user has access to ‘my-bucket’ ‘my-bucket2’ and ‘my-bucket3’. By not passing the specific bucket name the buckets will be dynamically mounted. The individual buckets can be accessed as a subdirectory:

    ls /usr/me/myself/my-bucket/
    ls /usr/me/myself/my-bucket-2/
    ls /usr/me/myself/my-bucket-3/

Dynamically mounted buckets do not allow listing subdirectories at the root mount point, and bucket names must be specified in order to be accessed.
    
    ls /usr/me/myself/
    ls: reading directory .: Operation not supported

    ls /usr/me/myself/my-bucket-1
    foo.txt

# Mounting as read-only

Cloud Storage FUSE supports mounting as read-only by passing -o ro as a global option flag:
mkdir /usr/me/myself

    gcsfuse -o ro my-bucket /usr/me/myself 

# Mounting a specific directory in a Cloud Storage bucket instead of the entire bucket

By default, Cloud Storage FUSE mounts the entire contents and directory structure within a bucket. To mount only a specific directory, pass the --only-dir option. For example, if ‘my-bucket’ contains the path ‘my-bucket/a/b’ to mount only a/b to my local directory /usr/me/myself:

    gcsfuse --only-dir my-bucket a/b /usr/me/myself

# General filesystem mount options

Most of the generic mount options described in mount are supported, and can be passed along with the -o flag, such as ro, rw, suid, nosuid, dev, nodev, exec, noexec, atime, noatime, sync, async, dirsync. See [here](https://man7.org/linux/man-pages/man8/mount.fuse3.8.html) for additional information. For example

    gcsfuse -o ro my-bucket /usr/me/myself

# Foreground

After Cloud Storage FUSE exits, you should be able to see your bucket contents if you run ls /path/to/mount/point. If you would prefer the tool to stay in the foreground (for example to see debug logging), run it with the --foreground flag.

# Unmounting

On Linux, unmount using fuse's fusermount tool:

    fusermount -u /path/to/mount/point

# Logging

Use flags like --debug_gcs, --debug_fuse, --debug_http, --debug_fs, and --debug_mutex to get additional logs from Cloud Storage FUSE, and HTTP requests.

Cloud Storage FUSE logs its activity to a file if the user specifies one with --log-file flag. Otherwise, it logs to stdout in the foreground and to syslog in background mode. In addition you can use --log-format to specify the format as json or text. The directory of the log file must pre-exist.

Note: Cloud Storage FUSE prints a few lines of logs indicating the mounting status to stdout or stderr.

To support the log-rotation please follow the instructions [here](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/logging.md).

# Persisting a mount

The Cloud Storage FUSE installation process installs a helper understood by the mount command to your system at the path /sbin/mount.gcsfuse, allowing you to mount your bucket using the mount command (on Linux, only root can do this). Example:

    mount -t gcsfuse -o rw,user my-bucket /path/to/mount/point

You can also add entries to your /etc/fstab file like the following:

    my-bucket /mount/point gcsfuse rw,noauto,user

Afterward, you can run mount /mount/point as a non-root user.

The noauto option above specifies that the file system should not be mounted at boot time.

If you would prefer to mount the file system automatically, you may need to pass the x-systemd.requires=network-online.target or _netdev option to ensure that Cloud Storage FUSE waits for the network system to be ready prior to mounting.

    my-bucket /mount/point gcsfuse 
    rw,x-systemd.requires=network-online.target,user

You can also mount the file system automatically as a non-root user by specifying the options uid and/or gid:

    my-bucket /mount/point gcsfuse rw,_netdev,allow_other,uid=1001,gid=1001

# Directory semantics

Cloud Storage FUSE presents directories logically using “/” prefixes. Cloud Storage object names map directly to file paths using the separator '/'. Object names ending in a slash represent a directory, and all other object names represent a file. Directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.

Please see the Files and Directories section under docs/semantics for more details, including how to mount a bucket with existing prefixes. 

# Ownership

Cloud Storage FUSE should run as the user who will be using the file system, not as root. Similarly, the directory should be owned by that user. Do not use sudo for either of the steps above or you will wind up with permissions issues.

# Access permissions

By default, the access to the Cloud Storage FUSE mount is restricted to the user that mounts it, which is a security measure implemented within the FUSE kernel. For this reason, Cloud Storage FUSE by default shows all files as owned by the invoking user. Therefore you should invoke Cloud Storage FUSE as the user that will be using the file system, not as root.

To allow others to access the GCSFuse mount, use the ‘allow_other’ mounting option at the time of mounting (‘-o allow_other’).

     mount -t gcsfuse -o allow_other my-bucket /usr/me/myself

If the user mounting the Cloud Storage FUSE is not root then the ‘allow_other’ requires 'user_allow_other' to be added to the /etc/fuse.conf file.

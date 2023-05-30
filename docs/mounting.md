# Prerequisites

1. Before invoking Cloud Storage FUSE, you must have a Cloud Storage bucket that you want to mount. If you haven't yet, [create](https://cloud.google.com/storage/docs/creating-buckets#storage-create-bucket-console) a storage bucket. 
2. Provide [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials#howtheywork) to authenticate Cloud Storage FUSE requests to Cloud Storage. By default, Cloud Storage FUSE automatically loads the Application Default Credentials without any further configuration if they exist. You can use the gcloud auth login command to easily generate Application Default Credentials.

        gcloud auth application-default login
        gcloud auth login
        gcloud auth list

   Alternatively, you can authenticate Cloud Storage FUSE using a service account
key.

   - [Create a service account](https://cloud.google.com/iam/docs/service-accounts-create)
  with one of the following roles:

     * `Storage Object Viewer (roles/storage.objectViewer)` role to mount a
    bucket with read-only permissions.
     * `Storage Object Admin (roles/storage.objectAdmin)` role to mount a bucket with read-write permissions.

   - [Create and download the service account key](https://cloud.google.com/iam/docs/keys-create-delete#iam-service-account-keys-create-console)
  and set the ```--key-file``` flag to the path of the downloaded JSON key file while
  mounting the bucket.

           gcsfuse --key-file <path to service account key> [bucket] /path/to/mount/point

  You can also set the ```GOOGLE_APPLICATION_CREDENTIALS``` environment
  variable to the path of the JSON key.

           GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]
When mounting with an fstab entry, use the key_file option:

    my-bucket /mount/point gcsfuse rw,noauto,user,key_file=/path/to/key.json

When you create a Compute Engine VM, its service account can also be used to authenticate access to Cloud Storage FUSE. When Cloud Storage FUSE is run from such a VM, it automatically has access to buckets owned by the same project as the VM.

# Basic syntax for mounting

The base syntax for using Cloud Storage FUSE is:

    gcsfuse [global options] [bucket] mountpoint

Where [global options] are optional specific flags you can pass (use ```gcsfuse --help```), [bucket] is the optional name of your bucket, and ‘mountpoint’ is the directory on your machine that you are mounting the bucket to. For example:
    
    gcsfuse my-bucket /path/to/mount/point

# Static Mounting

Static mounting means mounting a specific bucket. For example, say I want to mount the bucket ```my-bucket``` to the directory ```/path/to/mount/point```

    mkdir /path/to/mount/point
    gcsfuse my-bucket /path/to/mount/point
Note: Avoid using the name of the bucket as the local directory mount point name.

# Dynamic Mounting

Dynamic mounting dynamically mounts all buckets a user has access to as subdirectories, without passing a specific bucket name.
   
    mkdir /path/to/mount/point
    gcsfuse /path/to/mount/point

As an example, let’s say a user has access to ‘my-bucket’ ‘my-bucket2’ and ‘my-bucket3’. By not passing the specific bucket name the buckets will be dynamically mounted. The individual buckets can be accessed as a subdirectory:

    ls /path/to/mount/point/my-bucket/
    ls /path/to/mount/point/my-bucket-2/
    ls /path/to/mount/point/my-bucket-3/

Dynamically mounted buckets do not allow listing subdirectories at the root mount point, and bucket names must be specified in order to be accessed.
    
    ls /path/to/mount/point
    ls: reading directory .: Operation not supported

    ls /path/to/mount/point/my-bucket-1
    foo.txt

# Mounting as read-only

Cloud Storage FUSE supports mounting as read-only by passing ```-o ro``` as a global option flag:
mkdir ```/path/to/mount/point```

    gcsfuse -o ro my-bucket  /path/to/mount/point 

# Mounting a specific directory in a Cloud Storage bucket instead of the entire bucket

By default, Cloud Storage FUSE mounts the entire contents and directory structure within a bucket. To mount only a specific directory, pass the ```--only-dir``` option. For example, if ```my-bucket``` contains the path ```my-bucket/a/b``` to mount only a/b to my local directory ```/path/to/mount/point```:

    gcsfuse --only-dir a/b my-bucket /path/to/mount/point

# General filesystem mount options

Most of the generic mount options described in mount are supported, and can be passed along with the ```-o``` flag, such as ```ro```, ```rw```, ```suid```, ```nosuid```, ```dev```, ```nodev```, ```exec```, ```noexec```, ```atime```, ```noatime```, ```sync```, ```async```, ```dirsync```. See [here](https://man7.org/linux/man-pages/man8/mount.fuse3.8.html) for additional information. For example

    gcsfuse -o ro my-bucket  /path/to/mount/point

# Foreground

After Cloud Storage FUSE exits, you should be able to see your bucket contents if you run ls ```/path/to/mount/point```. If you would prefer the tool to stay in the foreground (for example to see debug logging), run it with the ```--foreground``` flag.

# Unmounting

On Linux, unmount using fuse's fusermount tool:

    fusermount -u /path/to/mount/point

# Logging

Use flags like ```--debug_gcs```, ```--debug_fuse```, ```--debug_http```, ```--debug_fs```, and ```--debug_mutex``` to get additional logs from Cloud Storage FUSE, and HTTP requests.

Cloud Storage FUSE logs its activity to a file if the user specifies one with ```--log-file``` flag. Otherwise, it logs to stdout in the foreground and to syslog in background mode. In addition you can use ```--log-format``` to specify the format as json or text. The directory of the log file must pre-exist.

Note: Cloud Storage FUSE prints a few lines of logs indicating the mounting status to stdout or stderr.

To support the log-rotation please follow the instructions [here](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/logging.md).

# Persisting a mount

The Cloud Storage FUSE installation process installs a helper understood by the mount command to your system at the path ```/sbin/mount```.gcsfuse, allowing you to mount your bucket using the mount command (on Linux, only root can do this). Example:

    mount -t gcsfuse -o rw,user my-bucket /path/to/mount/point

You can also add entries to your ```/etc/fstab``` file like the following:

    my-bucket /mount/point gcsfuse rw,noauto,user

Afterward, you can run mount ```/mount/point``` as a non-root user.

The noauto option above specifies that the file system should not be mounted at boot time.

If you would prefer to mount the file system automatically, you may need to pass the x-systemd.requires=network-online.target or _netdev option to ensure that Cloud Storage FUSE waits for the network system to be ready prior to mounting.

    my-bucket /mount/point gcsfuse 
    rw,x-systemd.requires=network-online.target,user

You can also mount the file system automatically as a non-root user by specifying the options ```uid``` and/or ```gid```:

    my-bucket /mount/point gcsfuse rw,_netdev,allow_other,uid=1001,gid=1001

# Directory semantics

Cloud Storage FUSE presents directories logically using “/” prefixes. Cloud Storage object names map directly to file paths using the separator '/'. Object names ending in a slash represent a directory, and all other object names represent a file. Directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.

Please see the Files and Directories section under docs/semantics for more details, including how to mount a bucket with existing prefixes. 

# Ownership

Cloud Storage FUSE should run as the user who will be using the file system, not as root. Similarly, the directory should be owned by that user. Do not use sudo for either of the steps above or you will wind up with permissions issues.

# Access permissions

By default, the access to the Cloud Storage FUSE mount is restricted to the user that mounts it, which is a security measure implemented within the FUSE kernel. For this reason, Cloud Storage FUSE by default shows all files as owned by the invoking user. Therefore you should invoke Cloud Storage FUSE as the user that will be using the file system, not as root.

To allow others to access the GCSFuse mount, use the ```allow_other``` mounting option at the time of mounting (```-o allow_other```).

     mount -t gcsfuse -o allow_other my-bucket /path/to/mount/point

If the user mounting the Cloud Storage FUSE is not root then the ```allow_other``` requires ```user_allow_other``` to be added to the ```/etc/fuse.conf``` file.

# Full list of mount options

Type ```gcsfuse --help``` to see the full list:

| Option                                               | Description                                                                                                                                                                                                                                                                                                                                                                                  | 
|------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| --app-name value                                     | The application name of this mount.                                                                                                                                                                                                                                                                                                                                                          |
| --foreground                                         | Stay in the foreground after mounting.                                                                                                                                                                                                                                                                                                                                                       |
| -o value                                             | Additional system-specific mount options                                                                                                                                                                                                                                                                                                                                                     |
| -o ro                                                | Mount as read-only                                                                                                                                                                                                                                                                                                                                                                           |
| --dir-mode value                                     | Permissions bits for directories, in octal. (default: 755)                                                                                                                                                                                                                                                                                                                                   |
| --file-mode value                                    | Permission bits for files, in octal. (default: 644)                                                                                                                                                                                                                                                                                                                                          |
| --uid value                                          | UID owner of all inodes. (default: -1)                                                                                                                                                                                                                                                                                                                                                       |
| --gid value                                          | GID owner of all inodes. (default: -1)                                                                                                                                                                                                                                                                                                                                                       |
| --implicit-dirs                                      | Implicitly define directories based on content. See files and directories in docs/semantics for more information                                                                                                                                                                                                                                                                             |
| --only-dir value                                     | Mount only a specific directory within the bucket. See docs/mounting for more information                                                                                                                                                                                                                                                                                                    |
| --rename-dir-limit value                             | Allow rename a directory containing fewer descendants than this limit. (default: 0)                                                                                                                                                                                                                                                                                                          |
| --endpoint value                                     | The endpoint to connect to. (default: "https://storage.googleapis.com:443")                                                                                                                                                                                                                                                                                                                  |
| --billing-project value                              | Project to use for billing when accessing a bucket enabled with “Requester Pays” (default: none)                                                                                                                                                                                                                                                                                             |
| --key-file value                                     | Absolute path to JSON key file for use with GCS. (default: none, Google application default credentials used)                                                                                                                                                                                                                                                                                |
| --token-url value                                    | A url for getting an access token when the key-file is absent.                                                                                                                                                                                                                                                                                                                               |
| --reuse-token-from-url                               | If false, the token acquired from token-url is not reused.                                                                                                                                                                                                                                                                                                                                   |
| --limit-bytes-per-sec value                          | Bandwidth limit for reading data, measured over a 30-second window. (use -1 for no limit) (default: -1)                                                                                                                                                                                                                                                                                      |
| --limit-ops-per-sec value                            | Operations per second limit, measured over a 30-second window (use -1 for no limit) (default: -1)                                                                                                                                                                                                                                                                                            |
| --sequential-read-size-mb value                      | File chunk size to read from GCS in one call. Need to specify the value in MB. ChunkSize less than 1MB is not supported (default: 200)                                                                                                                                                                                                                                                       |
| --max-retry-sleep value                              | The maximum duration allowed to sleep in a retry loop with exponential backoff for failed requests to GCS backend. Once the backoff duration exceeds this limit, the retry stops. The default is 1 minute. A value of 0 disables retries. (default: 1m0s)                                                                                                                                    |
| --stat-cache-capacity value                          | How many entries can the stat cache hold (impacts memory consumption) (default: 4096)                                                                                                                                                                                                                                                                                                        |
| --stat-cache-ttl value                               | How long to cache StatObject results and inode attributes. (default: 1m0s)                                                                                                                                                                                                                                                                                                                   |
| --type-cache-ttl value                               | How long to cache name -> file/dir mappings in directory inodes. (default: 1m0s)                                                                                                                                                                                                                                                                                                             |
| --max-retry-duration value                           | The operation will be retried till the value of max-retry-duration. (default: 30s)                                                                                                                                                                                                                                                                                                           |
| --retry-multiplier value                             | Param for exponential backoff algorithm, which is used to increase waiting time b/w two consecutive retries. (default: 2)                                                                                                                                                                                                                                                                    |
| --http-client-timeout value                          | The time duration that http client will wait to get response from the server. The default value 0 indicates no timeout.  (default: 0s)                                                                                                                                                                                                                                                       |
| --experimental-local-file-cache                      | Experimental: Cache GCS files on local disk for reads.                                                                                                                                                                                                                                                                                                                                       |
| --temp-dir value                                     | Path to the temporary directory where writes are staged prior to upload to Cloud Storage. (default: system default, likely /tmp)                                                                                                                                                                                                                                                             |
| --client-protocol value                              | The protocol used for communicating with the GCS backend. Value can be 'http1' (HTTP/1.1) or 'http2' (HTTP/2). (default: http1)                                                                                                                                                                                                                                                              |
| --max-conns-per-host value                           | The max number of TCP connections allowed per server. This is effective when --client-protocol is set to 'http1'. (default: 100)                                                                                                                                                                                                                                                             |
| --max-idle-conns-per-host value                      | The number of maximum idle connections allowed per server (default: 100)                                                                                                                                                                                                                                                                                                                     |
| --enable-nonexistent-type-cache                      | Once set, if an inode is not found in GCS, a type cache entry with type NonexistentType will be created. This also means new file/dir created might not be seen. For example, if this flag is set, and flag type-cache-ttl is set to 10 minutes, then if we create the same file/node in the meantime using the same mount, since we are not refreshing the cache, it will still return nil. |
| --stackdriver-export-interval value                  | Experimental: Export metrics to stackdriver with this interval. The default value 0 indicates no exporting. (default: 0s)                                                                                                                                                                                                                                                                    |
| --experimental-opentelemetry-collector-address value | Experimental: Export metrics to the OpenTelemetry collector at this address.                                                                                                                                                                                                                                                                                                                 |
| --log-file value                                     | The file for storing logs that can be parsed by fluentd. When not provided, plain text logs are printed to stdout.                                                                                                                                                                                                                                                                           |
| --log-format value                                   | The format of the log file: 'text' or 'json'. (default: "json")                                                                                                                                                                                                                                                                                                                              |
| --debug_fuse_errors                                  | If false, fuse errors will not be logged to the console (in case of --foreground) or the log-file (if specified)                                                                                                                                                                                                                                                                             |
| --debug_fuse                                         | Enable fuse-related debugging output.                                                                                                                                                                                                                                                                                                                                                        |
| --debug_fs                                           | This flag is currently unused.                                                                                                                                                                                                                                                                                                                                                               |
| --debug_gcs                                          | Print GCS request and timing information.                                                                                                                                                                                                                                                                                                                                                    |
| --debug_http                                         | Dump HTTP requests and responses to/from GCS, doesn't work when --enable-storage-client-library flag is true.                                                                                                                                                                                                                                                                                |
| --debug_invariants                                   | Panic when internal invariants are violated.                                                                                                                                                                                                                                                                                                                                                 |
| --debug_mutex                                        | Print debug messages when a mutex is held too long.                                                                                                                                                                                                                                                                                                                                          |
| --enable-storage-client-library                      | If true, will use go storage client library otherwise jacobsa/gcloud                                                                                                                                                                                                                                                                                                                         |
| --help, -h                                           | Show help                                                                                                                                                                                                                                                                                                                                                                                    |
| --version, -v                                        | Print version                                                                                                                                                                                                                                                                                                                                                                                |

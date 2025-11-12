# Troubleshooting for production issues

This page enumerates some common user facing issues around GCSFuse and also
discusses potential solutions to the same.

### Generic Mounting Issue

Most of the common mount point issues are around permissions on both local mount point and the Cloud Storage bucket. It is highly recommended to retry with --foreground --log-severity=TRACE flags which would provide much more detailed logs to understand the errors better and possibly provide a solution.

### Mount successful but files are not visible

Try mounting the gcsfuse with `--implicit-dirs` flag. Read the [semantics](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#files-and-directories) to know the reasoning.

### Mount failed with fusermount3 exit status 1

- It can come when the bucket is already mounted in a folder, and we try to mount it again. You need to unmount first and then remount.
- It can also happen if you're trying to mount the bucket on a directory that has read-only permissions. Please provide write permissions to the directory and try mounting it again. You can use the below command to grant write permissions.
  ```
    chmod 755 mount_point
  ```

### version GLIBC_x.yz not found

GCSFuse should not be linking to glibc. Please either `export CGO_ENABLED=0` in your environment or prefix `CGO_ENABLED=0` to any <code>go build&#124;run&#124;test</code> commands that you're invoking.

### Mount get stuck with error: DefaultTokenSource: google: could not find default credentials

Run ```gcloud auth application-default login``` command to fetch default credentials to the VM. This will fetch the credentials to the following locations: <ol type="a"><li>For linux - $HOME/.config/gcloud/application_default_credentials.json</li><li>For windows - %APPDATA%/gcloud/application_default_credentials.json </li></ol>

### Input/Output Error

It’s a generic error, but the most probable culprit is the bucket not having the right permission for Cloud Storage FUSE to operate on. Ref - [here](https://stackoverflow.com/questions/36382704/gcsfuse-input-output-error)

### Stale File Handle Error (ESTALE)

This error occurs when GCSFuse detects that a file has been modified or deleted on GCS by another process or mount since it was opened. This is a data integrity feature that prioritizes providing clear indications of potential conflicts, preventing silent data loss.

To avoid this, it's best to prevent multiple sources from modifying the same object simultaneously. For a detailed explanation of the scenarios that can cause this error, please refer to the [Stale File Handle Errors](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#stale-file-handle-errors) section in our semantics documentation.

### Generic NO_PUBKEY Error - while installing Cloud Storage FUSE on ubuntu 22.04

It happens while running - ```sudo apt-get update``` - working on installing Cloud Storage FUSE. You just have to add the pubkey you get in the error using the below command: ```sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys <PUBKEY> ``` And then try running ```sudo apt-get update```

### Cloud Storage FUSE fails with Docker container

Though not tested extensively, the [community](https://stackoverflow.com/questions/65715624/permission-denied-with-gcsfuse-in-unprivileged-ubuntu-based-docker-container) reports that Cloud Storage FUSE works only in privileged mode when used with Docker. There are [solutions](https://cloud.google.com/iam/docs/service-account-overview) which exist and claim to do so without privileged mode, but these are not tested by the Cloud Storage FUSE team

### SetUpBucket: OpenBucket: Bad credentials for bucket BUCKET_NAME: permission denied


Log may look similar to this ```daemonize.Run: readFromProcess: sub-process: mountWithArgs: mountWithStorageHandle: fs.NewServer: create file system: SetUpBucket: OpenBucket: Bad credentials for bucket BUCKET_NAME: permission denied```

Check the bucket name. Make sure it is within your project. Make sure the applied roles on the bucket  contain storage.objects.list permission. You can refer to them [here](https://cloud.google.com/storage/docs/access-control/iam-roles).

### SetUpBucket: OpenBucket: Unknown bucket BUCKET_NAME: no such file or directory

Log may look similar to this ```daemonize.Run: readFromProcess: sub-process: mountWithArgs: mountWithStorageHandle: fs.NewServer: create file system: SetUpBucket: OpenBucket: Unknown bucket BUCKET_NAME: no such file or directory```

Check the bucket name. Make sure the [service account](https://www.google.com/url?q=https://cloud.google.com/iam/docs/service-accounts&sa=D&source=docs&ust=1679992003850814&usg=AOvVaw3nJ6wNQK4FZdgm8gBTS82l) has permissions to access the files. It must at least have the permissions of the Storage Object Viewer role.

### mount: running fusermount: exit status 1 stderr: /bin/fusermount: fuse device not found, try 'modprobe fuse' first

Log may look similar to this - ```daemonize.Run: readFromProcess: sub-process: mountWithArgs: mountWithStorageHandle: Mount: mount: running fusermount: exit status 1 stderr: /bin/fusermount: fuse device not found, try 'modprobe fuse' first```

To run the container locally, add the --privilege flag to the docker run command: ```docker run --privileged  gcr.io/PROJECT/my-fs-app ``` <ul><li>You must create a local mount directory</li> <li>If you want all the logs from the mount process use the --foreground flag in combination with the mount command: ```gcsfuse --foreground --log-severity=TRACE $GCSFUSE_BUCKET $MNT_DIR ``` </li><li> Add --log-severity=TRACE for enabling debug logs</li></ul>

### Cloud Storage FUSE installation fails with an error at build time.

Only specific OS distributions are currently supported, learn more about [Installing Cloud Storage FUSE](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md).

### Cloud Storage FUSE not mounting after reboot when entry is present in ```/etc/fstab``` with 1 or 2 as fsck order

Pass [_netdev option](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/mounting.md#persisting-a-mount) in fstab entry (reference issue [here](https://github.com/GoogleCloudPlatform/gcsfuse/issues/1043)). With this option, mount will be attempted on reboot only when network is connected.

### Cloud Storage FUSE get stuck when using it to concurrently work with a large number of opened files (reference issue [here](https://github.com/GoogleCloudPlatform/gcsfuse/issues/1043))

This happens when gcsfuse is mounted with http1 client (default) and the application using gcsfuse tries to keep more than value of `--max-conns-per-host` number of files opened. You can try (a) Passing a value higher than the number of files you want to keep open to `--max-conns-per-host` flag. (b) Adding some timeout for http client connections using `--http-client-timeout` flag.

### Permission Denied error.

Please refer [here](https://cloud.google.com/storage/docs/gcsfuse-mount#authenticate_by_using_a_service_account) to know more about permissions
(e.g.  **Issue**:mkdir: cannot create directory ‘gcs/test’: Permission denied. User can check specific errors by enabling logs with --log-severity=TRACE flags.  
**Solution** - depending upon the use-case, you can choose one of the following options.
* If you are explicitly authenticating for a specific service account by providing say a key-file, then make sure that the service account has appropriate IAM role for the operation e.g. roles/storage.objectAdmin, roles/storage.objectUser
* If you are using the default service account i.e. not specifying a key-file, then ensure that
    * The VM's service account has got the required IAM roles for the operation e.g. roles/storage.objectUser to allow read-write access.
    * The VM's scope has been appropriately set. You can set the scope to storage-full to give the VM full-access to the cloud-storage buckets. For this:
        * Turn-off the instance
        * Change the VM's scope either by using the GCP console or by executing  `gcloud beta compute instances set-scopes INSTANCE_NAME --scopes=storage-full`
        * Start the instance

### Bad gateway error while installing/upgrading GCSFuse:
`Err: http://packages.cloud.google.com/apt gcsfuse-focal/main amd64 gcsfuse amd64 1.2.0`<br/>`502  Bad Gateway [IP: xxx.xxx.xx.xxx 80]`

This error is seen when the url used in /etc/apt/sources.list.d/gcsfuse.list file uses HTTP protocol instead of HTTPS protocol. Run the following commands to update /etc/apt/sources.list.d/gcsfuse.list file with the https:// url.<br/> <code>$ sudo rm /etc/apt/sources.list.d/gcsfuse.list</code> <br/> <code>$ export GCSFUSE_REPO=gcsfuse-$(lsb_release -c -s)</code> <br/> <code>$ echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" &#124; sudo tee /etc/apt/sources.list.d/gcsfuse.list </code>

### Repository changed 'Origin' and 'Label' error while running apt-get update command:

<br/>`E: Repository 'http://packages.cloud.google.com/apt gcsfuse-focal InRelease' changed its 'Origin' value from 'gcsfuse-jessie' to 'namespaces/gcs-fuse-prod/repositories/gcsfuse-focal'`<br/>`E: Repository 'http://packages.cloud.google.com/apt gcsfuse-focal InRelease' changed its 'Label' value from 'gcsfuse-jessie' to 'namespaces/gcs-fuse-prod/repositories/gcsfuse-focal'`<br/>`N: This must be accepted explicitly before updates for this repository can be applied. See apt-secure(8) manpage for details. `

Use one of the following commands to upgrade to latest GCSFuse version<br/> `sudo apt-get update --allow-releaseinfo-change `<br/>OR<br/>`sudo apt update -y && sudo apt-get update`

### Unable to unmount or stop GCSFuse

Unable to unmount or stop GCSFuse due to an error message like:`fusermount: failed to unmount: Device or resource busy` or `umount: /path/to/mountpoint: target is busy`.</br>This typically indicates active processes are using files or directories within the GCSFuse mount.<br/>
Find the process ID of GCSFuse:<br/>`BUCKET=<Enter your bucket name>`</br>` MOUNT_POINT=<Enter your mount point>`</br>`PID=$(ps -aux &#124; grep "gcsfuse.*$BUCKET.*$MOUNT_POINT" &#124; grep -v grep &#124; tr -s ' ' &#124; cut -d' ' -f2)`</br>Kill the GCSFuse process:</br>`sudo kill -SIGKILL "$PID"`</br>Unmount GCSFuse</br>`fusermount -u $"MOUNT_POINT"`

### mount: exec: "fusermount": executable file not found in $PATH`

Unable to mount with the following error `daemonize.Run: readFromProcess: sub-process: Error while mounting gcsfuse: mountWithArgs: mountWithStorageHandle: Mount: mount: exec: "fusermount": executable file not found in $PATH`

Fuse package is not installed. It may throw this error. Run the following command to install fuse <br/> <code> sudo apt-get install fuse </code> for Debian/Ubuntu <br/> <code> sudo yum install fuse </code> for RHEL/CentOs/Rocky <br/>

### Encountered unsupported prefixes during listing, or ls: reading directory \<mountpath>/\<path>:  Input/output error
Unable to list unsupported objects in a mounted bucket, i.e. objects which have names with `//` in them or have names starting with `/` e.g. `gs://<bucket>//A` or `gs://<bucket>/A//B` etc. Such objects can be listed by the following command: `gcloud storage ls gs://<bucket>/**//**`.

This error can be mitigated in one of the following two ways.

* Move/Rename such objects e.g. for objects of names like `A//B`, use `gcloud storage mv gs://<bucket>/A//* gs://<bucket>/A/`), Or
* Delete such objects e.g. for objects of names like `A//B`, use `gcloud storage rm gs://<bucket>/A//**`.

Refer [semantics](semantics.md#unsupported-object-names) for more details.

### Experiencing hang while executing "ls" on a directory containing large number of files/directories.

By default `ls` does listing but sometimes additionally does `stat` for each list entry. Additional stat on each entry slows down the entire `ls` operation and may look like hang, but in reality is iterating one by one and slow. There are two ways to overcome this: <ul><li>**Get rid of `stat`:** Execute without coloring `ls --color=never` or `\ls` to remove the stat part of `ls`.</li> <li>**Improve the `stat` lookup time:** increase the metadata ([stat](https://cloud.google.com/storage/docs/gcsfuse-cache#stat-cache-overview) + [type](https://cloud.google.com/storage/docs/gcsfuse-cache#type-cache-overview)) cache ttl/capacity which is populated as part of listing and makes the `stat` lookup faster. Important to note here, `ls` will be faster from the first execution itself as cache is populated in listing phase, leading to quicker stat lookup for each entry.</li></ul>

### GCSFuse Errors on Google Cloud Console Metrics Dashboard: "CANCELLED ReadObject" & "NOT_FOUND GetObjectMetadata"

Both these errors are expected and part of GCSFuse standard operating procedure. More details [here](https://github.com/GoogleCloudPlatform/gcsfuse/discussions/2300).

### GCSFuse logs showing errors for StatObject NotFoundError

`StatObject(\"<object_name>") (<time>ms): gcs.NotFoundError: storage: object doesn't exist"`.
This is an expected error. Please refer to **NOT_FOUND GetObjectMetadata** section [here](https://github.com/GoogleCloudPlatform/gcsfuse/discussions/2300#discussioncomment-10261838).

### Empty directory doesn't obey "--kernel-list-cache-ttl-secs" ttl.

In case a new file is added to the empty directory remotely, outside of the mount, the client will not be able to see the new file in listing even if ttl is expired.<br></br>Please don't use kernel-list-cache for multi node/mount-point file create scenario, this is recommended to be used only for read only workloads, e.g. for Serving and Training workloads. Or you need to create a new file/directory from the mount to see the newly added remote files.

### fuse: *fuseops.FallocateOp error: function not implemented or similar function not implemented errors

This is an expected error for file operations unsupported in FUSE file system (details [here](https://github.com/GoogleCloudPlatform/gcsfuse/discussions/2386#discussioncomment-10417635)).

### Error "transport endpoint is not connected"

It is possible customer is seeing the error "transport endpoint is not connected" because if the previous mount crashed or a mounted filesystem or device was abruptly disconnected or the mounting process failed unexpectedly, the system might not properly update its mount table. This leaves a stale entry referencing a resource that's no longer available.

**Solution:** Run the command `mount | grep "gcsfuse"`. If you find any entries, unmount the corresponding directory multiple times until all the entries cleared up and then try to remount the bucket.

**Additional troubleshooting steps:**

- Try to unmount and mount the mount-point using the command:
  `umount -f /<mount point>` && `mount /<mount point>`
- Try restarting/rebooting the VM Instance.

If it's running on GKE, the issue could be caused by an Out-of-Memory (OOM) error. Consider increasing the memory allocated to the GKE sidecar container. For more info refer [here](https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/main/docs/known-issues.md#implications-of-the-sidecar-container-design).

### GCSFuse crashes with `fatal error: sync: unlock of unlocked mutex` or `Panic: Inode 'a/' cannot have child file ''`

**Solution:** This happens when the mounting bucket contains an object with suffix `/\n` like, `gs://gcs-bkt/a/\n`
You need to find such objects and replace them with any other valid gcs object names. - [How](https://github.com/GoogleCloudPlatform/gcsfuse/discussions/2894)?

### Mount fails with 'can't create with 0 workers' when using buffered read

When using buffered reads (`--enable-buffered-read`) with `--read-global-max-blocks` set to `-1`, GCSFuse versions `v3.3.0` and `v3.4.0` fail to mount with an error similar to this:

```
Error: ... failed to create worker pool for buffered read: staticWorkerPool: can't create with 0 workers, priority: 0, normal: 0
```

This is a known issue and is fixed in later versions.

**Workaround:** If you are using an affected version, avoid setting `--read-global-max-blocks` to `-1`. Instead, set it to a large positive integer (e.g., `2147483647`) to approximate an infinite limit while avoiding the bug.

### OSError [ErrNo 28] No space left on device

The Writes in GCSFuse are staged locally before they are uploaded to GCS buckets. It takes up disk space equivalent to the size of the files that are being uploaded concurrently and deleted locally once they are uploaded. During this time, since the disk is used, this error may come up.

The path can be configured by using the mount flag [--temp-dir](https://cloud.google.com/storage/docs/cloud-storage-fuse/cli-options) to a path which has the disk space if available. By default, it takes the `/tmp` directory of the machine. (sometimes may be limited depending on the machine ).

Alternatively, from [GCSFuse version 2.9.1](https://github.com/GoogleCloudPlatform/gcsfuse/releases/tag/v2.9.1) onwards, writes can be configured with streaming writes feature ( which doesnt involve staging the file locally ) with the help of `--enable-streaming-writes` flag

### Permission Denied When Accessing a Mounted File or Directory

By default, GCSFuse assigns file-mode 0644 and dir-mode 0755 for mounted files and directories. As a result, other users (such as third-party clients or the root user) may not have the necessary permissions to access the mounted file system. To resolve this issue, you can modify the permissions using the following options:

- **Adjust File and Directory Permissions:**
  Use the `--file-mode` and `--dir-mode` flags to set the appropriate file and directory permissions when mounting.
- **Allow Access for Other Users:**
  To allow users other than the mounting user to access the bucket, use the `-o allow_other` flag during the mount process. Additionally, for this flag to function, the `user_allow_other` option must be enabled in the `/etc/fuse.conf` file, or the gcsfuse command must be run as the root user.

**Note:** Be aware that allowing access to other users can introduce potential [security risks](https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt#L218-L310). Therefore, it should be done with caution.

- **Set User and Group IDs:**
  Use the `--uid` and `--gid` flags to specify the correct user and group IDs for access.

Please note that GCSFuse does not support using `chmod` or similar commands to manage file access. For more detailed information, refer to the [Permissions and Ownership](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#permissions-and-ownership).


### Cloning GitHub repository inside mounted bucket is extremely slow
Ensure your GCS bucket mount configuration does not include `o=sync` or `o=dirsync`. \
During a Git clone, Git doesn’t just fetch object data, it builds out the entire .git directory structure, including initializing config, refs, hooks, and other internals. As part of this setup:

- Git repeatedly writes and updates .git/config, especially when setting remotes, branches, and defaults.
- Each update uses Git’s lock-write-rename-delete pattern to ensure consistency.

While using the mount configuration `o=sync,o=dirsync`, all modifications to the config file incur a network call due to enforced synchronous writes, resulting in
performance bottleneck. \
**Note** : There is no impact of disabling this mount configuration on the user workflow, since we avoid flushing data to GCS on sync( happens multiple times during the course of a Git clone ) , but only on close(), thus ensuring data persistence.

### Increased CPU Utilization with File Cache after upgrade to version 2.12.0
Starting with [version 2.12.0](https://github.com/GoogleCloudPlatform/gcsfuse/releases/tag/v2.12.0), you might observe a slight increase in CPU utilization when the file cache is enabled. This occurs because GCSFuse uses parallel threads to download data to the read cache. While this dramatically improves read performance, it may consume slightly more CPU than in previous versions.

If this increased CPU usage negatively impacts your workload's performance, you can disable this behavior by setting the `file-cache:enable-parallel-downloads` configuration option to `false`.

### Potential Stat Consistency Issues on high-performance machines with Default TTL
Starting with [version 3.0.0](https://github.com/GoogleCloudPlatform/gcsfuse/releases/tag/v3.0.0), On high-performance machines - gcsfuse will default to infinite stat cache TTL ([refer](https://cloud.google.com/storage/docs/cloud-storage-fuse/automated-configurations)), potentially causing stale file/directory information if the bucket is modified externally. If strict consistency is needed, manually set a finite TTL (e.g., --stat-cache-ttl 1m) to ensure metadata reflects recent changes. Consult [semantics](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md) doc for more details.

### Writes still using legacy staged writes even though streaming writes are enabled.
If you observe that GCSFuse is still utilizing staged writes despite streaming writes being enabled, several factors could be at play.

- **Concurrent Streaming Write Limit Reached:** When the number of concurrent streaming writes exceeds the configured limit (`--write-global-max-blocks`), GCSFuse automatically uses legacy staged writes for concurrent file writes above this limit. You will see an warnining log message when this happens, similar to:
  > File <var>file_name</var> will use legacy staged writes because concurrent streaming write limit (set by --write-global-max-blocks) has been reached.

  This is not an error, but a fallback mechanism to manage memory usage. If your system has sufficient memory, you can increase the number of allowed concurrent streaming writes by adjusting the `--write-global-max-blocks` flag to prevent this warning. For memory usage refer [write semantics](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#writes) doc for more details.

- **Unsupported Streaming Write Operations:** Streaming writes only work for sequential writes to new or empty files. GCSFuse will automatically revert to legacy staged writes for the following scenarios:
    - **Modifying existing files (non-zero size):** Writing to a file that is not empty will cause that file to use legacy staged writes. You will see an informational log message similar to:
      > Existing file <var>file_name</var> of size <var>size</var> bytes (non-zero) will use legacy staged writes.

    - **Performing out-of-order writes:** Streaming writes require data to be written sequentially. If a write occurs at an unexpected offset, GCSFuse will finalize the currently written sequential data and switch to legacy staged writes. You will see an informational log message similar to:
      > Out of order write detected. File <var>file_name</var> will now use legacy staged writes.

    - **Reading from a file while writes are in progress:** Performing a read on a file that is being actively written to using streaming writes will finalizes the object on GCS. Subsequent writes to that file will use the legacy staged writes.

    - **Truncating a file downwards while streaming writes are in progress:** If a file is truncated to a smaller size while being written via streaming writes, the object is finalized on GCS, and subsequent writes will use the legacy staged writes. You will see an informational log message similar to:
      > Out of order write detected. File <var>file_name</var> will now use legacy staged writes.

### Issues related to the gcs-fuse-csi-driver
The `gcs-fuse-csi-driver` serves as the orchestration layer that manages the gke-gcsfuse-sidecar which hosts GCSFuse in Google Kubernetes Engine (GKE) environments. This driver is maintained in a separate repository. Consequently, any issues regarding the gcs-fuse-csi-driver should be reported in its dedicated GitHub repository: https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver

### Errors for unsupported file system operations
This is an expected error for file operations unsupported in GCSFUSE file system. Currently, GCSFuse does not support
the following operations:
- **Fallocate:** Used for pre-allocating disk space for a file so that disk space does not run out before writing. This
  is usually not implemented for Cloud based FUSE products as the disk space running out is not a concern there.
- **SetXattr, ListXattr, GetXattr, RemoveXattr:** GCSFuse doesn't support extended-attributes (x-attrs) operations.
  Extended attributes provide a way to associate additional metadata or information with files and directories beyond
  the standard attributes like file size, modification time, etc. This is not usually implemented for Cloud based Fuse
  products.
- **CreateLink:** Creates a hard link (a directory entry that associates a name with a file). GCSFuse doesn't support
  hardlinks.
- **BatchForget:**  This is a performance optimization for batch-forgetting inodes. When this is unimplemented,
  filesystem instead utilizes individual ForgetInode calls.

### Installation error: The repository does not have a Release file
The full error log would be something like: `Error: The repository 'https://packages.cloud.google.com/apt gcsfuse-<abc> Release' does not have a Release file.`

This occurs when the gcsfuse package corresponding to OS version returned by `lsb_release` (say x) is not in the [list of supported OS versions](https://cloud.google.com/storage/docs/cloud-storage-fuse/overview#frameworks-os-architectures) .

**Workaround**: Install GCSFuse for the closest supported OS version (say y), by running `export GCSFUSE_REPO="gcsfuse-y"` and retrying installation. An example of this is in https://github.com/GoogleCloudPlatform/gcsfuse/issues/3779, with x=`trixie` (for debian-13), and y=`bookworm` (for debian-12).  

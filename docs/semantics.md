# Read/Writes

## Reads

### Default Reads

Cloud Storage FUSE makes API calls to Cloud Storage to read an object directly, without downloading it to a local directory. A TCP connection is established, in which the entire object, or just portions as specified by the application/operating system via an offset, can be read back.

Files that have not been modified are read portion by portion on demand. Cloud Storage FUSE uses a heuristic to detect when a file is being read sequentially, and will issue fewer, larger read requests to Cloud Storage in this case, increasing performance.

### Buffered Reads

Buffered Read feature accelerates large sequential reads by asynchronously prefetching data into in-memory buffers and serving subsequent reads from in-memory buffer instead of making network calls.

The feature is **disabled by default** and can be enabled using:
- Command-line flag: `--enable-buffered-read`
- Config file: `read:enable-buffered-read: true`

**Note:** Buffered reads are designed to operate exclusively when the file cache is disabled. If both features are enabled, the file cache takes precedence and buffered reads will be ignored.

**Best Use Cases:**
- Single-threaded applications reading large files (> 100MB) sequentially.

**Performance Gains:**
- Can provide 2-5x improvement in sequential read throughput.
- Most effective for files larger than 100 MB.

**Memory Usage:**
- **Per file handle:** Up to 320 MB (20 × 16MB memory blocks) while reading.
- **Global limit:** Controlled by `--read-global-max-blocks` flag or `read:global-max-blocks` config (default: 40 blocks). By default 640 MB (40 × 16MB) across all the file handles.

**Important:** Please Consider available system memory when enabling buffered reads or adjusting `--read-global-max-blocks` to prevent out-of-memory (OOM) issues.

**CPU Usage:** The CPU overhead is typically proportional to the performance gains achieved.

**Known Limitations:** Workloads combining sequential and random reads (e.g., some model serving scenarios) may not benefit and could automatically fall back to default reads. Future releases will include improved heuristics for these scenarios.


## Writes

Starting with v3.0, streaming writes is the default write path. For more details, see the `With Streaming Writes`
section below. You can revert to the previous default write path (staging writes to a temporary file on disk) using the
`--enable-streaming-writes=false` flag or `write:enable-streaming-writes: false` in the config file.

### Streaming Writes - Default Write Path

Starting with version 2.9.1, and becoming the default in v3.0.0, GCSFuse supports streaming-writes, which is a new write
path that uploads data directly to Google Cloud Storage (GCS) as it's written without fully staging the file in the
temp-dir. This reduces both latency and disk space usage, making it particularly beneficial for large, sequential writes
such as checkpoints. Streaming writes can be enabled using `--enable-streaming-writes` flag or
`write:enable-streaming-writes:true` in the config file (Default starting GCSFuse v3.0.0).

**Memory Usage:** Each file opened for streaming writes will consume
approximately 96MiB of RAM during the upload process. This memory is released
when the file handle is closed. This should be considered when planning resource
allocation for applications using streaming writes.

Memory usage can be controlled using the `--write-global-max-blocks` flag or `write:global-max-blocks` config. The
default value is 4 for low-spec machines and 1600 for high-spec machines. One block is used per file, which means that
on low-spec machines, writes will automatically fall back to legacy staged writes if more than 4 files are concurrently
opened for streaming writes.

#### Note on Streaming Writes:

- **New files, Sequential Writes:** Streaming writes are designed for sequential
  writes to a new file only. Modifying existing files, or doing out-of-order
  writes (whether from the same file handle or concurrent writes from multiple
  file handles) will cause GCSFuse to automatically revert to the existing write
  path of staging writes to a temporary file on disk. An informational log
  message will be emitted when this fallback occurs.

- **Concurrent Writes to the Same File:** While concurrent writes to the same
  file are possible, they are not the primary use case for this initial phase of
  streaming writes. If a (rare, often server-related) error occurs during
  concurrent writes, all file handles must be closed before any future writes
  can resume. This phase of streaming writes is optimized for single-stream
  writes to new files, such as for AI/ML checkpointing.

- **File System Semantics Change:**
    - **FSync operation does not finalize the object:** When streaming writes
      are enabled, the fsync operation will not finalize the object on GCS.
      Instead, the object will be finalized only when the file is closed.
      Only finalized objects are visible to the end user. This is a key
      difference from the default non-streaming-writes behavior and should be considered when
      using streaming writes. Relying on fsync for data durability with
      streaming writes enabled is not recommended. Data is guaranteed to be
      on GCS only after the file is closed.
    - **Rename Operation Syncs the File:** Rename operation on a file undergoing
      writes via streaming writes will be finalized and then renamed. This means
      that any follow up writes will automatically revert to the existing
      behavior of staging writes to a temporary file on disk.
    - **Read Operations During Write:** Reads are now supported on files that are being
      written to with streaming writes. However, performing a read operation will finalize the object on GCS. Any
      subsequent write operations to that file will then automatically revert to legacy staged writes. Applications should
      generally avoid reading from a file while it is being written to using streaming writes, as this will prematurely
      finalize the object.
    - **Truncate During Writes:** If a file is truncated downwards using truncate() or ftruncate() while streaming
      writes
      are in progress, the file on GCS is finalized, and any subsequent writes revert to legacy staged writes.

### Staged Writes - Legacy Write Path

Files are written locally as a temporary file (temp-file for short) whose
location is controlled by the flag `--temp-dir`. Upon closing or fsyncing
the file, the file is then written to your Cloud Storage bucket and the
temp-file is deleted.

For modifications to existing files, Cloud Storage FUSE downloads the
entire
backing object's contents from Cloud Storage, storing them in the same temporary
directory as mentioned above. When the file is closed or fsync'd, Cloud Storage
FUSE writes the contents of the local file back to Cloud Storage as a new object
generation, and deletes
temp-file. Modifying even a single bit of an object results in the full
re-upload of the object. The
exception is if an append is done to the end of a file, where the original file
is at least 2MB, then only the appended content is uploaded.

As new and modified files are fully staged in the local temporary directory
until they are written out to Cloud Storage, you
must ensure that there is enough free space available to handle staged content
when writing large files.

#### Notes

- Prior to version 1.2.0, you will notice that an empty file is created in the
  Cloud Storage bucket as a hold. Upon closing or fsyncing the file, the file
  is written to your Cloud Storage bucket, with the existing empty file now
  reflecting the accurate file size and content. Starting with version 1.2,
  the default behavior is to not create this zero-byte file, which increases
  write performance. If needed, it can be re-enabled by setting the
  `create-empty-file: true` configuration in the config file.
- If the application never sends fsync for a file, it will leave behind its
  temp-file (in temp-dir), which will not be cleared until the user unmounts
  the bucket. As an example, if you are writing a large file, and temp-dir
  does not have enough free space available, then you will get 'out of space'
  error. Then the temp-file will not be deleted until you do an fsync for that
  file, or unmount the bucket.

___

# Concurrency

Multiple readers can access the same or different objects within a bucket without issue. Likewise, multiple writers can modify different objects in the same bucket simultaneously without any issue. Concurrent writes to the same gcs object are supported from the same mount and behave similar to native file system.

However, when different mounts try to write to the same object, the flush from first mount wins. Other mounts that have not updated their local file descriptors after the object is modified will encounter a ```syscall.ESTALE``` error when attempting to save their edits due to precondition checks. Therefore, to ensure data is consistently written, it is strongly recommended that multiple sources do not modify the same object.

### Write/Read consistency

Cloud Storage by nature is [strongly consistent](https://cloud.google.com/storage/docs/consistency). Cloud Storage FUSE offers close-to-open and fsync-to-open consistency. Once a file is closed, consistency is guaranteed in the following open and read immediately.

Close and fsync create a new generation of the object before returning, as long as the object hasn't been changed since it was last observed by the Cloud Storage FUSE process. On the other end, open guarantees to observe a generation at least as recent as all generations created before open was called.
Examples:

- Machine A opens a file and writes then successfully closes or syncs it, and the file was not concurrently unlinked from the point of view of A. Machine B then opens the file after machine A finishes closing or syncing. Machine B will observe a version of the file at least as new as the one created by machine A.
- Machine A and B both open the same file, which contains the text ‘ABC’. Machine A modifies the file to ‘ABC-123’ and closes/syncs the file which gets written back to Cloud Storage. Afterward, Machine B, which still has the file open, modifies its local copy to ‘ABC-XYZ’, then tries to save and close the file. Since the first writer wins, the final state of the file in the cloud storage will be 'ABC-123'. Consequently, Machine B's file descriptor will receive an ESTALE error.

### Stale File Handle Errors

To ensure consistency, Cloud Storage FUSE returns a ```syscall.ESTALE``` error when an application tries to access stale data. This can occur in the following circumstances:

- **Concurrent Writes**: When multiple mounts have the same file open for writing, and one mount modifies and syncs the file, other mounts with open file descriptors will encounter this error when attempting to sync or close the file.
- **Read During Modification**:  When an application is reading a file through a GCSFuse mount, and the same object is modified on GCS (by deleting, renaming, or changing its content or metadata), the GCSFuse reader will encounter this error. This is because GCSFuse detects that the file it was accessing has changed.
- **File Renaming During Write**: When an application is writing to a file through a GCSFuse mount, and the same object is renamed on Google Cloud Storage (via same or different GCSFuse mount or through another interface), the writer will encounter this error when syncing or closing the file.
- **File Deletion During Write**: When an application is writing to a file through a GCSFuse mount, and the same object is deleted on Google Cloud Storage (via different GCSFuse mount or through another interface), the writer will encounter this error when syncing or closing the file.

These changes in Cloud Storage FUSE prioritize data integrity and provide users with clear indications of potential conflicts, preventing silent data loss and ensuring a more robust and reliable experience.

___

# Caching

Cloud Storage FUSE has four forms of optional caching: stat, type, list, and file. Stat and type caches are enabled by default. Using Cloud Storage FUSE with file caching, list caching, stat caching, or type caching enabled can significantly increase performance but reduces consistency guarantees.
The different forms of caching are discussed in this section, along with their trade-offs and the situations in which they are and are not safe to use.

The default behavior is appropriate, and brings significant performance benefits, when the bucket is never modified or is modified only via a single Cloud Storage FUSE mount. If you are using Cloud Storage FUSE in a situation where multiple actors will be modifying a bucket, be sure to read the rest of this section carefully and consider disabling caching.

**Important**: The rest of this document assumes that caching is disabled (by setting ```--stat-cache-ttl 0``` and ```--type-cache-ttl 0``` or ```metadata-cache:ttl-secs: 0```). This is not the default. If you want the consistency guarantees discussed in this document, you must use these options to disable caching. 

## Stat caching

The cost of the consistency guarantees discussed in the rest of this document is that Cloud Storage FUSE must frequently send stat object requests to Cloud Storage in order to get the freshest possible answer for the kernel when it asks about a particular name or inode, which happens frequently. This can make what appear to the user to be simple operations, like ```ls -l```, take quite a long time.

To alleviate this slowness, Cloud Storage FUSE supports using cached data where it would otherwise send a stat object request to Cloud Storage, saving some round trips. Caching these can help with file system performance, since otherwise the kernel must send a request for inode attributes to Cloud Storage FUSE for each call to ```write(2)```, ```stat(2)```, and others.

The behavior of stat cache is controlled by the following flags/config parameters:

1. **Stat-cache size**: It controls the maximum memory-size of the stat-cache entries. It can be configured in the following ways.
   - `metadata-cache:stat-cache-max-size-mb`: This is an integer parameter set through the config-file. It sets the stat-cache size in MBs. This can be set to -1 for infinite stat-cache size, 0 for disabling stat-cache, and > 0 for setting a finite stat-cache size. Values below -1 will return error on mounting.
   If this is missing, then `--stat-cache-capacity` is used.
   - `--stat-cache-capacity`: This is an integer commandline flag. It sets the stat-cache size in count.
   This has been deprecated (starting v2.0) and is ignored if the user sets `metadata-cache:stat-cache-max-size-mb` .
   This can be set to 0 for disabling stat-cache and > 0 for setting a finite stat-cache size.

   If neither of these two is set, then a size of 32MB is used, which is
   equivalent to about 20460 stat-cache entries (assuming just as many negative
   stat-cache entries).

   If you have more objects (folders or files) than that in your bucket that you
   want to access, then you may want to increase this, otherwise the caching
   will not function properly when listing that folder's contents:
    - ListObjects will return information on the items within the folder. Each item's data is cached
    - Because there are more objects than cache capacity, the earliest entries will be evicted
    - The linux kernel then asks for a little more information on each file.
    - As the earliest cache entries were evicted, this is a fresh GetObjectDetails request
    - This cycle repeats and sends a GetObjectDetails request for every item in the folder, as though caching were disabled

2. **Stat-cache TTL**: It controls the duration for which Cloud Storage FUSE allows the kernel to cache inode attributes. It can be set in one of the following two ways.
   * ```metadata-cache: ttl-secs``` in the config-file. This is set as an integer, which sets the TTL in seconds. If this is -1, TTL is taken as infinite i.e. no-TTL based expirations of entries. If this is 0, that disables the stat-cache.
   If this config variable is missing, then the value of ```--stat-cache-ttl``` is used.
   * ```--stat-cache-ttl``` commandline flag, which can be set to a value like ```10s``` or ```1.5h```. The default is one minute. This has been deprecated (starting v2.0) and is currently only available for backward compatibility. If ```metadata-cache: ttl-secs``` is set, ```--stat-cache-ttl``` is ignored.
   
   Positive and negative stat results will be cached for the specified amount of time.

Warnings: 
- Using stat caching breaks the consistency guarantees discussed in this document. It is safe only in the following situations:
  - The mounted bucket is never modified.
  - The mounted bucket is only modified on a single machine, via a single Cloud Storage FUSE mount.
  - The mounted bucket is modified by multiple actors, but the user is confident that they don't need the guarantees discussed in this document.
- On high performance machines GCSFuse sets TTL to infinite by default ([refer](https://cloud.google.com/storage/docs/cloud-storage-fuse/caching#cache-invalidation)). Please override it manually if your workload requires consistency guarantees.

## Type caching

Because Cloud Storage does not forbid an object named ```foo``` from existing next to an object named ```foo/``` (see the Name conflicts section), when Cloud Storage FUSE is asked to look up the name "foo" it must stat both objects.

Stat cache can help with this, but it does not help until after the first request. For example, assume that there is an object named foo but not one named ```foo/```, and the stat cache is enabled. When the user runs ```ls -l```, the following happens:
- The objects in the bucket are listed. This causes a stat cache entry for ```foo``` to be created.
- ```ls``` asks to stat the name ```foo```, causing a lookup request to be sent for that name.
- Cloud Storage FUSE sends Cloud Storage stat requests for the object named ```foo``` and the object named ```foo/```. The first will hit in the stat cache, but the second will have to go all the way to Cloud Storage to receive a negative result.

The negative result for ```foo/``` will be cached, but that only helps with the second invocation of ```ls -l```.

To alleviate this, Cloud Storage FUSE supports a "type cache" on directory inodes. When type cache is enabled, each directory inode will maintain a mapping from the name of its children to whether those children are known to be files or directories or both. When a child is looked up, if the parent's cache says that the child is a file but not a directory, only one Cloud Storage object will need to be stated. Similarly if the child is a directory but not a file.

The behavior of type cache is controlled by the following flags/config parameters:
1. **Type-cache size**: This is configurable at per-directory level by setting `metadata-cache: type-cache-max-size-mb` in config-file. This is the maximum size of type-cache per-directory in MiBs. By default, this is set at 4, which roughly equates to about 21k entries.
1. **Type-cache TTL**: It controls the duration for which Cloud Storage FUSE caches an inode's type attribute. It can be set in one of the following two ways.
* ```metadata-cache: ttl-secs``` in the config-file. This is set as an integer, which sets the TTL in seconds. If this is -1, TTL is taken as infinite i.e. no-TTL based expirations of entries. If this is 0, that disables the type-cache. If this is <-1, then an error is thrown on mount.
- ```--type-cache-ttl``` commandline flag, which can be set to a value like ```10s``` or ```1.5h```. The default is one minute. This has been deprecated (starting v2.0) and is currently only available for backward compatibility. If ```metadata-cache: ttl-secs``` is set, ```--type-cache-ttl``` is ignored.

**Warning**: Using type caching breaks the consistency guarantees discussed in this document. It is safe only in the following situations:
- The mounted bucket is never modified.
- The type (file or directory) for any given path never changes.

## File caching

The Cloud Storage FUSE file cache feature is a client-based read cache that lets repeat file reads to be served from a faster local cache storage media of your choice.

The behavior of file cache is controlled by the following config-file parameters:

1. **cache-dir**: Specifies the directory to use for the file cache. Passing a path to a directory enables the file cache feature.

2. **file-cache: max-file-size-mb**: is the maximum size in MiB that the file cache can use. This is useful if you want to limit the total capacity the Cloud Storage FUSE cache can use within its mounted directory.
   - Use the default value of -1 to use the cache's entire available capacity in the directory you specify for cache-dir.
   - Use a value of 0 to disable the file cache.
   - The eviction of cached metadata and data is based on a least recently used (LRU) algorithm that begins once the space threshold configured per max-size-mb limit is reached.     

3. **file-cache: cache-file-for-range-read**: is a boolean that determines whether the full object should be downloaded asynchronously and stored in the Cloud Storage FUSE cache directory when the first read is done from a non-zero offset. This should be set to 'true' if you plan on performing several random reads or partial reads. The default value is 'false'
   - If doing a partial read starting at offset 0, Cloud Storage FUSE always asynchronously downloads and caches the full object.

4. **metadata-cache: ttl-secs**: As mentioned above, defines the time to live (TTL), in seconds, of metadata entries used for the stat, type, and the file cache.  Apart from specifying a value that represents the number of seconds, the ttl-secs flag also supports the values of 0 and -1: 
   - Use a value of -1 to bypass a TTL expiration and serve the file from the cache whenever it's available. Serving files without checking for consistency can serve inconsistent data, and should only be used temporarily for workloads that run in jobs with non-changing data. For example, using a value of -1 is useful for machine learning training, where the same data is read across multiple epochs without changes.
   - Use a value of 0 to ensure that the most up to date file is read. Using a value of 0 issues a Get metadata call to make sure that the object generation for the file in the cache matches what's stored in Cloud Storage. 

Additional file cache [behavior](https://cloud.google.com/storage/docs/gcsfuse-cache):
1. **Persistence**: Cloud Storage FUSE caches aren't persisted on unmounts and restarts. For file caching, while the metadata entries needed to serve files from the cache are evicted on unmounts and restarts, data in the file cache may still be present in the file directory. You should delete data in the file cache directory after unmounts or restarts.

2. **Security**: When you enable caching, Cloud Storage FUSE uses the specified 'cache-dir' you set as the underlying directory for the cache to persist files from your Cloud Storage bucket in an unencrypted format. Any user or process that has access to this cache directory can access these files. We recommend that you restrict access to this directory.

3. **Direct or multiple access to the file cache**: Using a process other than Cloud Storage FUSE to access or modify a file in the cache directory can lead to data corruption. Cloud Storage FUSE caches are specific to each Cloud Storage FUSE running process with no awareness across different Cloud Storage FUSE processes running on the same or different machines. Subsequently, the same cache directory shouldn't be used by different Cloud Storage FUSE processes.

4. **Eviction**: The eviction of cached metadata and data is based on a least recently used (LRU) algorithm that begins once the space threshold configured per max-size-mb limit is reached.

5. **Invalidation**: File cache data is invalidated per the set 'metadata-cache: ttl-secs' value:
   - If a file cache entry hasn't yet expired based on its TTL and the file is in the cache, the entire operation is served from the local client cache without any request being issued to Cloud Storage.

   - If a file cache entry has expired based on its TTL, a Get metadata call is first made to Cloud Storage, and if the file isn't in the cache, the file is retrieved from Cloud Storage. Both operations are subject to network latencies. If the metadata entry has been invalidated, but the file is in the cache, and its object generation has not changed, the file is served from the cache only after the Get metadata call is made to check if the data is valid.

   - If a Cloud Storage FUSE client modifies a cached file or its metadata, then the file is immediately invalidated and consistency is ensured in the following read by the same client. However, if different clients access the same file or its metadata, and its entries are cached, then the cached version of the file or metadata is read and not the updated version until the file is invalidated by that specific client's TTL setting.     

## Kernel List Cache

As the name suggests, the Cloud Storage FUSE kernel-list-cache is used to cache the directory listing (output of `ls`) in kernel page-cache. It significantly improves the workload which involves repeated listing. For multi node/mount-point scenario, this is recommended to be used only for read only workloads, e.g. for Serving and Training workloads.

By default, the list cache is disabled. It can be enabled by configuring the `--kernel-list-cache-ttl-secs` cli flag or `file-system:kernel-list-cache-ttl-secs` config flag where:
*   A value of 0 means disabled. This is the default value.
*   A positive value represents the ttl (in seconds) to keep the directory list response in the kernel page-cache.
*   -1 to bypass entry expiration and always return the list response from the cache if available.

**Important Points**
*   The kernel-list-cache is kept within the kernel's page-cache. Consequently, this functionality depends upon the availability of page-cache memory on the system. This contrasts with the stat and type caches, which are retained in user memory as part of Cloud Storage Fuse daemon.
*   The kernel's list cache is maintained on a per-directory level, resulting in either all list entries being retained in the page cache or none at all.
*   The creation, renaming, or deletion of new files or folders causes the eviction of the page-cache of their immediate parent directory, but not of all ancestral directories.
*   The ttl-based eviction doesn't work with kernel versions 6.9.x to 6.12.x ([details](https://github.com/GoogleCloudPlatform/gcsfuse/issues/2792)), because of a [bug](https://lore.kernel.org/linux-fsdevel/CAEW=TRr7CYb4LtsvQPLj-zx5Y+EYBmGfM24SuzwyDoGVNoKm7w@mail.gmail.com/) in kernel-fuse driver, which is [fixed](https://github.com/torvalds/linux/commit/03f275adb8fbd7b4ebe96a1ad5044d8e602692dc) in 6.13.x. Although, eviction because of creation, renaming, or deletion of a file or folders from the same mount works as expected.

**Consistency**
*   Kernel List cache ensures consistency within the mount. That means, creation, deletion or rename of files/folder within a directory evicts the kernel list cache of the directory.
*   Externally added objects are only visible after the kernel-list-cache-ttl-secs ttl expires, even if they are touched (stat) via the same mount point. Since stat doesn’t evict the kernel-list-cache so there might be some list-stat inconsistency.
*   Kernel-list-cache-ttl doesn't work with empty directories. In case a new file is added to the empty directory remotely outside of the mount, the client will not be able to access the new file even if ttl is expired.
*   One of the known consistency issue: `rm -R` encounters consistency issues when objects are created externally in a bucket. Specifically, if a client (e.g., `Cloud Storage Fuse` client1) caches a directory listing and another client (client2) adds a new file to the directory before the cached listing expires, `rm -R` on the directory will fail with a "Directory not empty" error. This occurs because `rm -R` initially deletes the directory's children based on the cached listing and then checks the directory's emptiness by making a List call, which returns not empty due to the externally added file.

**Note**:

1. ```--stat-cache-ttl``` and ```--type-cache-ttl``` have been deprecated (starting v2.0) and only ```metadata-cache: ttl-secs``` in the gcsfuse config-file will be supported. So, it is recommended to switch from these two to ```metadata-cache: ttl-secs```.
For now, for backward compatibility, both are accepted, and the minimum of the two, rounded to the next higher multiple of a second, is used as TTL for both stat-cache and type-cache, when ```metadata-cache: ttl-secs``` is not set.
1. Both stat-cache and type-cache internally use the same TTL.

___

# Files and Directories

As Cloud Storage FUSE is a way to mount a bucket as a local filesystem, and directories are essential to filesystems, Cloud Storage FUSE presents directories logically using ```/``` prefixes. Cloud Storage object names map directly to file paths using the separator '/'. Object names ending in a slash represent a directory, and all other object names represent a file. Directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.


How Cloud Storage FUSE uses them depends on where the source structure was originally created - created by Cloud Storage FUSE in a new deployment, or mounting a bucket that already has objects using a ```/``` prefix to represent a directory structure.

In the most basic example, let's say a user is creating the following new structure from their Cloud Storage FUSE mount:

- A/ -Type:Directory folder
    - 1.txt - Type: File
    - B/ -Type:(sub)Directory folder
        - 2.txt -Type:File
- C/ -Type:Directory folder
     - 3.txt -Type:File

The Cloud Storage bucket will then contain the following objects, where a trailing slash indicated a directory:
- A/
- A/1.txt
- A/B/
- A/B/2.txt
- C/
- C/3.txt

Even though A/, A/B/, and C/ are directories in the filesystem, a 0-byte object is created for each directory in Cloud Storage in order for Cloud Storage FUSE to recognize it as a directory. 

**Mounting a bucket with existing prefixes**

The above example was based on greenfield deployments which assumes starting fresh, where the directories are created from Cloud Storage FUSE. If a user unmounts this Cloud Storage FUSE bucket, and then re-mounts it to a different path, the user will see the directory structure correctly in the filesystem because it was originally created by Cloud Storage FUSE.

However, if a user already has objects with prefixes to simulate a directory structure in their buckets that did not originate from Cloud Storage FUSE, and mounts the bucket using Cloud Storage FUSE, the directories and objects under the directories will not be visible until a user manually creates the directory, with the same name, using mkdir on the local instance. This is because with Cloud Storage FUSE, directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.  These backing objects for directories are special 0-byte objects which are placeholders for directories. Note that these can also be created via the WebUI and Cloud Storage SDKs, but not by the Cloud Storage CLI tools such as gcloud.

If a user has the following objects in their Cloud Storage buckets, for example created by uploading a local directory using `gcloud storage cp --recursive` command.
- A/1.txt
- A/B/2.txt
- C/3.txt

then mounting the bucket and running ```ls``` to see its content will not show any files until placeholder directory objects A/, A/B/, and C/ are created in the GCS bucket. These placeholder directory objects can be created by running `mkdir ./A`, `mkdir ./A/B` and `mkdir ./C` on GCSFuse mounted directory.
At which point the user will correctly see:
- A/
- A/1.txt
- A/B/
- A/B/2.txt
- C/
- C/3.txt

**Using ```--implicit-dirs``` flag:**

Cloud Storage FUSE supports a flag called ```--implicit-dirs``` that changes the behavior for how pre-existing directory structures, not created by Cloud Storage FUSE, are mounted and visible to Cloud Storage FUSE. When this flag is enabled, name lookup requests from the kernel use the Cloud Storage API's Objects.list operation to search for objects that would implicitly define the existence of a directory with the name in question. 

The example above describes how from the local filesystem the user doesn't see any files, until the user creates A/, A/B/, C/ using mkdir.  If instead the ```--implicit-dirs``` flag is passed, you would see the intended directory structure without first having to create the directories A/, A/B/, C/. 

However, implicit directories does have drawbacks:
- The feature requires an additional request to Cloud Storage for each name lookup, which may have costs in terms of both charges for operations and latency.
- With the example above, it will appear as if there is a directory called "A/" containing a file called "1.txt". But when the user runs ‘rm A/1.txt’, it will appear as if the file system is completely empty. This is contrary to expectations, since the user hasn't run ```rmdir A/```.
- Cloud Storage FUSE sends a single Objects.list request to Cloud Storage, and treats the directory as being implicitly defined if the results are non-empty. In rare cases (notably when many objects have recently been deleted) Objects.list may return an arbitrary number of empty responses with continuation tokens, even for a non-empty name range. In order to bound the number of requests, Cloud Storage FUSE simply ignores this subtlety. Therefore in rare cases an implicitly defined directory will fail to appear.

Alternatively, users can create a script which lists the buckets and creates the appropriate objects for the directories so that the ```--implicit-dirs``` flag is not used.

**Using Cloud Storage Buckets with [Hierarchical Namespaces Enabled](https://cloud.google.com/storage/docs/hns-overview) :**

Cloud Storage Fuse also offers seamless support for buckets with hierarchical namespaces enabled. Mounting an HNS-enabled bucket using gcsfuse works exactly like mounting a standard bucket, with no additional configuration required.

HNS-enabled buckets offer several advantages over standard buckets when used with cloud storage fuse:

- HNS-enabled buckets eliminate the need for --implicit-dirs flag. HNS buckets inherently understand directories, so gcsfuse does not need to simulate directories using placeholder objects ( 0-byte objects ending with '/' ). Users will see consistent directory listings with or without the flag.
- In HNS buckets, renaming a folder and its child folders is an atomic operation, meaning all associated resources—including objects and managed folders—are renamed in a single step. This ensures data consistency and significantly improves operation performance.
- HNS buckets treat folders as first-class entities, closely aligning with traditional file system semantics. Commands like mkdir now directly create folder resources within the bucket, unlike with traditional buckets where directories were simulated using prefixes and 0-byte objects.
- List object calls ([BucketHandle.Objects](https://cloud.google.com/storage/docs/json_api/v1/objects/list)), are replaced with [get folder](https://cloud.google.com/storage/docs/json_api/v1/folders/getfoldermetadata) calls, resulting in quicker response times and fewer overall list calls for every lookup operation.

___

# Generations

With each record in Cloud Storage is stored object and metadata [generation numbers](https://cloud.google.com/storage/docs/generations-preconditions). These provide a total order on requests to modify an object's contents and metadata, compatible with causality. So if insert operation A happens before insert operation B, then the generation number resulting from A will be less than that resulting from B.

In the discussion below, the term "generation" refers to both object generation and meta-generation numbers from Cloud Storage. In other words, what we call "generation" is a pair ```(G, M)``` of Cloud Storage object generation number ```G``` and associated meta-generation number ```M```.

___

# File inodes

As in any file system, file inodes in a Cloud Storage FUSE file system logically contain file contents and metadata. A file inode is initialized with a particular generation of a particular object within Cloud Storage (the "source generation"), and its contents are initially exactly the contents and metadata of that generation.

**Creation**

When a new file is created and ```open(2)``` was called with ```O_CREAT```, an empty object with the appropriate name is created in Cloud Storage. The resulting generation is used as the source generation for the inode, and it is as if that object had been pre-existing and was opened.

**Pubsub notifications on file creation**

[Pubsub notifications](https://cloud.google.com/storage/docs/reporting-changes) may be enabled on a Cloud Storage bucket to help track changes to Cloud Storage objects. Due to the semantics that Cloud Storage FUSE uses to create files, an OBJECT_FINALIZE event is generated per file created indicating that a non-zero sized object has been created.

These Cloud Storage events can be used from other cloud products, such as AppEngine, Cloud Functions, etc. It is recommended to ignore events for files with zero size.

**Modifications**

Inodes may be opened for writing. Modifications are reflected immediately in reads of the same inode by processes local to the machine using the same file system. After a successful ```fsync``` or a successful ```close```, the contents of the inode are guaranteed to have been written to the Cloud Storage object with the matching name if the object's generation and meta-generation numbers still match the source generation of the inode - they may not have if there had been modifications from another actor in the meantime. There are no guarantees about whether local modifications are reflected in Cloud Storage after writing but before syncing or closing.

Modification time (```stat::st_mtim)``` on Linux) is tracked for file inodes, and can be updated in the usual way using ```utimes(2)``` or ```futimens(2)```. When dirty inodes are written out to Cloud Storage objects, mtime is stored in the custom metadata key gcsfuse_mtime in an unspecified format.

There is one special case worth mentioning: mtime updates to unlinked inodes may be silently lost (of course content updates to these inodes will also be lost once the file is closed).

There are no guarantees about other inode times (such as ```stat::st_ctim``` and ```stat::st_atim``` on Linux) except that they will be set to something reasonable.

**Identity**

If a new generation is assigned to a Cloud Storage object due to a flush of a file inode, the source generation of the inode is updated and the inode ID remains stable. Otherwise, if a new generation is created by another machine or in some other manner from the local machine, the new generation is treated as an inode distinct from any other inode already created for the object name.

In other words: inode IDs don't change when the file system causes an update to Cloud Storage, but any update caused remotely will result in a new inode.

Inode IDs are local to a single Cloud Storage FUSE process, and there are no guarantees about their stability across machines or invocations on a single machine.

**Lookups**

One of the fundamental operations in the VFS layer of the kernel is looking up the inode for a particular name within a directory. Cloud Storage FUSE responds to such lookups as follows:

- Stat the object with the given name within the Cloud Storage bucket.
- If the object does not exist, return an error.
- Call the generation of the object ```(G, M)```. If there is already an inode for this name with source generation ```(G, M)```, return it.
- Create a new inode for this name with source generation ```(G, M)```.

**User-visible semantics**

The intent of these conventions is to make it appear as though local writes to a file are in-place modifications as with a traditional file system, whereas remote overwrites of a Cloud Storage object appear as some other process unlinking the file from its directory and then linking a distinct file using the same name. The ```st_nlink``` field will reflect this when using ```fstat(2)```.

Note the following consequence: if machine A opens a file and writes to it, then machine B deletes or replaces its backing object, or updates it’s metadata, then machine A closes the file, machine A's writes will be lost. This matches the behavior on a single machine when process A opens a file and then process B unlinks it. Process A continues to have a consistent view of the file's contents until it closes the file handle, at which point the contents are lost.

**Cloud Storage object metadata**

Cloud Storage FUSE sets the following pieces of Cloud Storage object metadata for file objects:
- contentType is set to Cloud Storage's best guess as to the MIME type of the file, based on its file extension.
- The custom metadata key gcsfuse_mtime is set to track mtime, as discussed above.

___

# Directory Inodes

Cloud Storage FUSE directory inodes exist simply to satisfy the kernel and export a way to look up child inodes. Unlike file inodes:
- There are no guarantees about stability of directory inode IDs. They may change from lookup to lookup even if nothing has changed in the Cloud Storage bucket. They may not change even if the directory object in the bucket has been overwritten.
- Cloud Storage FUSE does not keep track of modification time for directories. There are no guarantees for the contents of ```stat::st_mtim``` or equivalent, or the behavior of ```utimes(2)``` and similar.
- There are no guarantees about ```stat::st_nlink```.

Despite no guarantees about the actual times for directories, their time fields in stat structs will be set to something reasonable.

**Unlinking**

There is no way to delete an empty directory in Cloud Storage atomically. The only way to do it is by making two calls - first to list the objects in the directory object and then delete the directory object if it is empty.

With the perspective of Cloud Storage FUSE if a directory object is deleted without checking the emptiness condition, the child object becomes inaccessible and leads to non-standard file systems behavior.



Cloud Storage FUSE makes similar calls while deleting a directory: it lists objects with the directory's name as a prefix, returning ```ENOTEMPTY``` if anything shows up, and otherwise deletes the backing object.

Note that by definition, implicit directories cannot be empty.

___

# Symlink inodes

Cloud Storage FUSE represents symlinks with empty Cloud Storage objects that contain the custom metadata key ```gcsfuse_symlink_target```, with the value giving the target of a symlink. In other respects they work like a file inode, including receiving the same permissions.


**Note**

While GCSFuse supports symlinks that point to paths external to the mount point, it should be avoided as it could lead to broken links and security issues.


___

# Permissions and ownership

**Inodes**

By default, all inodes in a Cloud Storage FUSE file system show up as being owned by the UID and GID of the Cloud Storage FUSE process itself, i.e. the user who mounted the file system. All files have permission bits ```0644```, and all directories have permission bits ```0755``` (but see below for issues with use by other users). Changing inode mode (using chmod(2) or similar) is unsupported, and changes are silently ignored.

These defaults can be overridden with the ```--uid```, ```--gid```, ```--file-mode```, and ```--dir-mode``` flags.

**Fuse**

The fuse kernel layer itself restricts file system access to the mounting user ([fuse.txt](https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt##L102-L105)). No matter what the configured inode permissions are, by default other users will receive "permission denied" errors when attempting to access the file system. This includes the root user.

This can be overridden by setting ```-o allow_other``` to allow other users to access the file system. However, there may be [security implications](https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt#L218-L310).

___

# Non-standard filesystem behaviors

See [Key Differences from a POSIX filesystem](https://cloud.google.com/storage/docs/gcs-fuse#expandable-1)

## Unlinking directories

Because Cloud Storage offers no way to delete an object conditional on the non-existence of other objects, there is no way for Cloud Storage FUSE to unlink a directory if and only if it is empty. So Cloud Storage FUSE first performs a list call to Cloud Storage to check if the directory is empty or not. And then deletes the directory object only if the list call response is empty.  

Since, the complete unlink directory operation is not atomic, this may lead to inconsistency. E.g. as soon as Cloud Storage fuse gets the response of list call as `empty`, meanwhile some other machines may create a file in the same directory. As a result, the content of a non-empty directory that is unlinked are not deleted but simply become inaccessible (Unless --implicit-dirs is set; see the section on implicit directories above.)

**Reading directories**

Cloud Storage FUSE implements requests from the kernel to read the contents of a directory (as when listing a directory with ls, for example) by calling [Objects.list](https://cloud.google.com/storage/docs/json_api/v1/objects/list) in the Cloud Storage API. The call uses a delimiter of ```/``` to avoid paying the bandwidth and request cost of also listing very large sub-directories.

However, with this implementation there is no way for Cloud Storage FUSE to distinguish a child directory that actually exists (because its placeholder object is present) and one that is only implicitly defined. So when ```--implicit-dirs``` is not set, directory listings may contain names that are inaccessible in a later call from the kernel to Cloud Storage FUSE to look up the inode by name. For example, a call to ```readdir(3) ```may return names for which ```fstat(2)``` returns ```ENOENT```.

## Name conflicts

It is possible to have a Cloud Storage bucket containing an object named foo and another object named ```foo/```:
- This situation can easily happen when writing to Cloud Storage directly, since there is nothing special about those names as far as Cloud Storage is concerned.
- This situation may happen if two different machines have the same bucket mounted with Cloud Storage FUSE, and at about the same time one creates a file named ```foo``` and the other creates a directory with the same name. This is because the creation of the object ```foo/``` is not preconditioned on the absence of the object named foo, and vice versa.

Traditional file systems do not allow multiple directory entries with the same name, so all tools and kernel code are structured around this assumption. Therefore, it is not possible for Cloud Storage FUSE to faithfully preserve both the file and the directory in this case.

Instead, when a conflicting pair of foo and ```foo/``` objects both exist, it appears in the Cloud Storage FUSE file system as if there is a directory named foo and a file or symlink named ```foo\n``` (i.e. foo followed by U+000A, line feed). This is what will appear when the parent's directory entries are read, and Cloud Storage FUSE will respond to requests to look up the inode named ```foo\n``` by returning the file inode. ```\n``` in particular is chosen because it is not legal in Cloud Storage object names, and therefore is not ambiguous.

### Unsupported object names

- Objects in GCS with `double slashes '//'` as a name or prefix are not supported in GCSfuse. Accessing a directory with such named files will cause an 'input/output error', as the Linux filesystem does not support files or directories named with a '/'. The most common example of this is an object called, for example 'A//C.txt' where 'A' indicates a directory and 'C.txt' indicates a file, and is missing directory 'B/' between 'A/' and 'C.txt'.


- Objects in GCS with suffix `/\n` like, `gs://gcs-bkt/a/\n`:
Mounting bucket with such objects leads to crash `sync: unlock of unlocked mutex` or `Panic: Inode 'a/' cannot have child file`.
`\n` in GCSFuse is used to resolve the name conflicts, in case there is a file and directory exists with the same name. Ref: [name-conflict](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#name-conflicts) section.


## Memory-mapped files

Cloud Storage FUSE files can be memory-mapped for reading and writing using ```mmap(2)```. If you make modifications to such a file and want to ensure that they are durable, you must do the following:

- Keep the file descriptor you supplied to ```mmap(2)``` open while you make your modifications.
- When you are finished modifying the mapping, call ```msync(2)``` and check for errors.
- Call ```munmap(2)``` and check for errors.
- Call ```close(2)``` on the original file descriptor and check for errors.
- If none of the calls returns an error, the modifications have been made durable in Cloud Storage, according to the usual rules documented above.

See the notes on [fuseops.FlushFileOp](http://godoc.org/github.com/jacobsa/fuse/fuseops#FlushFileOp) for more details.

## Error Handling

Transient errors can occur in distributed systems like Cloud Storage, such as network timeouts. Cloud Storage FUSE implements Cloud Storage [retry best practices](https://cloud.google.com/storage/docs/retry-strategy) with exponential backoff. 


## Missing features

Not all of the usual file system features are supported. Most prominently:
- Renaming directories is only supported in Hierarchical Namespace Buckets, where they are fast and atomic. Renaming directories in flat namespace buckets is by default not supported. A directory rename cannot be performed atomically in these flat buckets and would therefore be arbitrarily expensive in terms of Cloud Storage operations, and for large directories would have high probability of failure, leaving the two directories in an inconsistent state.
- However, if your application is using Flat buckets and can tolerate the risks, you may enable renaming directories in a non-atomic way, by setting ```--rename-dir-limit```. If a directory contains fewer files than this limit and no subdirectory, it can be renamed.
- File and directory permissions and ownership cannot be changed. See the permissions section above.
- Modification times are not tracked for any inodes except for files.
- No other times besides modification time are tracked. For example, ctime and atime are not tracked (but will be set to something reasonable). Requests to change them will appear to succeed, but the results are unspecified.

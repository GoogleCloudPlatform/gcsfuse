# Read/Writes

**Reads**

Cloud Storage FUSE makes API calls to Cloud Storage to read an object directly, without downloading it to a local directory. A TCP connection is established, in which the entire object, or just portions as specified by the application/operating system via an offset, can be read back.

Files that have not been modified are read portion by portion on demand. Cloud Storage FUSE uses a heuristic to detect when a file is being read sequentially, and will issue fewer, larger read requests to Cloud Storage in this case, increasing performance. 

**Writes**

For modifications to existing objects, Cloud Storage FUSE downloads the entire backing object's contents from Cloud Storage. The contents are stored in a local temporary file whose location is controlled by the flag --temp-dir. Later, when the file is closed or fsync'd, Cloud Storage FUSE writes the contents of the local file back to Cloud Storage as a new object generation. Modifying even a single bit of an object results in the full re-upload of the object. The exception is if an append is done to the end of a file, where the original file is at leat 2MB,  then only the appended content is uploaded.

For new objects, objects are first written to the same temporary directory as mentioned above, and you will notice an empty file is created in the Cloud Storage bucket as a hold. Upon closing or fsyncing the file, the file is then written to your Cloud Storage bucket, with the existing empty file now reflecting the accurate file size and content.

As new and modified files are fully staged in the local temporary directory until they are written out to Cloud Storage from being closed or fsync'd, you must ensure that there is enough free space available to handle staged content when writing large files.

**Concurrency**

Multiple readers can access the same or different objects from the same bucket without issue. Multiple writers can also write to different objects in the same bucket without issue. However, there is no concurrency control for multiple writers to the same file. When multiple writers try to replace a file, the last write wins and all previous writes are lost - there is no merging, version control, or user notification of the subsequent overwrite. Therefore, for data integrity it is recommended that multiple sources do not modify the same object.

**Write/read consistency**

Cloud Storage by nature is [strongly consistent](https://cloud.google.com/storage/docs/consistency). Cloud Storage FUSE offers close-to-open and fsync-to-open consistency. Once a file is closed, consistency is guaranteed in the following open and read immediately.

Close and fsync create a new generation of the object before returning, as long as the object hasn't been changed since it was last observed by the Cloud Storage FUSE process. On the other end, open guarantees to observe a generation at least as recent as all generations created before open was called.
Examples:

- Machine A opens a file and writes then successfully closes or syncs it, and the file was not concurrently unlinked from the point of view of A. Machine B then opens the file after machine A finishes closing or syncing. Machine B will observe a version of the file at least as new as the one created by machine A.
- Machine A and B both open the same file, which contains the text ‘ABC’. Machine A modifies the file to ‘ABC-123’ and closes/syncs the file which gets written back to Cloud Storage. After, Machine B, which still has the file open, instead modifies the file to ‘ABC-ZYX’, and saves and closes the file. As the last writer wins, the current state of the file will read ‘ABC-XYZ’.

# Caching

By default, Cloud Storage FUSE has two forms of optional caching, which are enabled by default: stat and type. However, using Cloud Storage FUSE with either cache forms reduces consistency guarantees.
They are discussed in this section, along with their trade-offs and the situations in which they are and are not safe to use.

The default behavior is appropriate, and brings significant performance benefits, when the bucket is never modified or is modified only via a single Cloud Storage FUSE mount. If you are using Cloud Storage FUSE in a situation where multiple actors will be modifying a bucket, be sure to read the rest of this section carefully and consider disabling caching.

Important: The rest of this document assumes that caching is disabled (by setting ```--stat-cache-ttl 0``` and ```--type-cache-ttl 0```). This is not the default. If you want the consistency guarantees discussed in this document, you must use these options to disable caching.

**Stat caching**

The cost of the consistency guarantees discussed in the rest of this document is that Cloud Storage FUSE must frequently send stat object requests to Cloud Storage in order to get the freshest possible answer for the kernel when it asks about a particular name or inode, which happens frequently. This can make what appear to the user to be simple operations, like ls -l, take quite a long time.

To alleviate this slowness, Cloud Storage FUSE supports using cached data where it would otherwise send a stat object request to Cloud Storage, saving some round trips. This behavior is controlled by the --stat-cache-ttl flag, which can be set to a value like 10s or 1.5h. The default is one minute. Positive and negative stat results will be cached for the specified amount of time.

--stat-cache-ttl also controls the duration for which Cloud Storage FUSE allows the kernel to cache inode attributes. Caching these can help with file system performance, since otherwise the kernel must send a request for inode attributes to Cloud Storage FUSE for each call to write(2), stat(2), and others.

The size of the stat cache can also be configured with --stat-cache-capacity. By default the stat cache will hold up to 4096 items. If you have folders containing more than 4096 items (folders or files) you may want to increase this, otherwise the caching will not function properly when listing that folder's contents:
- ListObjects will return information on the items within the folder. Each item's data is cached
- Because there are more objects than cache capacity, the earliest entries will be evicted
- The linux kernel then asks for a little more information on each file.
- As the earliest cache entries were evicted, this is a fresh GetObjectDetails request
- This cycle repeats and sends a GetObjectDetails request for every item in the folder, as though caching were disabled

Warning: Using stat caching breaks the consistency guarantees discussed in this document. It is safe only in the following situations:
- The mounted bucket is never modified.
- The mounted bucket is only modified on a single machine, via a single Cloud Storage FUSE mount.
- The mounted bucket is modified by multiple actors, but the user is confident that they don't need the guarantees discussed in this document.

**Type caching**

Because Cloud Storage does not forbid an object named foo from existing next to an object named foo/ (see the Name conflicts section), when Cloud Storage FUSE is asked to look up the name "foo" it must stat both objects.

Stat cache enabled with --stat-cache-ttl can help with this, but it does not help until after the first request. For example, assume that there is an object named foo but not one named foo/, and the stat cache is enabled. When the user runs ls -l, the following happens:
- The objects in the bucket are listed. This causes a stat cache entry for foo to be created.
- ls asks to stat the name "foo", causing a lookup request to be sent for that name.
- Cloud Storage FUSE sends Cloud Storage stat requests for the object named foo and the object named foo/. The first will hit in the stat cache, but the second will have to go all the way to Cloud Storage to receive a negative result.

The negative result for foo/ will be cached, but that only helps with the second invocation of ls -l.

To alleviate this, Cloud Storage FUSE supports a "type cache" on directory inodes. When --type-cache-ttl is set, each directory inode will maintain a mapping from the name of its children to whether those children are known to be files or directories or both. When a child is looked up, if the parent's cache says that the child is a file but not a directory, only one Cloud Storage object will need to be stated. Similarly if the child is a directory but not a file.

Warning: Using type caching breaks the consistency guarantees discussed in this document. It is safe only in the following situations:
- The mounted bucket is never modified.
- The type (file or directory) for any given path never changes.

# Files and Directories

As Cloud Storage FUSE is a way to mount a bucket as a local filesystem, and directories are essential to filesystems, Cloud Storage FUSE presents directories logically using “/” prefixes. Cloud Storage object names map directly to file paths using the separator '/'. Object names ending in a slash represent a directory, and all other object names represent a file. Directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.


How Cloud Storage FUSE uses them depends on where the source structure was originally created - created by Cloud Storage FUSE in a new deployment, or mounting a bucket that already has objects using a “/” prefix to represent a directory structure.

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

However, If a user already has objects with prefixes to simulate a directory structure in their buckets that did not originate from Cloud Storage FUSE, and mounts the bucket using Cloud Storage FUSE, the directories and objects under the directories will not be visible until a user manually creates the directory using mkdir on the local instance. This is because with Cloud Storage FUSE, directories are by default not implicitly defined; they exist only if a matching object ending in a slash exists.


So if a user has the following objects in their Cloud Storage buckets, that were not originally created via Cloud Storage FUSE (for example created by uploading a local directory using gsutil cp -r command)

- A/
- A/1.txt
- A/B/
- A/B/2.txt
- C/
- C/3.txt

then mounting the bucket and running ‘ls’ to see its content will not show any files until the directories A/, A/B/, and C/ are created on the local filesystem using the ‘mkdir’ command.

This is the default behavior, unless a user passes the --implicit-dirs flag.

**Using --implicit-dirs flag:**

Cloud Storage FUSE supports a flag called --implicit-dirs that changes the behavior for how pre-existing directory structures, not created by Cloud Storage FUSE, are mounted and visible to Cloud Storage FUSE. When this flag is enabled, name lookup requests from the kernel use the Cloud Storage API's Objects.list operation to search for objects that would implicitly define the existence of a directory with the name in question. 

The example above describes how from the local filesystem the user sees only 0.txt, until the user creates A/, A/B/, C/ using mkdir.  If instead the --implicity-dirs flag is passed, you would see the intended directory structure without first having to create the directories A/, A/B/, C/. 

However, implicit directories does have drawbacks:
- The feature requires an additional request to Cloud Storage for each name lookup, which may have costs in terms of both charges for operations and latency.
- With the example above, it will appear as if there is a directory called "A/" containing a file called "1.txt". But when the user runs ‘rm A/1.txt’, it will appear as if the file system is completely empty. This is contrary to expectations, since the user hasn't run ‘rmdir A/’.
- Cloud Storage FUSE sends a single Objects.list request to Cloud Storage, and treats the directory as being implicitly defined if the results are non-empty. In rare cases (notably when many objects have recently been deleted) Objects.list may return an arbitrary number of empty responses with continuation tokens, even for a non-empty name range. In order to bound the number of requests, Cloud Storage FUSE simply ignores this subtlety. Therefore in rare cases an implicitly defined directory will fail to appear.

Alternatively, users can create a script which lists the buckets and creates the appropriate objects for the directories so that the --implicit-dirs flag is not used. 

# Generations

With each record in Cloud Storage is stored object and metadata [generation numbers](https://cloud.google.com/storage/docs/generations-preconditions). These provide a total order on requests to modify an object's contents and metadata, compatible with causality. So if insert operation A happens before insert operation B, then the generation number resulting from A will be less than that resulting from B.

In the discussion below, the term "generation" refers to both object generation and meta-generation numbers from Cloud Storage. In other words, what we call "generation" is a pair (G, M) of Cloud Storage object generation number G and associated meta-generation number M.

# File inodes

As in any file system, file inodes in a Cloud Storage FUSE file system logically contain file contents and metadata. A file inode is initialized with a particular generation of a particular object within Cloud Storage (the "source generation"), and its contents are initially exactly the contents and metadata of that generation.

**Creation**

When a new file is created and open(2) was called with O_CREAT, an empty object with the appropriate name is created in Cloud Storage. The resulting generation is used as the source generation for the inode, and it is as if that object had been pre-existing and was opened.

**Pubsub notifications on file creation**

[Pubsub notifications](https://cloud.google.com/storage/docs/reporting-changes) may be enabled on a Cloud Storage bucket to help track changes to Cloud Storage objects. Due of the semantics that Cloud Storage FUSE uses to create files, 3 different events are generated, per file created:

- One OBJECT_FINALIZE event: a zero sized object has been created.
- One OBJECT_DELETE event: the first generation of the object has been deleted.
- One OBJECT_FINALIZE event: a non-zero sized object has been created.

These Cloud Storage events can be used from other cloud products, such as AppEngine, Cloud Functions, etc. It is recommended to ignore events for files with zero size.

**Modifications**

Inodes may be opened for writing. Modifications are reflected immediately in reads of the same inode by processes local to the machine using the same file system. After a successful fsync or a successful close, the contents of the inode are guaranteed to have been written to the Cloud Storage object with the matching name if the object's generation and meta-generation numbers still match the source generation of the inode - they may not have if there had been modifications from another actor in the meantime. There are no guarantees about whether local modifications are reflected in Cloud Storage after writing but before syncing or closing.

Modification time (stat::st_mtim on Linux) is tracked for file inodes, and can be updated in the usual way using utimes(2) or futimens(2). When dirty inodes are written out to Cloud Storage objects, mtime is stored in the custom metadata key gcsfuse_mtime in an unspecified format.

There is one special case worth mentioning: mtime updates to unlinked inodes may be silently lost (of course content updates to these inodes will also be lost once the file is closed).

There are no guarantees about other inode times (such as stat::st_ctim and stat::st_atim on Linux) except that they will be set to something reasonable.

**Identity**

If a new generation is assigned to a Cloud Storage object due to a flush of a file inode, the source generation of the inode is updated and the inode ID remains stable. Otherwise, if a new generation is created by another machine or in some other manner from the local machine, the new generation is treated as an inode distinct from any other inode already created for the object name.

In other words: inode IDs don't change when the file system causes an update to Cloud Storage, but any update caused remotely will result in a new inode.

Inode IDs are local to a single Cloud Storage FUSE process, and there are no guarantees about their stability across machines or invocations on a single machine.

**Lookups**

One of the fundamental operations in the VFS layer of the kernel is looking up the inode for a particular name within a directory. Cloud Storage FUSE responds to such lookups as follows:

- Stat the object with the given name within the Cloud Storage bucket.
- If the object does not exist, return an error.
- Call the generation of the object (G, M). If there is already an inode for this name with source generation (G, M), return it.
- Create a new inode for this name with source generation (G, M).

**User-visible semantics**

The intent of these conventions is to make it appear as though local writes to a file are in-place modifications as with a traditional file system, whereas remote overwrites of a Cloud Storage object appear as some other process unlinking the file from its directory and then linking a distinct file using the same name. The st_nlink field will reflect this when using fstat(2).

Note the following consequence: if machine A opens a file and writes to it, then machine B deletes or replaces its backing object, or updates it’s metadata, then machine A closes the file, machine A's writes will be lost. This matches the behavior on a single machine when process A opens a file and then process B unlinks it. Process A continues to have a consistent view of the file's contents until it closes the file handle, at which point the contents are lost.

**Cloud Storage object metadata**

Cloud Storage FUSE sets the following pieces of Cloud Storage object metadata for file objects:
- contentType is set to Cloud Storage's best guess as to the MIME type of the file, based on its file extension.
- The custom metadata key gcsfuse_mtime is set to track mtime, as discussed above.







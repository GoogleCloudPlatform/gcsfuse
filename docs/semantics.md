This document defines the semantics of a gcsfuse file system mounted with a GCS
bucket, including how files and directories map to object names, what
consistency guarantees are made, etc.

# Compatibility

The following compatibility constraints are worth noting:

*   gcsfuse does not support GCS buckets with [object versioning][] enabled.
    (The default is to have this disabled.) No guarantees are made about its
    behavior when used with such a bucket.

*   Reading or modifying a file backed by an object with a `contentEncoding`
    property set may yield surprising results, and no guarantees are made about
    the behavior. It is recommended that you do not use gcsfuse to interact
    with such objects.

    (See the [Object resource][] and [performance tips][] pages for more info
    on `contentEncoding`, and [this][encoding-writeup] writeup for an
    explanation of why it can't be reliably supported.)

[object versioning]: https://cloud.google.com/storage/docs/object-versioning
[Object resource]: https://cloud.google.com/storage/docs/json_api/v1/objects#resource
[performance tips]: https://cloud.google.com/storage/docs/json_api/v1/how-tos/performance
[encoding-writeup]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/131#issuecomment-146031206

<a name="caching"></a>
# Caching

By default, gcsfuse has two forms of caching enabled that reduce consistency
guarantees. They are discussed in this section, along with their trade-offs and
the situations in which they are and are not safe to use.

The default behavior is appropriate, and brings significant performance
benefits, when the bucket is never modified or is modified only via a single
gcsfuse mount. If you are using gcsfuse in a situation where multiple actors
will be modifying a bucket, be sure to read the rest of this section carefully
and consider disabling caching.

**Important**: The rest of this document assumes that caching is disabled (by
setting `--stat-cache-ttl 0` and `--type-cache-ttl 0`). This is not the default.
If you want the consistency guarantees discussed in this document, you must use
these options to disable caching.

<a name="stat-caching"></a>
## Stat caching

The cost of the consistency guarantees discussed in the rest of this document
is that gcsfuse must frequently send stat object requests to GCS in order to
get the freshest possible answer for the kernel when it asks about a particular
name or inode, which happens frequently. This can make what appear to the user
to be simple operations, like `ls -l`, take quite a long time.

To alleviate this slowness, gcsfuse supports using cached data where it would
otherwise send a stat object request to GCS, saving some round trips. This
behavior is controlled by the `--stat-cache-ttl` flag, which can be set to a
value like `10s` or `1.5h`. (The default is one minute.) Positive and negative
stat results will be cached for the specified amount of time.

`--stat-cache-ttl` also controls the duration for which gcsfuse allows the
kernel to cache inode attributes. Caching these can help with file system
performance, since otherwise the kernel must send a request for inode attributes
to gcsfuse for each call to `write(2)`, `stat(2)`, and others.

**Warning**: Using stat caching breaks the consistency guarantees discussed in
this document. It is safe only in the following situations:

 *  The mounted bucket is never modified.
 *  The mounted bucket is only modified on a single machine, via a single
    gcsfuse mount.
 *  The mounted bucket is modified by multiple actors, but the user is
    confident that they don't need the guarantees discussed in this document.

<a name="type-caching"></a>
## Type caching

Because GCS does not forbid an object named `foo` from existing next to an
object named `foo/` (see the [Name conflicts](#name-conflicts) section),
when gcsfuse is asked to look up the name "foo" it must stat both objects.

The stat cache enabled with `--stat-cache-ttl` can help with this, but it does
not help fully until after the first request. For example, assume that there is
an object named `foo` but not one named `foo/`, and the stat cache is enabled.
When the user runs `ls -l`, the following happens:

 *  The objects in the bucket are listed. This causes a stat cache entry for
    `foo` to be created.

 *  `ls` asks to stat the name "foo", causing a lookup request to be sent for
    that name.

 *  gcsfuse sends GCS stat requests for the object named `foo` and the object
    named `foo/`. The first will hit in the stat cache, but the second will
    have to go all the way to GCS to receive a negative result.

The negative result for `foo/` will be cached, but that only helps with the
second invocation of `ls -l`.

To alleviate this, gcsfuse supports a "type cache" on directory inodes. When
`--type-cache-ttl` is set, each directory inode will maintain a mapping from
the name of its children to whether those children are known to be files or
directories or both. When a child is looked up, if the parent's cache says that
the child is a file but not a directory, only one GCS object will need to be
statted. Similarly if the child is a directory but not a file.

**Warning**: Using type caching breaks the consistency guarantees discussed in
this document. It is safe only in the following situations:

 *  The mounted bucket is never modified.
 *  The type (file or directory) for any given path never changes.


<a name="buckets"></a>
# Buckets

GCS has a feature called [object versioning][versioning] that allows buckets to
be put into a mode where the history of each object is maintained, even when it
is overwritten or deleted. gcsfuse does not support such buckets, and the file
system semantics discussed below do not apply to buckets in this mode—the
behavior for such buckets is undefined.

[versioning]: https://cloud.google.com/storage/docs/object-versioning


<a name="files-and-dirs"></a>
# Files and directories

GCS object names map directly to file paths using the separator '/'. Object
names ending in a slash represent a directory, and all other object names
represent a file. Directories are by default not implicitly defined; they exist
only if a matching object ending in a slash exists.

This is all much clearer with an example. Say that the GCS bucket contains the
following objects:

*   burrito/
*   enchilada/
*   enchilada/0
*   enchilada/1
*   queso/
*   queso/carne/
*   queso/carne/nachos
*   taco

Then the gcsfuse directory structure will be as follows, where a trailing slash
indicates a directory and the top level is the contents of the root directory
of the file system:

     burrito/
     enchilada/
         0
         1
     queso/
         carne/
             nachos
     taco

<a name="implicit-dirs"></a>
## Implicit directories

As mentioned above, by default there is no allowance for the implicit existence
of directories. Since the usual file system operations like `mkdir` will do the
right thing, if you set up a bucket's structure using only gcsfuse then you
will not notice anything odd about this. If, however, you use some other tool
to set up objects in GCS (such as the storage browser in the Google Developers
Console), you may notice that not all objects are visible until you create
leading directories for them.

For example, say that you use some other tool to set up a single object named
"foo/bar" in your bucket, then mount the bucket with gcsfuse. The file system
will initially appear empty, since there is no "foo/" object. However if you
subsequently run `mkdir foo`, you will now see a directory named "foo"
containing a file named "bar".

gcsfuse supports a flag called `--implicit-dirs` that changes the behavior.
When this flag is enabled, name lookup requests from the kernel use the GCS
API's Objects.list operation to search for objects that would implicitly define
the existence of a directory with the name in question. So, in the example
above, there would appear to be a directory named "foo".

The use of `--implicit-dirs` has some drawbacks (see [issue #7][issue-7] for a
more thorough discussion):

*   The feature requires an additional request to GCS for each name lookup,
    which may have costs in terms of request budget and latency.

*   With this setup, it will appear as if there is a directory called "foo"
    containing a file called "bar". But when the user runs `rm foo/bar`,
    suddenly it will appear as if the file system is completely empty. This is
    contrary to expectations, since the user hasn't run `rmdir foo`.

*   gcsfuse sends a single Objects.list request to GCS, and treats the
    directory as being implicitly defined if the results are non-empty. In rare
    cases (notably when many objects have recently been deleted) Objects.list
    may return an arbitrary number of empty responses with continuation tokens,
    even for a non-empty name range. In order to bound the number of requests,
    gcsfuse simply ignores this subtlety. Therefore in rare cases an implicitly
    defined directory will fail to appear.

Another option would be to periodically mkdir all missing directory entries. 
You could automate this with a webhook to reduce the delay between creating files 
and them apearing in gcsfuse. [issue-7-bash][Example bash script]

[issue-7-bash]: 
https://github.com/GoogleCloudPlatform/gcsfuse/issues/7#issuecomment-264221351
[issue-7]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/7


<a name="generations"></a>
# Generations

With each record in GCS is stored object and metadata
[generation numbers][generations]. These provide a total order on requests to
modify an object's contents and metadata, compatible with causality. So if
insert operation A happens before insert operation B, then the generation
number resulting from A will be less than that resulting from B.

In the discussion below, the term "generation" refers to both object generation
and meta-generation numbers from GCS. In other words, what we call "generation"
is a pair `(G, M)` of GCS object generation number `G` and associated
meta-generation number `M`.

[generations]: https://cloud.google.com/storage/docs/generations-preconditions


<a name="file-inodes"></a>
# File inodes

As in any file system, file inodes in a gcsfuse file system logically contain
file contents and metadata. A file inode is initialized with a particular
generation of a particular object within GCS (the "source generation"), and its
contents are initially exactly the contents and metadata of that generation.

### Creation

When a file is created anew because it doesn't already exist and `open(2)` was
called with `O_CREAT`, an empty object with the appropriate name is created in
GCS. The resulting generation is used as the source generation for the inode,
and it is as if that object had been pre-existing and was opened.

<a name="pubsub-creation"></a>
### Pubsub notifications on file creation.

[Pubsub notifications][gcs_notifications] may be enabled on a GCS bucket to help track changes to
Cloud Storage objects. Due of the semantics that GCSFuse uses to create files,
3 different events are generated, per file created:
1. One OBJECT_FINALIZE event: a zero sized object has been created.
2. One OBJECT_DELETE event: the first generation of the object has been deleted.
3. One OBJECT_FINALIZE event: a non-zero sized object has been created.

These GCS events can be used from other cloud products, such as AppEngine,
Cloud Functions, etc. It is recommended to ignore events for files with zero size.

[gcs_notifications]: https://cloud.google.com/storage/docs/reporting-changes

<a name="file-inode-modifications"></a>
### Modifications

Inodes may be opened for writing. Modifications are reflected immediately in
reads of the same inode by processes local to the machine using the same file
system. After a successful `fsync` or a successful `close`, the contents of the
inode are guaranteed to have been written to the GCS object with the matching
name if the object's generation and meta-generation numbers still matched the
source generation of the inode. (They may not have if there had been
modifications from another actor in the meantime.) There are no guarantees
about whether local modifications are reflected in GCS after writing but before
syncing or closing.

Modification time (`stat::st_mtim` on Linux) is tracked for file inodes, and can
be updated in usual the usual way using `utimes(2)` or `futimens(2)`. When dirty
inodes are written out to GCS objects, mtime is stored in the custom metadata
key `gcsfuse_mtime` in an unspecified format.

There is one special case worth mentioning: mtime updates to unlinked inodes
may be silently lost. (Of course content updates to these inodes will also be
lost once the file is closed.)

There are no guarantees about other inode times (such as `stat::st_ctim` and
`stat::st_atim` on Linux) except that they will be set to something reasonable.


<a name="file-inode-identity"></a>
### Identity

If a new generation is assigned to a GCS object due to a flush of a file
inode, the source generation of the inode is updated and the inode ID
remains stable. Otherwise, if a new generation is created by another machine or
in some other manner from the local machine, the new generation is treated as
an inode distinct from any other inode already created for the object name.

In other words: inode IDs don't change when the file system causes an update to
GCS, but any update caused remotely will result in a new inode.

Inode IDs are local to a single gcsfuse process, and there are no guarantees
about their stability across machines or invocations on a single machine.

<a name="file-inode-lookups"></a>
### Lookups

One of the fundamental operations in the VFS layer of the kernel is looking up
the inode for a particular name within a directory. gcsfuse responds to such
lookups as follows:

1.  Stat the object with the given name within the GCS bucket.
2.  If the object does not exist, return an error.
3.  Call the generation of the object `(G, M)`. If there is already an inode
    for this name with source generation `(G, M)`, return it.
4.  Create a new inode for this name with source generation `(G, M`).

<a name="file-inode-semantics"></a>
### User-visible semantics

The intent of these conventions is to make it appear as though local writes to
a file are in-place modifications as with a traditional file system, whereas
remote overwrites of a GCS object appear as some other process unlinking the
file from its directory and then linking a distinct file using the same name.
The `st_nlink` field will reflect this when using `fstat(2)`.

Note the following consequence: if machine A opens a file and writes to it,
then machine B deletes or replaces its backing object, or updates it metadata,
then machine A closes the file, machine A's writes will be lost. This matches
the behavior on a single machine when process A opens a file and then process B
unlinks it. Process A continues to have a consistent view of the file's
contents until it closes the file handle, at which point the contents are lost.


### GCS object metadata

gcsfuse sets the following pieces of GCS object metadata for file objects:

*   `contentType` is set to GCS's best guess as to the MIME type of the file,
    based on its file extension.

*   The custom metadata key `gcsfuse_mtime` is set to track mtime, as discussed
    above.


<a name="dir-inodes"></a>
# Directory inodes

gcsfuse directory inodes exist simply to satisfy the kernel and export a way to
look up child inodes. Unlike file inodes:

*   There are no guarantees about stability of directory inode IDs. They may
    change from lookup to lookup even if nothing has changed in the GCS bucket.
    They may not change even if the directory object in the bucket has been
    overwritten.

*   gcsfuse does not keep track of modification time for
    directories. There are no guarantees for the contents of `stat::st_mtim` or
    equivalent, or the behavior of `utimes(2)` and similar.

*   There are no guarantees about `stat::st_nlink`.

Despite no guarantees about the actual times for directories, their time fields
in `stat` structs will be set to something reasonable.

<a name="dir-inode-unlinking"></a>
### Unlinking

GCS offers no way to delete an object if and only if other objects don't exist.
It is therefore impossible to atomically check whether a directory is empty and
delete its backing object.

gcsfuse does the pragmatic thing here: it lists objects with the directory's
name as a prefix, returning `ENOTEMPTY` if anything shows up, and otherwise
deletes the backing object.

Note that by their definition, [implicit directories](#implicit-directories)
cannot be empty.


<a name="symlink-inodes"></a>
# Symlink inodes

gcsfuse represents symlinks with empty GCS objects that contain the custom
metadata key `gcsfuse_symlink_target`, with the value giving the target of a
symlink. In other respects they work like a file inode, including receiving the
same permissions.


<a name="write-read-consistency"></a>
# Write/read consistency

gcsfuse offers close-to-open and fsync-to-open consistency. As discussed above,
`close` and `fsync` create a new generation of the object before returning, as
long as the object hasn't been changed since it was last observed by the
gcsfuse process. On the other end, `open` guarantees to observe a generation at
least as recent as all generations created before `open` was called.

Therefore if:

*   machine A opens a file and writes then successfully closes or syncs it, and
*   the file was not concurrently unlinked from the point of view of A, and
*   machine B opens the file after machine A finishes closing or syncing,

then machine B will observe a version of the file at least as new as the one
created by machine A.


<a name="permissions"></a>
# Permissions and ownership

<a name="permissions-inodes"></a>
## Inodes

By default, all inodes in a gcsfuse file system show up as being owned by the
UID and GID of the gcsfuse process itself, i.e. the user who mounted the file
system. All files have permission bits `0644`, and all directories have
permission bits `0755` (but see below for issues with use by other users).
Changing inode mode (using `chmod(2)` or similar) is unsupported, and changes
are silently ignored.

These defaults can be overriden with the `--uid`, `--gid`, `--file-mode`, and
`--dir-mode` flags.

<a name="permissions-fuse"></a>
## Fuse

The fuse kernel layer itself restricts file system access to the mounting user
(cf. [fuse.txt][allow_other]). So no matter what the configured inode
permissions, by default other users will receive "permission denied" errors
when attempting to access the file system. This includes the root user.

[allow_other]: https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt##L102-L105

This can be overridden by setting `-o allow_other` to allow other users to
access the file system. Be careful! There may be [security
implications][fuse-security].

[fuse-security]: https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt#L218-L310


<a name="surprising-behaviors"></a>
# Surprising behaviors

<a name="unlinking-dirs"></a>
## Unlinking directories

Because GCS offers no way to delete an object conditional on the non-existence
of other objects, there is no way for gcsfuse to unlink a directory if and only
if it is empty. So gcsfuse takes the simple route, and always allows a
directory to be unlinked, even if non-empty. The contents of a non-empty
directory that is unlinked are not deleted but simply become inaccessible—the
placeholder object for the unlinked directory is simply removed. (Unless
`--implicit-dirs` is set; see the section on implicit directories above.)


<a name="reading-dirs"></a>
## Reading directories

gcsfuse implements requests from the kernel to read the contents of a directory
(as when listing a directory with `ls`, for example) by calling
[Objects.list][] in the GCS API. The call uses a delimiter of `/` to avoid
paying the bandwidth and request cost of also listing very large
sub-directories.

[Objects.list]: https://cloud.google.com/storage/docs/json_api/v1/objects/list

However, with this implementation there is no way for gcsfuse to distinguish a
child directory that actually exists (because its placeholder object is
present) and one that is only implicitly defined. So when `--implicit-dirs` is
not set, directory listings may contain names that are inaccessible in a later
call from the kernel to gcsfuse to look up the inode by name. For example, a
call to readdir(3) may return names for which fstat(2) returns `ENOENT`.


<a name="name-conflicts"></a>
## Name conflicts

It is possible to have a GCS bucket containing an object named `foo` and
another object named `foo/`:

*   This situation can easily happen when writing to GCS directly, since there
    is nothing special about those names as far as GCS is concerned.

*   This situation may happen if two different machines have the same bucket
    mounted with gcsfuse, and at about the same time one creates a file named
    "foo" and the other creates a directory with the same name. This is because
    the creation of the object `foo/` is not preconditioned on the absence of
    the object named `foo`, and vice versa.

Traditional file systems did not allow multiple directory entries with the same
name, so all tools and kernel code are structured around this assumption.
Therefore it's not possible for gcsfuse to faithfully preserve both the file
and the directory in this case.

Instead, when a conflicting pair of `foo` and `foo/` objects both exist, it
appears in the gcsfuse file system as if there is a directory named `foo` and a
file or symlink named `foo\n` (i.e. `foo` followed by U+000A, line feed). This
is what will appear when the parent's directory entries are read, and gcsfuse
will respond to requests to look up the inode named `foo\n` by returning the
file inode. `\n` in particular is chosen because it is [not
legal][object-names] in GCS object names, and therefore is not ambiguous.

[object-names]: https://cloud.google.com/storage/docs/bucket-naming#objectnames


<a name="mmaped-files"></a>
## Memory-mapped files

gcsfuse files can be memory-mapped for reading and writing using mmap(2). If
you make modifications to such a file and want to ensure that they are durable,
you must do the following:

*   Keep the file descriptor you supplied to mmap(2) open while you make your
    modifications.

*   When you are finished modifying the mapping, call msync(2) and check for
    errors.

*   Call munmap(2) and check for errors.

*   Call close(2) on the original file descriptor and check for errors.

If none of the calls returns an error, the modifications have been made durable
in GCS, according to the usual rules documented above.

See the notes on [fuseops.FlushFileOp][flush-op] for more details.

[flush-op]: http://godoc.org/github.com/jacobsa/fuse/fuseops#FlushFileOp


<a name="missing-features"></a>
## Missing features

Not all of the usual file system features are supported. Most prominently:

*   Renaming directories is not supported. A directory rename cannot be
    performed atomically in GCS and would therefore be arbitrarily expensive in
    terms of GCS operations, and for large directories would have high
    probability of failure, leaving the two directories in an inconsistent
    state.

*   File and directory permissions and ownership cannot be changed. See the
    [section](#permissions-and-ownership) above.

*   Modification times are not tracked for any inodes except for files.

*   No other times besides modification time are tracked. For example, ctime
    and atime are not tracked (but will be set to something reasonable).
    Requests to change them will appear to succeed, but the results are
    unspecified.

This document defines the semantics of a gcsfuse file system mounted with a GCS
bucket, including how files and directories map to object names, what
consistency guarantees are made, etc.

# Buckets

GCS has a feature called [object versioning][versioning] that allows buckets to
be put into a mode where the history of each object is maintained, even when it
is overwritten or deleted. gcsfuse does not support such buckets, and the file
system semantics discussed below do not apply to buckets in this mode—the
behavior for such buckets is undefined.

[versioning]: https://cloud.google.com/storage/docs/object-versioning


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

gcsfuse supports a flag called `--implicit_dirs` that changes the behavior.
When this flag is enabled, name lookup requests from the kernel use the GCS
API's Objects.list operation to search for objects that would implicitly define
the existence of a directory with the name in question. So, in the example
above, there would appear to be a directory named "foo".

The use of `--implicit_dirs` has some drawbacks (see [issue #7][issue-7] for a
more thorough discussion):

*   The feature requires an additional request to GCS for each name lookup,
    which may have costs in terms of request budget and latency.

*   GCS object listings are only eventually consistent, so directories that
    recently implicitly sprang into existence due to the creation of a child
    object may not show up for several minutes (or, in rare extreme cases,
    hours or days). Similarly, recently deleted objects may continue for a time
    to implicitly define directories that eventually wink out of existence.
    Even if an up to date listing is seen once, it is not guaranteed to be seen
    on the next lookup.

*   With this setup, it will appear as if there is a directory called "foo"
    containing a file called "bar". But when the user runs `rm foo/bar`,
    suddenly it will appear as if the file system is completely empty. This is
    contrary to expectations, since the user hasn't run `rmdir foo`.

*   gcsfuse sends a single Objects.list request to GCS, and treats the
    directory as being implicitly defined if the results are non-empty. In rare
    cases (notably when many objects have recently been deleted) Objects.list
    may return an abitrary number of empty response with continuation tokens,
    even for a non-empty name range. In order to bound the number of requests,
    gcsfuse simply ignores this subtlety. Therefore in rare cases an implicitly
    defined directory will fail to appear.

[issue-7]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/7


# Generations

GCS has a notion of an [object generation number][generations] associated with
each object name. These provide a total order on requests to modify an object,
compatible with causality. So if insert operation A happens before insert
operation B, then the generation number resulting from A will be less than that
resulting from B.

Although the GCS documentation doesn't define it this way, it is convenient
when discussing consistency to think of deletions as resulting in generations
as well, with a number also assigned according to this total order in a way
compatible with causality. Such a "tombstone" generation can be thought of as
having empty contents. Additionally, it is convenient to think of the initial
non-existent state of any object name as being generation number zero.

The discussion below uses the term "generation" in this manner.

[generations]: https://cloud.google.com/storage/docs/generations-preconditions


# File inodes

As in any file system, file inodes in a gcsfuse file system logically contain
file contents and metadata. A file inode is initialized with a particular
generation of a particular object within GCS (the "source generation"), and its
contents are initially exactly the contents of that generation.

### Modifications

Inodes may be opened for writing. Modifications are reflected immediately in
reads of the same inode by processes local to the machine using the same file
system. After a successful `fsync` or a successful `close`, the contents of the
inode are guaranteed to have been written to the GCS object with the matching
name if the object's generation number still matches the source generation
number of the inode. (It may not if there have been modifications from another
actor in the meantime.) There are no guarantees about whether local
modifications are reflected in GCS after writing but before syncing or closing.

Modification time (`stat::st_mtime` on Linux) is tracked for file inodes. No
other times are tracked.

### Identity

If a new generation number is assigned to a GCS object due to a flush from an
inode, the source generation number of the inode is updated and the inode ID
remains stable. Otherwise, if a new generation is created by another machine or
in some other manner from the local machine, the new generation is treated as
an inode distinct from any other inode already created for the object name.
Inode IDs are local to a single gcsfuse process, and there are no guarantees
about their stability across machines or invocations on a single machine.

### Lookups

One of the fundamental operations in the VFS layer of the kernel is looking up
the inode for a particular name within a directory. gcsfuse responds to such
lookups as follows:

1.  Stat the object with the given name within the GCS bucket.
2.  If the object does not exist, return an error.
3.  Call the generation number of the object N. If there is already an inode
    for this name with source generation number N, return it.
4.  Create a new inode for this name with source generation number N.

### User-visible semantics

The intent of these conventions is to make it appear as though local writes to
a file are in-place modifications as with a traditional file system, whereas
remote overwrites of a GCS object appear as some other process unlinking the
file from its directory and then linking a distinct file using the same name.
The `st_nlink` field will reflect this when using fstat(2) if `--support_nlink`
is set; see below.

Note the following consequence: if machine A opens a file and writes to it,
then machine B deletes or replaces its backing object, then machine A closes
the file, machine A's writes will be lost. This matches the behavior on a
single machine when process A opens a file and then process B unlinks it.
Process A continues to have a consistent view of the file's contents until it
closes the file handle, at which point the contents are lost.


# Directory inodes

gcsfuse directory inodes exist simply to satisfy the kernel and export a way to
look up child inodes. Unlike file inodes:

*   There are no guarantees about stability of directory inode IDs. They may
    change from lookup to lookup even if nothing has changed in the GCS bucket.
    They may not change even if the directory object in the bucket has been
    overwritten.

*   gcsfuse does not keep track of modification time for
    directories. There are no guarantees for the contents of `stat::st_mtime`
    or equivalent.

### Reading

Unlike reads for a particular object, listing operations in GCS are
[eventually consistent][consistency]. This means that directory listings in
gcsfuse may be arbitrarily far out of date. Additionally, seeing a fresh
listing once does not imply that future listings will be fresh. This applies at
the user level to commands like `ls`, and to the posix interfaces they use like
`readdir`.

[consistency]: https://cloud.google.com/storage/docs/concepts-techniques#consistency


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


# Permissions and ownership

All inodes in a gcsfuse file system show up as being owned by the UID and GID
of the gcsfuse process itself (i.e. the user who mounted the file system). All
inodes have permission bits `0700`, i.e. read-write-execute permission for the
owner but no one else.

Rationale: the FUSE kernel layer restricts file system access to the mounting
user (cf. [fuse.txt][allow_other]), so it is not worth supporting arbitrary
inode owners. Additionally, posix permissions bits are not nearly as expressive
as GCS object/bucket ACLs, so we cannot faithfully represent the latter and to
do so would be misleading since it would not necessarily offer any security.

[allow_other]: https://github.com/torvalds/linux/blob/a33f32244d8550da8b4a26e277ce07d5c6d158b5/Documentation/filesystems/fuse.txt##L102-L105


# Surprising behaviors

## Unlinking directories

Because GCS offers no way to delete an object conditional on the non-existence
of other objects, there is no way for gcsfuse to unlink a directory if and only
if it is empty. So gcsfuse takes the simple route, and always allows a
directory to be unlinked, even if non-empty. The contents of a non-empty
directory that is unlinked are not deleted but simply become inaccessible—the
placeholder object for the unlinked directory is simply removed. (Unless
`--implicit_dirs` is set; see the section on implicit directories above.)


## Reading directories

gcsfuse implements requests from the kernel to read the contents of a directory
(as when listing a directory with `ls`, for example) by calling
[Objects.list][] in the GCS API. The call uses a delimiter of `/` to avoid
paying the bandwidth and request cost of also listing very large
sub-directories.

[Objects.list]: https://cloud.google.com/storage/docs/json_api/v1/objects/list

However, with this implementation there is no way for gcsfuse to distinguish a
child directory that actually exists (because its placeholder object is
present) and one that is only implicitly defined. So when `--implicit_dirs` is
not set, directory listings may contain names that are inaccessible in a later
call from the kernel to gcsfuse to look up the inode by name. For example, a
call to readdir(3) may return names for which fstat(2) returns `ENOENT`.


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
file named `foo\n` (i.e. `foo` followed by U+000A, line feed). This is what
will appear when the parent's directory entries are read, and gcsfuse will
respond to requests to look up the inode named `foo\n` by returning the file
inode. `\n` in particular is chosen because it is [not legal][object-names] in
GCS object names, and therefore is not ambiguous.

[object-names]: https://cloud.google.com/storage/docs/bucket-naming#objectnames


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


## Inode link counts

By default, fstat(2) always shows a value of one for `stat::st_nlink` for any
gcsfuse inode. This saves a round trip to GCS for getattr requests, which can
add up to significant time savings when e.g. doing `ls -l` on a large
directory.

This behavior can be changed with the `--support_nlink` flag, which will cause
a stat request to be sent to GCS. Inodes for names that have been deleted or
overwritten will then show a value of zero for `stat::st_nlink`.


## Missing features

Not all of the usual file system features are supported. Most prominently:

*   Renaming is not currently supported. See [issue #11][issue-11].
*   Symlinks are not currently supported. See [issue #12][issue-12].

[issue-11]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/11
[issue-12]: https://github.com/GoogleCloudPlatform/gcsfuse/issues/12


<a name="relax-for-performance"></a>
# Relaxing guarantees for performance

The cost of the consistency guarantees discussed above is that gcsfuse must
frequently send stat object requests to GCS in order to get the freshest
possible answer for the kernel when it asks about a particular name or inode,
which it does often. This can make what appear to the user to be simple
operations, like `ls -l`, take quite a long time.

To alleviate this slowness, gcsfuse supports using cached data where it would
otherwise send a stat object request to GCS, saving some round trips. To enable
this behavior, set the flag `--stat_cache_ttl` to a value like `10s` or `1.5h`.
Positive and negative stat results will be cached for the given amount of time.

**Warning**: Setting `--stat_cache_ttl` breaks the consistency guarantees
discussed in this document. It is safe in the following situations:

 *  The mounted bucket is never modified.
 *  The mounted bucket is only modified on a single machine, via a single
    gcsfuse mount.
 *  You are otherwise confident that you do not need the guarantees discussed
    in this document.

This document defines the semantics of a gcsfuse file system mounted with a GCS
bucket, including how files and directories map to object names, what
consistency guarantees are made, etc.

*WARNING*: This document is aspirational. As of 2015-03-04, it does not yet
reflect reality. TODO(jacobsa): Update this warning when this is no longer
aspirational.

# Files and directories

GCS object names map directly to file paths using the separator '/'. Object
names ending in a slash represent a directory, and all other object names
represent a file. Directories are not implicitly defined; they exist only if a
matching object ending in a slash exists.

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

## Caveat

As mentioned above, there is no allowance for the implicit existence of
directories. Since the usual file system operations like `mkdir` will do the
right thing, if you set up a bucket's structure using only gcsfuse then you
will not notice anything odd about this. If, however, you use some other tool
to set up objects in GCS, you may notice that not all objects are visible until
you create leading directories for them.

For example, say that you use some other tool to set up a single object named
"foo/bar" in your bucket, then mount the bucket with gcsfuse. The file system
will initially appear empty, since there is no "foo/" object. However if you
subsequently run `mkdir foo`, you will now see a directory named "foo"
containing a file named "bar".

The alternative is to have an object named "foo/bar/baz" implicitly define a
directory named "foo" with a child directory named "bar". This would work fine
if GCS offered consistent object listing, but it does not: object listings
may be arbitrarily far out of date and seeing a fresh listing once does not
guarantee you will see it again. Because of this, implicit definition of
directories would cause problems for consistency guarantees (see the
consistency section below):

*   Imagine the initial contents of the bucket are a single recently-created
    object named "foo/bar/baz".

*   Say that machine A can see this object in a listing and runs
    `echo "hello" > foo/bar/qux`. This should work because the intermediate
    directories exist implicitly.

*   Say that machine B then runs `cat foo/bar/qux`.

The `cat` command on machine B will result in the following sequence of calls
from the kernel VFS layer to gcsfuse:

*   Look up the inode for 'foo' within the root. Call it F.
*   Look up the inode for 'bar' within F. Call it B.
*   Look up the inode for 'qux' within B. Call it Q.
*   Open Q for reading. Call the resulting handle H.
*   Read from H.

However, it is possible (and even likely) that machine B will not be able to
see either of the two objects in a GCS object listing. Therefore when it
receives the first lookup request from the kernel it will neither be able to
find an object named "foo/" nor be able to see such a prefix in a listing of
the root of the bucket, and it will have no choice but to return `ENOENT`.
So the `cat` command will result in a "no such file or directory" error. This
violates our close-to-open consistency guarantee documented below -- after
machine A successfully writes the file, machine B should be able to read it.


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
file contents and metadata. A file inode is iniitalized with a particular
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
The `st_nlink` field will reflect this when using `fstat`.

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

gcsfuse attempts to paper over this issue somewhat by remembering local
modifications (creations and removals) for some period of time, so that e.g.
creating a file and then listing its parent directory on the same machine will
result in the expected experience. However note that this means that e.g. a
subsequent deletion of the file on another machine will not be reflected in a
directory listing on the creating machine for that time period.


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

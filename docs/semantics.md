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
may be arbitrarily far out of date. Because of this, implicit definition of
directories would cause problems for consistency guarantees (see the
consistency section below):

*   Imagine the initial contents of the bucket are a single recently-created
    object named "foo/bar/baz".

*   Say that machine A can see this object in a listing and runs
    `echo "hello" > foo/bar/qux`. This should work because the intermediate
    directories exist implicitly.

*   Say that machine B then runs `cat foo/bar/qux`.

The `cat` command on machine B will result in the following series of calls
from the kernel VFS layer to gcsfuse:

*   Look up the inode for 'foo' within the root. Call it F.
*   Look up the inode for 'bar' within F. Call it B.
*   Look up the inode for 'qux' within B. Call it Q.
*   Open Q for reading. Call the handle H.
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
having empty contents. The definitions below include such generations.

[generations]: https://cloud.google.com/storage/docs/generations-preconditions


# File contents

When you open a file, the contents you observe inside of it are as follows:

*   If the file is already opened on the local machine via the same file system,
    in your process or in another, the contents are the same as the contents
    seen by that file descriptor. This includes any local modifications that
    have not yet been flushed (see below).

*   Otherwise, if the creation of some generation of an object (including
    tombstone generations for deletions) happened before the open operation,
    then the contents will be some particular generation of the GCS object. See
    the section on consistency below.

*   Otherwise, the contents will be some particular generation of the GCS
    object (created concurrently with the open and first read) or the empty
    string.

Less precisely: you will see the same contents as all other processes on your
machine, and those contents will initially consist of the contents of some
generation of the object, or the file may be empty if the object was not
created before opening the file.

The contents of an open file do *not* spontaneously change when a new
generation of the object is changed. If you open a file and read a bit, then
overwrite or delete the object in GCS, your file handle will still observe the
same contents.


# Writing, syncing, and closing

Files may be opened for writing. Modifications are reflected immediately in
reads of the file by processes local to the machine using the same file system.

After a successful `fsync` or a successful `close`, modifications to a file are
guaranteed to be reflected in the underlying GCS object. There are no
guarantees about whether they are reflected in GCS after writing but before
syncing or closing.


# Write/read consistency

gcsfuse offers close-to-open consistency. This means that any writes made to a
file before a successful `close` are guaranteed to be visible to an `open` that
happens after the `close` completes, on the same or a remote machine. The same
is true of an `fsync` to `open` sequence.

More precisely: Let O be an object in GCS, and A and B be machines. Say that A
opens a file backed by O, makes some modifications to it and then sees `close`
return successfully. Because `close` returned successfully, the contents that A
saw at `close` time were committed to some generation G of O.

Now say that B opens a file backed by O, that B did not already have an open
handle for the file, and that the call to `close` on A happened before the call
to `open` on B. Then the contents of the file as seen by B are guaranteed to be
exactly the contents of some generation G' of O such that `G <= G'`.


# Listing consistency

TODO(jacobsa)

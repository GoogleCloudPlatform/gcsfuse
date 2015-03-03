This document defines the semantics of a gcsfuse file system mounted with a GCS
bucket, including how files and directories map to object names, what
consistency guarantees are made, etc.

*WARNING*: This document is aspirational. As of 2015-03-04, it does not yet
reflect reality. TODO(jacobsa): Update this warning when this is no longer
aspirational.

## Files and directories

GCS object names map directly to file paths using the separator '/', with
directories implicitly defined by the existence of an object containing the
directory path as a prefix. Directories may also be explicitly defined (whether
empty or not) by an object with a trailing slash in its name.

This is all much clearer with an example. Say that the GCS bucket contains the
following objects:

*   burrito/
*   enchilada/
*   enchilada/0
*   enchilada/1
*   queso/carne/carnitas
*   queso/carne/nachos/
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
             carnitas
             nachos/
     taco

In particular, note that some directories are explicitly defined by a
placeholder object, whether empty (burrito/, queso/carne/nachos/) or non-empty
(enchilada/), and others are implicitly defined by their children
(queso/carne/).

The full technical definition is in [listing_proxy.go][].

[listing_proxy.go]: https://github.com/jacobsa/gcsfuse/blob/bb13286d818c6fd76262bf559f1a386c109f3638/gcsproxy/listing_proxy.go#L33-L81


## Generations

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


## File contents

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


## Writing, syncing, and closing

Files may be opened for writing. Modifications are reflected immediately in
reads of the file by processes local to the machine using the same file system.

After a successful `fsync` or a successful `close`, modifications to a file are
guaranteed to be reflected in the underlying GCS object. There are no
guarantees about whether they are reflected in GCS after writing but before
syncing or closing.


## Write/read consistency

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

### Caveat

There is one caveat to this guarantee, due to the lack of listing consistency
in GCS and to the way the kernel VFS layer works. Although the guarantee on
file contents above holds when `open` succeeds, it is possible that `open` will
fail with `ENOENT` if there are no placeholder objects for the directory
ancestors of the object being opened. This will persist until requests to GCS
to list the directories involved return up to date responses.

For example, say that A does this:

    cd /mount_point
    mkdir -p foo/bar
    echo "hello" > foo/bar/baz

and then B does this:

    cd /mount_point
    cat foo/bar/baz

This will result in the kernel issuing the following set of requests to gcsfuse
on B:

*   Look up the inode for 'foo' within the root. Call it F.
*   Look up the inode for 'bar' within F. Call it B.
*   Look up the inode for 'baz' within B. Call it B'.
*   Open B' for reading. Call the handle H.
*   Read from H.

`mkdir -p` on A will ensure that there are inodes for each of the directories,
and GCS's write-to-read consistency will ensure that B can see them during the
lookup requests. So this will work fine.

But now assume that the initial contents of the bucket are a single object
named "foo/bar/qux", very recently created, and A does only the following:

    cd /mount_point
    echo "hello" > foo/bar/baz

This will succeed because the directory "foo/bar" already implicitly exists.
But unlike before it will not result in the creation of objects named "foo/"
and "foo/bar/". Therefore if GCS does not yet show any of the objects in a
request to list objects, there is no way that the first lookup request from the
kernel on B can succeed, and the `echo` command will fail.

TODO(jacobsa): We are probably causing this problem for ourselves. Perhaps we
should adopt the following convention: there is no 'real' directory inode
unless the placeholder object exists. That is, there is no implicit definition
of directories. So when we do Objects.list we follow by stat requests for the
placeholder objects, and filter out prefixes that have no result. If the user
sets up their directory structure only via gcsfuse this works out fine. If they
have existing objects (like the "foo/bar/qux" case), then the objects will be
inaccessible via gcsfuse until they create the directories leading up to them.


## Listing consistency

TODO(jacobsa)

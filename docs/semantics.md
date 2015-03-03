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


## Writing, flushing, and closing

TODO (jacobsa)


## Write/read consistency

TODO (jacobsa)


## Listing consistency

TODO(jacobsa)

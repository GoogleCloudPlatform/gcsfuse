This document defines the semantics of a gcsfuse file system mounted with a GCS
bucket, including how files and directories map to object names, what
consistency guarantees are made, etc.

*WARNING*: This document is aspirational. As of 2015-03-04, it does not yet
reflect reality. TODO(jacobsa): Update this doc.

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

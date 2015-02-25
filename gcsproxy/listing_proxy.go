// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// A view on a "directory" in GCS that caches listings and modifications.
//
// Directories are by convention defined by '/' characters in object names. A
// directory is uniquely identified by an object name prefix that ends with a
// '/', or the empty string for the root directory. Given such a prefix P, the
// contents of directory P are:
//
//  *  The "files" within the directory: all objects named N such that
//      *  N has P as a prefix.
//      *  N is not equal to P.
//      *  The portion of N following the prefix P contains no slashes.
//
//  *  The immediate "sub-directories": all strings P' such that
//      *  P' is a legal directory prefix according to the definition above.
//      *  P' has P as a prefix.
//      *  P' is not equal to P.
//      *  The portion of P' following the prefix P contains exactly one slash.
//      *  There is at least one objcet with name N such that N has P' as a
//         prefix.
//
// So for example, imagine a bucket contains the following objects:
//
//  *  burrito/
//  *  enchilada/
//  *  enchilada/0
//  *  enchilada/1
//  *  queso/carne/carnitas
//  *  queso/carne/nachos/
//  *  taco
//
// Then the directory structure looks like the following, where a trailing
// slash indicates a directory and the top level is the contents of the root
// directory:
//
//     burrito/
//     enchilada/
//         0
//         1
//     queso/
//         carne/
//             carnitas
//             nachos/
//     taco
//
// In particular, note that some directories are explicitly defined by a
// placeholder object, whether empty (burrito/, queso/carne/nachos/) or
// non-empty (enchilada/), and others are implicitly defined by
// their children (queso/carne/).
type ListingProxy struct {
}

// XXX: Comments
func NewListingProxy(
	bucket gcs.Bucket,
	clock timeutil.Clock,
	dir string) (lp *ListingProxy, err error)

// XXX: Comments
func (lp *ListingProxy) List(
	ctx context.Context) (objects []*storage.Object, subdirs []string, err error)

// XXX: Comments
func (lp *ListingProxy) NoteAddition(o *storage.Object) (err error)

// XXX: Comments
func (lp *ListingProxy) NoteRemoval(name string) (err error)

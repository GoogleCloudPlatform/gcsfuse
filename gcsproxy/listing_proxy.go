// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"time"

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
//      *  P is a strict prefix of N.
//      *  The portion of N following the prefix P contains no slashes.
//
//  *  The immediate "sub-directories": all strings P' such that
//      *  P' is a legal directory prefix according to the definition above.
//      *  P is a strict prefix of P'.
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

// How long we cache the most recent listing for a particular directory from
// GCS before regarding it as stale.
//
// Intended to paper over performance issues caused by quick follow-up calls;
// for example when the fuse VFS performs a readdir followed quickly by a
// lookup for each child. The drawback is that this increases the time before a
// write by a foreign machine within a recently-listed directory will be seen
// locally.
//
// TODO(jacobsa): Do we need this at all? Maybe the VFS layer does appropriate
// caching. Experiment with setting it to zero or ripping out the code.
//
// TODO(jacobsa): Set this according to real-world performance issues when the
// kernel does e.g. ReadDir followed by Lookup. Can probably be set quite
// small.
//
// TODO(jacobsa): Can this be moved to a decorator implementation of gcs.Bucket
// instead of living here?
const ListingProxy_ListingCacheTTL = 10 * time.Second

// How long we remember that we took some action on the contents of a directory
// (linking or unlinking), and pretend the action is reflected in the listing
// even if it is not reflected in a call to Bucket.ListObjects.
//
// Intended to paper over the fact that GCS doesn't offer list-your-own-writes
// consistency: it may be an arbitrarily long time before you see the creation
// or deletion of an object in a subsequent listing, and even if you see it in
// one listing you may not see it in the next. The drawback is that foreign
// modifications to recently-locally-modified directories will not be reflected
// locally for awhile.
//
// TODO(jacobsa): Set this according to information about listing staleness
// distributions from the GCS team.
//
// TODO(jacobsa): Can this be moved to a decorator implementation of gcs.Bucket
// instead of living here?
const ListingProxy_ModificationMemoryTTL = 5 * time.Minute

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

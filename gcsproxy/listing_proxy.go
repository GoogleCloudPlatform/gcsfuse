// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"container/list"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
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
//
// Not safe for concurrent access. The user must provide external
// synchronization if necessary.
type ListingProxy struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	bucket gcs.Bucket
	clock  timeutil.Clock

	/////////////////////////
	// Constant data
	/////////////////////////

	// INVARIANT: checkDirName(name) == nil
	name string

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Our current best understanding of the contents of the directory in GCS,
	// formed by listing the bucket and then patching according to child
	// modification records at the time, and patched since then by subsequent
	// modifications.
	//
	// The time after which this should be generated anew from a new listing is
	// also stored. This is set to the time at which the listing completed plus
	// the listing cache TTL.
	//
	// Sub-directories are of type string, and objects are of type
	// *storage.Object.
	//
	// INVARIANT: contents != nil
	// INVARIANT: All values are of type string or *storage.Object.
	// INVARIANT: For all string values v, checkDirName(v) == nil
	// INVARIANT: For all string values v, name is a strict prefix of v
	// INVARIANT: For all object values o, checkDirName(o.Name) != nil
	// INVARIANT: For all object values o, name is a strict prefix of o.Name
	// INVARIANT: All entries are indexed by the correct name.
	contents           map[string]interface{}
	contentsExpiration time.Time

	// A collection of children that have recently been added or removed locally
	// and the time at which it happened, ordered by the sequence in which it
	// happened. Elements M with M.node == nil are removals; all others are
	// additions.
	//
	// For a record M in this list with M's age less than the modification TTL,
	// any listing from the bucket should be augmented by pretending M just
	// happened.
	//
	// INVARIANT: All elements are of type childModification.
	// INVARIANT: Contains no duplicate names.
	// INVARIANT: For each M with M.node == nil, contents does not contain M.name.
	// INVARIANT: For each M with M.node != nil, contents[M.name] == M.node.
	childModifications list.List

	// An index of childModifications by name.
	//
	// INVARIANT: childModificationsIndex != nil
	// INVARIANT: For all names N in the map, the indexed modification has name N.
	// INVARIANT: Contains exactly the set of names in childModifications.
	childModificationsIndex map[string]*list.Element
}

// See ListingProxy.childModifications.
type childModification struct {
	time time.Time
	name string

	// INVARIANT: node == nil or node is of type string or *storage.Object
	node interface{}
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

// Create a listing proxy object for the directory identified by the given
// prefix (see comments on ListingProxy). The supplied clock will be used for
// cache TTLs.
func NewListingProxy(
	bucket gcs.Bucket,
	clock timeutil.Clock,
	dir string) (lp *ListingProxy, err error) {
	// Make sure the directory name is legal.
	if err = checkDirName(dir); err != nil {
		err = fmt.Errorf("Illegal directory name (%v): %s", err, dir)
		return
	}

	// Create the object.
	lp = &ListingProxy{
		bucket:                  bucket,
		clock:                   clock,
		name:                    dir,
		contents:                make(map[string]interface{}),
		childModificationsIndex: make(map[string]*list.Element),
	}

	return
}

// Return the directory prefix with which this object was configured.
func (lp *ListingProxy) Name() string {
	return lp.name
}

// Panic if any internal invariants are violated. Careful users can call this
// at appropriate times to help debug weirdness. Consider using
// syncutil.InvariantMutex to automate the process.
func (lp *ListingProxy) CheckInvariants() {
	if err := checkDirName(lp.name); err != nil {
		panic("Illegal name: " + err.Error())
	}

	// Check that maps are non-nil.
	if lp.contents == nil || lp.childModificationsIndex == nil {
		panic("Expected contents and childModificationsIndex to be non-nil.")
	}

	// Check each element of the contents map.
	for name, node := range lp.contents {
		switch typedNode := node.(type) {
		default:
			panic(fmt.Sprintf("Bad type for node: %v", node))

		case string:
			// Sub-directory
			if name != typedNode {
				panic(fmt.Sprintf("Name mismatch: %s vs. %s", name, typedNode))
			}

			if err := checkDirName(typedNode); err != nil {
				panic("Illegal directory name: " + typedNode)
			}

		case *storage.Object:
			if name != typedNode.Name {
				panic(fmt.Sprintf("Name mismatch: %s vs. %s", name, typedNode.Name))
			}

			if err := checkDirName(typedNode.Name); err == nil {
				panic("Illegal object name: " + typedNode.Name)
			}
		}
	}

	// Check each child modification. Build a list of names we've seen while
	// doing so.
	var listNames sort.StringSlice
	for e := lp.childModifications.Front(); e != nil; e = e.Next() {
		m := e.Value.(childModification)
		listNames = append(listNames, m.name)

		if m.node == nil {
			if n, ok := lp.contents[m.name]; ok {
				panic(fmt.Sprintf("lp.contents[%s] == %v for removal", m.name, n))
			}
		} else {
			if n := lp.contents[m.name]; n != m.node {
				panic(fmt.Sprintf("lp.contents[%s] == %v, not %v", m.name, n, m.node))
			}
		}
	}

	sort.Sort(listNames)

	// Check that there were no duplicate names.
	for i, name := range listNames {
		if i == 0 {
			continue
		}

		if name == listNames[i-1] {
			panic("Duplicated name in childModifications: " + name)
		}
	}

	// Check the index. Build a list of names it contains While doing so.
	var indexNames sort.StringSlice
	for name, e := range lp.childModificationsIndex {
		indexNames = append(indexNames, name)

		m := e.Value.(childModification)
		if m.name != name {
			panic(fmt.Sprintf("Index name mismatch: %s vs. %s", m.name, name))
		}
	}

	sort.Sort(indexNames)

	// Check that the index contains the same set of names.
	if !reflect.DeepEqual(listNames, indexNames) {
		panic(fmt.Sprintf("Names mismatch:\n%v\n%v", listNames, indexNames))
	}
}

// Obtain a listing of the objects directly within the directory and the
// immediate sub-directories. (See comments on ListingProxy for precise
// semantics.) Object and sub-directory names are fully specified, not
// relative.
//
// This listing reflects any additions and removals set up with NoteNewObject,
// NoteNewSubdirectory, or NoteRemoval.
func (lp *ListingProxy) List(
	ctx context.Context) (objects []*storage.Object, subdirs []string, err error) {
	// List the directory.
	query := &storage.Query{
		Delimiter: "/",
		Prefix:    lp.name,
	}

	if objects, subdirs, err = gcsutil.List(ctx, lp.bucket, query); err != nil {
		err = fmt.Errorf("gcsutil.List: %v", err)
		return
	}

	// Make sure the response is valid.
	for _, o := range objects {
		if err = checkDirName(o.Name); err == nil {
			err = fmt.Errorf("Illegal object name returned by List: %s", o.Name)
			return
		}
	}

	for _, subdir := range subdirs {
		if err = checkDirName(subdir); err != nil {
			err = fmt.Errorf(
				"Illegal directory name returned by List (%v): %s",
				err,
				subdir)
			return
		}
	}

	return
}

// Note that an object has been added to the directory, overriding any previous
// additions or removals with the same name. For awhile after this call, the
// response to a call to List will contain this object even if it is not
// present in a listing from the underlying bucket.
func (lp *ListingProxy) NoteNewObject(o *storage.Object) (err error) {
	err = errors.New("TODO: Implement NoteNewObject.")
	return
}

// Note that a sub-directory has been added to the directory, overriding any
// previous additions or removals with the same name. For awhile after this
// call, the response to a call to List will contain this object even if it is
// not present in a listing from the underlying bucket.
//
// The name must be a legal directory prefix for a sub-directory of this
// directory. See notes on ListingProxy for more details.
func (lp *ListingProxy) NoteNewSubdirectory(name string) (err error) {
	err = errors.New("TODO: Implement NoteNewSubdirectory.")
	return
}

// Note that an object or directory prefix has been removed from the directory,
// overriding any previous additions or removals. For awhile after this call,
// the response to a call to List will not contain this name even if it is
// present in a listing from the underlying bucket.
func (lp *ListingProxy) NoteRemoval(name string) (err error) {
	err = errors.New("TODO: Implement NoteRemoval.")
	return
}

func checkDirName(name string) (err error) {
	if name == "" || name[len(name)-1] == '/' {
		return
	}

	err = errors.New("Non-empty names must end with a slash")
	return
}

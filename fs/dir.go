// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

const dirSeparator = '/'

// Implementation detail, do not touch.
//
// How long we cache the most recent listing for a particular directory from
// GCS before regarding it as stale.
//
// Intended to paper over performance issues caused by quick follow-up calls;
// for example when the fuse VFS performs a readdir followed quickly by a
// lookup for each child. The drawback is that this increases the time before a
// write by a foreign machine within a recently-listed directory will be seen
// locally.
//
// TODO(jacobsa): Set this according to real-world performance issues when the
// kernel does e.g. ReadDir followed by Lookup. Can probably be set quite
// small.
//
// TODO(jacobsa): Can this be moved to a decorator implementation of gcs.Bucket
// instead of living here?
var ListingCacheTTL = 10 * time.Second

// Implementation detail, do not touch.
//
// How long we remember that we took some action on the contents of a directory
// (linking or unlinking), and pretend the action is reflected in the listing
// even if it is not.
//
// Intended to paper over the fact that GCS doesn't offer list-your-own-writes
// consistency: it may be an arbitrarily long time before you see the creation
// or deletion of an object in a subsequent listing, and even if you see it in
// one listing you may not in the next. The drawback is that modifications to
// recently-modified directories by foreign machines will not be reflected
// locally for awhile.
//
// TODO(jacobsa): Set this according to information about listing staleness
// distributions from the GCS team.
//
// TODO(jacobsa): Can this be moved to a decorator implementation of gcs.Bucket
// instead of living here?
var ChildActionMemoryTTL = 5 * time.Minute

// See the childModifications field of dir.
type childModification struct {
	time time.Time
	name string
	node fusefs.Node
}

// A "directory" in GCS, defined by an object name prefix.
//
// For example, if the bucket contains objects "foo/bar" and "foo/baz", this
// implicitly defines the directory "foo/". No matter what the contents of the
// bucket, there is an implicit root directory "".
type dir struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	logger *log.Logger
	clock  timeutil.Clock
	bucket gcs.Bucket

	/////////////////////////
	// Constant data
	/////////////////////////

	// INVARIANT: objectPrefix is "" (representing the root directory) or ends
	// with dirSeparator.
	objectPrefix string

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// Our current best understanding of the contents of the directory in GCS,
	// formed by listing the bucket and then patching according to child addition
	// and removal records at the time, and patched since then by subsequent
	// additions and removals.
	//
	// The time after which this should be generated anew from a new listing is
	// also stored. This is set to the time at which the listing completed plus
	// ListingCacheTTL.
	//
	// INVARIANT: contents != nil
	// INVARIANT: All nodes are of type *dir or *file.
	// INVARIANT: All nodes are indexed by names that agree with node contents.
	contents           map[string]fusefs.Node // GUARDED_BY(mu)
	contentsExpiration time.Time              // GUARDED_BY(mu)

	// A collection of children that have recently been added or removed locally
	// and the time at which it happened, ordered by the sequence in which it
	// happened. Elements M with M.node == nil are removals; all others are
	// additions.
	//
	// For a record M in this list with M's age less than ChildActionMemoryTTL,
	// any listing from the bucket should be augmented by pretending M just
	// happened.
	//
	// TODO(jacobsa): Make sure to test link followed by unlink, and unlink
	// followed by link.
	//
	// TODO(jacobsa): Make sure to test that these expire eventually, i.e. that
	// foreign overwrites, deletes, and recreates are reflected eventually.
	//
	// INVARIANT: All elements are of type childModification.
	// INVARIANT: Contains no duplicate names.
	// INVARIANT: For each M with M.node == nil, contents does not contain M.name.
	// INVARIANT: For each M with M.node != nil, contents[M.name] == M.node.
	childModifications list.List // GUARDED_BY(mu)

	// An index of childModifications by name.
	//
	// INVARIANT: childModificationsIndex != nil
	// INVARIANT: For all names N in the map, the indexed modification has name N.
	// INVARIANT: Contains exactly the set of names in childModifications.
	childModificationsIndex map[string]*list.Element // GUARDED_BY(mu)
}

// Make sure dir implements the interfaces we think it does.
var (
	// TODO(jacobsa): I think we want to embed fusefs.NdoeRef in all of our
	// fusefs.Node types, so that we better benefit from fusefs.Server node
	// caching.
	_ fusefs.Node               = &dir{}
	_ fusefs.NodeCreater        = &dir{}
	_ fusefs.NodeMknoder        = &dir{}
	_ fusefs.NodeStringLookuper = &dir{}

	_ fusefs.Handle             = &dir{}
	_ fusefs.HandleReadDirAller = &dir{}
)

func newDir(
	logger *log.Logger,
	clock timeutil.Clock,
	bucket gcs.Bucket,
	objectPrefix string) *dir {
	// Check the object prefix.
	if objectPrefix != "" && objectPrefix[len(objectPrefix)-1] != dirSeparator {
		panic("Unexpected object prefix: " + objectPrefix)
	}

	// Create the dir struct.
	d := &dir{
		logger:                  logger,
		clock:                   clock,
		bucket:                  bucket,
		objectPrefix:            objectPrefix,
		contents:                make(map[string]fusefs.Node),
		childModificationsIndex: make(map[string]*list.Element),
	}

	d.mu = syncutil.NewInvariantMutex(func() { d.checkInvariants() })

	return d
}

func (d *dir) checkInvariants() {
	// Check that maps are non-nil.
	if d.contents == nil || d.childModificationsIndex == nil {
		panic("Expected contents and childModificationsIndex to be non-nil.")
	}

	// Check each element of the contents map.
	for name, node := range d.contents {
		if node == nil {
			panic("nil node for name: " + name)
		}

		d, okDir := node.(*dir)
		f, okFile := node.(*file)
		if !okDir && !okFile {
			panic(fmt.Sprintf("Unexpected node type: %v", reflect.TypeOf(node)))
		}

		if okDir && name != path.Base(d.objectPrefix) {
			panic(fmt.Sprintf("Name mismatch: %s vs. %s", name, d.objectPrefix))
		}

		if okFile && name != path.Base(f.objectName) {
			panic(fmt.Sprintf("Name mismatch: %s vs. %s", name, f.objectName))
		}
	}

	// Check each child modification. Build a list of names we've seen while
	// doing so.
	var listNames sort.StringSlice
	for e := d.childModifications.Front(); e != nil; e = e.Next() {
		m := e.Value.(childModification)
		listNames = append(listNames, m.name)

		if m.node == nil {
			if n, ok := d.contents[m.name]; ok {
				panic(fmt.Sprintf("d.contents[%s] == %v for removal", m.name, n))
			}
		} else {
			if n := d.contents[m.name]; n != m.node {
				panic(fmt.Sprintf("d.contents[%s] == %v, not %v", m.name, n, m.node))
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
	for name, e := range d.childModificationsIndex {
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

// Ensure that d.contents is fresh and usable. Must be called before using
// d.contents.
//
// TODO(jacobsa): If contents hasn't expired, return immediately. Otherwise
// list parasitically while holding the lock (why not, we can make this more
// subtle later if we must) and modify the result appropriately.
//
// EXCLUSIVE_LOCKS_REQUIRED(d.mu)
func (d *dir) ensureContents(ctx context.Context) error {
	// Are the contents already fresh?
	if d.clock.Now().Before(d.contentsExpiration) {
		return nil
	}

	// Grab a listing.
	query := &storage.Query{
		Delimiter: string(dirSeparator),
		Prefix:    d.objectPrefix,
	}

	objects, prefixes, err := gcsutil.List(ctx, d.bucket, query)
	if err != nil {
		return fmt.Errorf("gcsutil.List: %v", err)
	}

	// Convert the listing into a contents map.
	d.contents = make(map[string]fusefs.Node)

	for _, o := range objects {
		// Special case: the GCS storage browser's "New folder" button causes an
		// object with a trailing slash to be created. For example, if you make a
		// folder called "bar" within a folder called "foo", that creates an object
		// called "foo/bar/". (It seems like the redundant ones may be removed
		// eventually, but that's not relevant.)
		//
		// When we list the directory "foo/", we don't want to return the object
		// named "foo/" as if it were a file within the directory. So skip it.
		if o.Name == d.objectPrefix {
			continue
		}

		d.contents[path.Base(o.Name)] =
			newFile(
				d.logger,
				d.bucket,
				o.Name,
				uint64(o.Size))
	}

	for _, prefix := range prefixes {
		d.contents[path.Base(prefix)] =
			newDir(
				d.logger,
				d.clock,
				d.bucket,
				prefix)
	}

	// Choose an expiration time.
	d.contentsExpiration = d.clock.Now().Add(ListingCacheTTL)

	// TODO(jacobsa): Apply child modifications.
	return nil
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Reflect that we allow writes now. Make sure to test.
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0500,
	}
}

// LOCKS_EXCLUDED(d.mu)
func (d *dir) ReadDirAll(ctx context.Context) (ents []fuse.Dirent, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Printf("ReadDirAll: [%s]/%s", d.bucket.Name(), d.objectPrefix)

	// Ensure that we can use d.contents.
	if err = d.ensureContents(ctx); err != nil {
		err = fmt.Errorf("d.ensureContents: %v", err)
		return
	}

	// Read out entries from the contents map.
	for name, node := range d.contents {
		ent := fuse.Dirent{
			Name: name,
		}

		if _, ok := node.(*dir); ok {
			ent.Type = fuse.DT_Dir
		} else {
			ent.Type = fuse.DT_File
		}

		ents = append(ents, ent)
	}

	return
}

// LOCKS_EXCLUDED(d.mu)
func (d *dir) Lookup(
	ctx context.Context,
	name string) (n fusefs.Node, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Printf("Lookup: ([%s]/%s) %s", d.bucket.Name(), d.objectPrefix, name)

	// Ensure that we can use d.contents.
	if err = d.ensureContents(ctx); err != nil {
		err = fmt.Errorf("d.ensureContents: %v", err)
		return
	}

	// Find the object within the map.
	var ok bool
	if n, ok = d.contents[name]; ok {
		return
	}

	err = fuse.ENOENT
	return
}

func (d *dir) Create(
	ctx context.Context,
	req *fuse.CreateRequest,
	resp *fuse.CreateResponse) (
	fusefs.Node,
	fusefs.Handle,
	error) {
	// Tell fuse to use Mknod followed by Open, rather than re-implementing much
	// of Open here.
	return nil, nil, fuse.ENOSYS
}

func (d *dir) Mknod(
	ctx context.Context,
	req *fuse.MknodRequest) (fusefs.Node, error) {
	// The kernel appears to do the appropriate locking and querying to ensure
	// that vfs_mknod is called only when a child with the given name doesn't exist.
	//
	// For example, here are some relative bits from the implementation of the
	// mknodat system call:
	//
	//  *  http://goo.gl/QZecHk: mknodat calls user_path_create and later
	//     done_path_create. The former acquires an inode mutex via
	//     filename_create, and the latter releases it.
	//
	//  *  http://goo.gl/rbhFg4: filename_create calls lookup_hash and returns an
	//     error if it returns a positive dentry.
	//
	//  *  http://goo.gl/7Ea9D9: lookup_hash eventually calls lookup_real, which
	//     calls the inode's lookup method.
	//
	// So this process recently certified the child doesn't already exist. It's
	// possible some other process has created it in the meantime, but we don't
	// guarantee we won't clobber its writes.
	//
	// Therefore, create an empty object.
	panic("TODO")

	return nil, errors.New("TODO(jacobsa): Support Mknod.")
}

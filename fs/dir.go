// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/jacobsa/gcloud/gcs"
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
// kernel does e.g. ReadDir followed by LookUp. Can probably be set quite
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

	// TODO(jacobsa): Make sure to initialize this correctly.
	fixmemu syncutil.InvariantMutex

	// A cache of the most recent listing of the directory from GCS, expressed as
	// a map from (relative) names of children of this directory to nodes for
	// those children. May be nil if there is no most recent listing or if the
	// listing expired (see below).
	//
	// The time at which the listing completed is also stored. If clock.Now()
	// minus this time is more than ListingCacheTTL, the most recent listing
	// should not be used for any purpose.
	//
	// INVARIANT: All nodes are of type *dir or *file.
	mostRecentListing     map[string]fusefs.Node // GUARDED_BY(mu)
	mostRecentListingTime time.Time              // GUARDED_BY(mu)

	// A collection of children that have recently been added locally and the
	// time at which it happened. For a record R in this list with R's age less
	// than ChildActionMemoryTTL, any listing from the bucket should be augmented
	// by adding R, overwriting it if its name already exists. See
	// ChildActionMemoryTTL for more info.
	//
	// TODO(jacobsa): Make sure to test link followed by unlink, and unlink
	// followed by link.
	//
	// INVARIANT: All nodes are of type *dir or *file.
	childAdditions []childAddition // GUARDED_BY(mu)

	// A collection of children that have recently been removed locally and the
	// time at which it happened. For a record R in this list with R's age less
	// than ChildActionMemoryTTL, any listing from the bucket should be augmented
	// by removing any child with the name given by R. See ChildActionMemoryTTL
	// for more info.
	childRemovals []childRemoval // GUARDED_BY(mu)
}

// Make sure dir implements the interfaces we think it does.
var (
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
	return &dir{
		logger:       logger,
		clock:        clock,
		bucket:       bucket,
		objectPrefix: objectPrefix,
	}
}

// Initialize d.children from GCS if it has not already been populated or has
// expired.
//
// EXCLUSIVE_LOCKS_REQUIRED(d.mu)
func (d *dir) initChildren(ctx context.Context) error {
	// Have we already initialized the map and is it up to date?
	if d.children != nil && d.clock.Now().Before(d.childrenExpiry) {
		return nil
	}

	children := make(map[string]fusefs.Node)

	// List repeatedly until there is no more to list.
	query := &storage.Query{
		Delimiter: string(dirSeparator),
		Prefix:    d.objectPrefix,
	}

	for query != nil {
		// Grab one set of results.
		objects, err := d.bucket.ListObjects(ctx, query)
		if err != nil {
			return fmt.Errorf("bucket.ListObjects: %v", err)
		}

		// Extract objects as files.
		for _, o := range objects.Results {
			// Special case: the GCS storage browser's "New folder" button causes an
			// object with a trailing slash to be created. For example, if you make a
			// folder called "bar" within a folder called "foo", that creates an
			// object called "foo/bar/". (It seems like the redundant ones may be
			// removed eventually, but that's not relevant.)
			//
			// When we list the directory "foo/", we don't want to return the object
			// named "foo/" as if it were a file within the directory. So skip it.
			if o.Name == d.objectPrefix {
				continue
			}

			children[path.Base(o.Name)] =
				newFile(
					d.logger,
					d.bucket,
					o.Name,
					uint64(o.Size))
		}

		// Extract prefixes as directories.
		for _, p := range objects.Prefixes {
			children[path.Base(p)] = newDir(d.logger, d.clock, d.bucket, p)
		}

		// Move on to the next set of results.
		query = objects.Next
	}

	// Save the map.
	d.children = children
	d.childrenExpiry = d.clock.Now().Add(DirListingCacheTTL)

	return nil
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0500,
	}
}

func (d *dir) ReadDirAll(ctx context.Context) (
	ents []fuse.Dirent, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Printf("ReadDirAll: [%s]/%s", d.bucket.Name(), d.objectPrefix)

	// Ensure that our cache of children has been initialized.
	if err := d.initChildren(ctx); err != nil {
		return nil, fmt.Errorf("d.initChildren: %v", err)
	}

	// Read out the contents of the cache.
	for name, node := range d.children {
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

func (d *dir) Lookup(ctx context.Context, name string) (fusefs.Node, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Printf("Lookup: ([%s]/%s) %s", d.bucket.Name(), d.objectPrefix, name)

	// Ensure that our cache of children has been initialized.
	if err := d.initChildren(ctx); err != nil {
		return nil, fmt.Errorf("d.initChildren: %v", err)
	}

	// Find the object within the map.
	if n, ok := d.children[name]; ok {
		return n, nil
	}

	return nil, fuse.ENOENT
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
	return nil, errors.New("TODO(jacobsa): Support Mknod.")
}

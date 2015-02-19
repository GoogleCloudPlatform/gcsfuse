// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"

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

	// INVARIANT: nil or of type *file or *dir.
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
	// additions and removals. May be nil if no listing has happened or the
	// listing expired (see below).
	//
	// The time after which this should be generated anew from a new listing is
	// also stored. This is set to the time at which the listing completed plus
	// ListingCacheTTL.
	//
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
	// INVARIANT: For each M with M.node != nil,
	//              contents == nil || contents[M.name] == M.node.
	childModifications list.List // GUARDED_BY(mu)

	// An index of childModifications by name.
	//
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
	d := &dir{
		logger:       logger,
		clock:        clock,
		bucket:       bucket,
		objectPrefix: objectPrefix,
	}

	d.mu = syncutil.NewInvariantMutex(func() { d.checkInvariants() })

	return d
}

// Ensure that d.contents is fresh and usable. Must be called before using
// d.contents.
//
// TODO(jacobsa): If contents hasn't expired, return immediately. Otherwise
// list parasitically while holding the lock (why not, we can make this more
// subtle later if we must) and modify the result appropriately.
//
// EXCLUSIVE_LOCKS_REQUIRED(d.mu)
func (d *dir) ensureContents(ctx context.Context) error

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
	return nil, errors.New("TODO(jacobsa): Support Mknod.")
}

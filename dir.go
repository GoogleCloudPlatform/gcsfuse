// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"sync"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

const dirSeparator = '/'

// A "directory" in GCS, defined by an object name prefix. All prefixes end
// with dirSeparator except for the special case of the root directory, where
// the prefix is the empty string.
type dir struct {
	bucket       gcs.Bucket
	objectPrefix string

	mu sync.RWMutex

	// A map from (relative) names of children to nodes for those children,
	// initialized from GCS the first time it is needed for a ReadDir, Lookup,
	// etc. All nodes within the map are of type *dir or *file.
	children map[string]fs.Node // GUARDED_BY(mu)
}

// Initialize d.children from GCS if it has not already been populated.
func (d *dir) initChildren(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Have we already initialized the map?
	if d.children != nil {
		return nil
	}

	children := make(map[string]fs.Node)

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

			children[path.Base(o.Name)] = &file{
				bucket:     d.bucket,
				objectName: o.Name,
			}
		}

		// Extract prefixes as directories.
		for _, p := range objects.Prefixes {
			children[path.Base(p)] = &dir{
				bucket:       d.bucket,
				objectPrefix: p,
			}
		}

		// Move on to the next set of results.
		query = objects.Next
	}

	// Save the map.
	d.children = children

	return nil
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0500,
	}
}

func (d *dir) readDir(ctx context.Context) (
	ents []fuse.Dirent, fuseErr fuse.Error) {
	log.Printf("ReadDir: [%s]/%s", d.bucket.Name(), d.objectPrefix)

	// Ensure that our cache of children has been initialized.
	if err := d.initChildren(ctx); err != nil {
		log.Println("d.initChildren:", err)
		return nil, fuse.EIO
	}

	// Read out the contents of the cache.
	d.mu.RLock()
	defer d.mu.RUnlock()

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

func (d *dir) lookup(ctx context.Context, name string) (fs.Node, fuse.Error) {
	log.Printf("Lookup: ([%s]/%s) %s", d.bucket.Name(), d.objectPrefix, name)

	// Join the directory's prefix with this node's name to get the full name
	// that we expect to see in GCS (minus the slash that will be on it if it's a
	// prefix representing a directory).
	fullName := path.Join(d.objectPrefix, name)

	// We must determine whether this is a file or a directory. List objects
	// whose names start with fullName.
	//
	// HACK(jacobsa): As of 2015-02-05 the documentation here doesn't guarantee
	// that object listing results are ordered by object name:
	//
	//    https://cloud.google.com/storage/docs/json_api/v1/objects/list
	//
	// Therefore in theory we are not guaranteed to see the object name on the
	// first page of results. It is reasonable to assume however that the results
	// are in order. Still, even if that is the case, we may have trouble with
	// directories since there are characters before '/' that could be in a path
	// name. Perhaps we should be sending a separate request for fullName plus
	// the slash, as much as it pains me to do so.
	query := &storage.Query{
		Delimiter: string(dirSeparator),
		Prefix:    fullName,
	}

	objects, err := d.bucket.ListObjects(ctx, query)
	if err != nil {
		log.Println("bucket.ListObjects:", err)
		return nil, fuse.EIO
	}

	// Is there a matching file name?
	for _, o := range objects.Results {
		if o.Name == fullName {
			node := &file{
				bucket:     d.bucket,
				objectName: o.Name,
			}
			return node, nil
		}
	}

	// Is there a matching directory name?
	for _, p := range objects.Prefixes {
		if p == fullName+"/" {
			node := &dir{
				bucket:       d.bucket,
				objectPrefix: p,
			}

			return node, nil
		}
	}

	return nil, fuse.ENOENT
}

func (d *dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	ctx, cancel := withIntr(context.Background(), intr)
	defer cancel()

	return d.readDir(ctx)
}

func (d *dir) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	ctx, cancel := withIntr(context.Background(), intr)
	defer cancel()

	return d.lookup(ctx, name)
}

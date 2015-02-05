// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"log"
	"os"
	"path"

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
	authContext  context.Context
	bucketName   string
	objectPrefix string
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		// TODO(jacobsa): Expose ACLs from GCS?
		Mode: os.ModeDir | 0700,
	}
}

func (d *dir) readDir(ctx context.Context) (
	ents []fuse.Dirent, fuseErr fuse.Error) {
	log.Printf("ReadDir: [%s]/%s", d.bucketName, d.objectPrefix)

	// List repeatedly until there is no more to list.
	query := &storage.Query{
		Delimiter: string(dirSeparator),
		Prefix:    d.objectPrefix,
	}

	for query != nil {
		// Grab one set of results.
		objects, err := storage.ListObjects(ctx, d.bucketName, query)
		if err != nil {
			log.Println("storage.ListObjects:", err)
			return nil, fuse.EIO
		}

		// Extract objects as files.
		for _, o := range objects.Results {
			ents = append(ents, fuse.Dirent{
				Type: fuse.DT_File,
				Name: o.Name, // TODO(jacobsa): Strip prefix.
			})
		}

		// Extract prefixes as directories.
		for _, p := range objects.Prefixes {
			ents = append(ents, fuse.Dirent{
				Type: fuse.DT_Dir,
				Name: p, // TODO(jacobsa): Strip prefix.
			})
		}

		// Move on to the next set of results.
		query = objects.Next
	}

	return
}

func (d *dir) lookup(ctx context.Context, name string) (fs.Node, fuse.Error) {
	log.Printf("Lookup: ([%s]/%s) %s", d.bucketName, d.objectPrefix, name)

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

	objects, err := storage.ListObjects(ctx, d.bucketName, query)
	if err != nil {
		log.Println("storage.ListObjects:", err)
		return nil, fuse.EIO
	}

	// Is there a matching file name?
	for _, o := range objects.Results {
		if o.Name == fullName {
			log.Println("TODO(jacobsa): Handle files.")
			return nil, fuse.EIO
		}
	}

	// Is there a matching directory name?
	for _, p := range objects.Prefixes {
		if p == fullName+"/" {
			node := &dir{
				authContext:  d.authContext,
				bucketName:   d.bucketName,
				objectPrefix: p,
			}

			return node, nil
		}
	}

	return nil, fuse.ENOENT
}

func (d *dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	ctx, cancel := withIntr(d.authContext, intr)
	defer cancel()

	return d.readDir(ctx)
}

func (d *dir) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	ctx, cancel := withIntr(d.authContext, intr)
	defer cancel()

	return d.lookup(ctx, name)
}

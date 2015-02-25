// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcsfuse/timeutil"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// XXX: Comments
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

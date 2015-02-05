// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcs

import (
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// Bucket represents a GCS bucket, pre-bound with a bucket name and necessary
// authorization information.
type Bucket interface {
	Name() string

	// ListObjects lists objects in the bucket that meet certain criteria.
	ListObjects(ctx context.Context, query *storage.Query) (*storage.Objects, error)
}

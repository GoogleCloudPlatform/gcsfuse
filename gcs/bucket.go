// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcs

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

// Bucket represents a GCS bucket, pre-bound with a bucket name and necessary
// authorization information.
type Bucket interface {
	Name() string

	// ListObjects lists objects in the bucket that meet certain criteria.
	ListObjects(ctx context.Context, query *storage.Query) (*storage.Objects, error)
}

type bucket struct {
	projID string
	client *http.Client
	name   string
}

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) ListObjects(ctx context.Context, query *storage.Query) (*storage.Objects, error) {
	authContext := cloud.WithContext(ctx, b.projID, b.client)
	return storage.ListObjects(authContext, b.name, query)
}

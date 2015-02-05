// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcs

import (
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// Conn represents a connection to GCS, pre-bound with a project ID and
// information required for authorization.
type Conn interface {
	// Return a Bucket object representing the GCS bucket with the given name. No
	// immediate validation is performed.
	GetBucket(name string) Bucket
}

// Bucket represents a GCS bucket, pre-bound with a bucket name and necessary
// authorization information.
type Bucket interface {
	Name() string

	// ListObjects lists objects in the bucket that meet certain criteria.
	ListObjects(ctx context.Context, query *storage.Query) (*storage.Objects, error)
}

// Open a connection to GCS for the project with the given ID using the
// supplied HTTP client, which is assumed to handle authorization and
// authentication.
func OpenConn(projID string, c *http.Client) (Conn, error)

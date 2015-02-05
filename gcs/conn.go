// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcs

import "net/http"

// Conn represents a connection to GCS, pre-bound with a project ID and
// information required for authorization.
type Conn interface {
	// Return a Bucket object representing the GCS bucket with the given name. No
	// immediate validation is performed.
	GetBucket(name string) Bucket
}

// Open a connection to GCS for the project with the given ID using the
// supplied HTTP client, which is assumed to handle authorization and
// authentication.
func OpenConn(projID string, c *http.Client) (Conn, error) {
	return &conn{projID, c}, nil
}

type conn struct {
	projID string
	client *http.Client
}

func (c *conn) GetBucket(name string) Bucket {
	return &bucket{
		projID: c.projID,
		client: c.client,
		name:   name,
	}
}

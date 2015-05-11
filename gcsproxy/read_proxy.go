// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcsproxy

import (
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// A read-only view on a particular generation of an object in GCS. Reads may
// involve reading from a local cache.
//
// This type is not safe for concurrent access. The user must provide external
// synchronization around the methods where it is not otherwise noted.
type ReadProxy struct {
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Create a view on the given GCS object generation.
func NewReadProxy(
	leaser lease.FileLeaser,
	bucket gcs.Bucket,
	o *gcs.Object) (rp *ReadProxy) {
	// Set up a lease.ReadProxy.
	_ = lease.NewReadProxy(
		leaser,
		int64(o.Size),
		func(ctx context.Context) (rc io.ReadCloser, err error) {
			rc, err = getObjectContents(ctx, bucket, o)
			return
		})

	// Serve from that.
	panic("TODO")

	return
}

// Destroy any local file caches, putting the proxy into an indeterminate
// state. Should be used before dropping the final reference to the proxy.
func (rp *ReadProxy) Destroy() (err error) {
	panic("TODO")
}

// Return a read/write lease for the contents of the object. This implicitly
// destroys the proxy, which must not be used further.
func (rp *ReadProxy) Upgrade() (rwl lease.ReadWriteLease, err error) {
	panic("TODO")
}

// Return the size of the object generation in bytes.
func (rp *ReadProxy) Size() (size int64) {
	panic("TODO")
}

// Make a random access read into our view of the content. May block for
// network access.
//
// Guarantees that err != nil if n < len(buf)
func (rp *ReadProxy) ReadAt(
	ctx context.Context,
	buf []byte,
	offset int64) (n int, err error) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// For use with lease.NewReadProxy.
func getObjectContents(
	ctx context.Context,
	bucket gcs.Bucket,
	o *gcs.Object) (rc io.ReadCloser, err error) {
	req := &gcs.ReadObjectRequest{
		Name:       o.Name,
		Generation: o.Generation,
	}

	rc, err = bucket.NewReader(ctx, req)
	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	return
}

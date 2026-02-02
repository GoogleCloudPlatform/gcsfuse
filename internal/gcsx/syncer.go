// Copyright 2015 Google LLC
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

package gcsx

import (
	"fmt"
	"io"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"
)

// Syncer is safe for concurrent access.
type Syncer interface {
	// Given an object record and content that was originally derived from that
	// object's contents (and potentially modified):
	//
	// *   If the temp file has not been modified, return a nil new object.
	//
	// *   Otherwise, write out a new generation in the bucket (failing with
	//     *gcs.PreconditionError if the source generation is no longer current).
	SyncObject(
		ctx context.Context,
		fileName string,
		srcObject *gcs.Object,
		content TempFile) (o *gcs.Object, err error)
}

// NewSyncer creates a syncer that syncs into the supplied bucket.
//
// When the source object has been changed only by appending, and the source
// object's size is at least composeThreshold, we will "append" to it by writing
// out a temporary blob and composing it with the source object.
//
// Temporary blobs have names beginning with tmpObjectPrefix. We make an effort
// to delete them, but if we are interrupted for some reason we may not be able
// to do so. Therefore the user should arrange for garbage collection.
func NewSyncer(
	composeThreshold int64,
	chunkTransferTimeoutSecs int64,
	chunkRetryDeadlineSecs int64,
	tmpObjectPrefix string,
	bucket gcs.Bucket) (os Syncer) {
	// Create the object creators.
	fullCreator := &fullObjectCreator{
		bucket: bucket,
	}

	// Zonal buckets do not currently support Compose, so we always write objects
	// in their entirety.
	var composeCreator objectCreator
	if !bucket.BucketType().Zonal {
		composeCreator = newComposeObjectCreator(
			tmpObjectPrefix,
			bucket)
	}

	// And the syncer.
	os = newSyncer(composeThreshold, chunkTransferTimeoutSecs, chunkRetryDeadlineSecs, fullCreator, composeCreator)

	return
}

////////////////////////////////////////////////////////////////////////
// fullObjectCreator
////////////////////////////////////////////////////////////////////////

type fullObjectCreator struct {
	bucket gcs.Bucket
}

func (oc *fullObjectCreator) Create(
	ctx context.Context,
	objectName string,
	srcObject *gcs.Object,
	mtime *time.Time,
	chunkTransferTimeoutSecs int64,
	chunkRetryDeadlineSecs int64,
	r io.Reader) (o *gcs.Object, err error) {
	req := gcs.NewCreateObjectRequest(srcObject, objectName, mtime, chunkTransferTimeoutSecs, chunkRetryDeadlineSecs)
	req.Contents = r
	o, err = oc.bucket.CreateObject(ctx, req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %w", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// syncer
////////////////////////////////////////////////////////////////////////

// An implementation detail of syncer. See notes on newSyncer.
type objectCreator interface {
	Create(
		ctx context.Context,
		objectName string,
		srcObject *gcs.Object,
		mtime *time.Time,
		chunkTransferTimeoutSecs int64,
		chunkRetryDeadlineSecs int64,
		r io.Reader) (o *gcs.Object, err error)
}

// Create a syncer that stats the mutable content to see if it's dirty before
// calling through to one of two object creators if the content is dirty:
//
//   - fullCreator accepts the source object and the full contents with which it
//     should be overwritten.
//
//   - composeCreator accepts the source object and the contents that should be
//     "appended" to it.
//
// composeThreshold controls the source object length at which we consider it
// worthwhile to make the append optimization. It should be set to a value on
// the order of the bandwidth to GCS times three times the round trip latency
// to GCS (for a small create, a compose, and a delete).
func newSyncer(
	composeThreshold int64,
	chunkTransferTimeoutSecs int64,
	chunkRetryDeadlineSecs int64,
	fullCreator objectCreator,
	composeCreator objectCreator) (os Syncer) {
	os = &syncer{
		composeThreshold:         composeThreshold,
		chunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
		chunkRetryDeadlineSecs:   chunkRetryDeadlineSecs,
		fullCreator:              fullCreator,
		composeCreator:           composeCreator,
	}

	return
}

type syncer struct {
	composeThreshold         int64
	chunkTransferTimeoutSecs int64
	chunkRetryDeadlineSecs   int64
	fullCreator              objectCreator
	composeCreator           objectCreator
}

func (os *syncer) SyncObject(
	ctx context.Context,
	objectName string,
	srcObject *gcs.Object,
	content TempFile) (o *gcs.Object, err error) {
	// Stat the content.
	sr, err := content.Stat()
	if err != nil {
		err = fmt.Errorf("stat: %w", err)
		return
	}

	// Local files are not present on GCS, hence only fullCreator is
	// invoked and append flow is never triggered.
	if srcObject == nil {
		// Content.Stat() seeks the current position to end of file. Seek it back
		// to beginning of the file.
		_, err = content.Seek(0, 0)
		if err != nil {
			err = fmt.Errorf("error in seeking: %w", err)
			return
		}
		return os.fullCreator.Create(ctx, objectName, srcObject, sr.Mtime, os.chunkTransferTimeoutSecs, os.chunkRetryDeadlineSecs, content)
	}

	// Make sure the dirty threshold makes sense.
	srcSize := int64(srcObject.Size)
	if sr.DirtyThreshold > srcSize {
		err = fmt.Errorf(
			"stat returned weird DirtyThreshold field: %d vs. %d",
			sr.DirtyThreshold,
			srcObject.Size)

		return
	}

	// If the content hasn't been dirtied (i.e. it is the same size as the source
	// object, and no bytes within the source object have been dirtied), we're
	// done.
	if sr.Size == srcSize && sr.DirtyThreshold == srcSize {
		return
	}

	// Sanity check: the branch above should ensure that by the time we get here,
	// the stat result's mtime is non-nil.
	if sr.Mtime == nil {
		err = fmt.Errorf("wacky stat result: %#v", sr)
		return
	}

	// Otherwise, we need to create a new generation. If the source object is
	// long enough, hasn't been dirtied, and has a low enough component count,
	// then we can make the optimization of not rewriting its contents.
	if os.composeCreator != nil && srcSize >= os.composeThreshold &&
		sr.DirtyThreshold == srcSize &&
		srcObject.ComponentCount < gcs.MaxComponentCount {
		_, err = content.Seek(srcSize, 0)
		if err != nil {
			err = fmt.Errorf("seek: %w", err)
			return
		}

		o, err = os.composeCreator.Create(ctx, objectName, srcObject, sr.Mtime, os.chunkTransferTimeoutSecs, os.chunkRetryDeadlineSecs, content)
	} else {
		_, err = content.Seek(0, 0)
		if err != nil {
			err = fmt.Errorf("seek: %w", err)
			return
		}

		o, err = os.fullCreator.Create(ctx, objectName, srcObject, sr.Mtime, os.chunkTransferTimeoutSecs, os.chunkRetryDeadlineSecs, content)
	}

	// Deal with errors.
	if err != nil {
		err = fmt.Errorf("create: %w", err)
		return
	}

	return
}

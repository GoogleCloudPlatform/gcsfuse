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

package gcsx

import (
	"fmt"
	"io"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// MtimeMetadataKey objects are created by Syncer.SyncObject and contain a
// metadata field with this key and with a UTC mtime in the format defined
// by time.RFC3339Nano.
const MtimeMetadataKey = "gcsfuse_mtime"

// Syncer is safe for concurrent access.
type Syncer interface {
	// Given an object record and content that was originally derived from that
	// object's contents (and potentially modified):
	//
	// *   If the temp file has not been modified, return a nil new object.
	//
	// *   Otherwise, write out a new generation in the bucket (failing with
	//     *gcs.PreconditionError if the source generation is no longer current).
	//
	// In the second case, the TempFile is destroyed. Otherwise, including when
	// this function fails, it is guaranteed to still be valid.
	SyncObject(
		ctx context.Context,
		srcObject *gcs.Object,
		content TempFile) (o *gcs.Object, err error)
}

// NewSyncer creates a syncer that syncs into the supplied bucket.
//
// When the source object has been changed only by appending, and the source
// object's size is at least appendThreshold, we will "append" to it by writing
// out a temporary blob and composing it with the source object.
//
// Temporary blobs have names beginning with tmpObjectPrefix. We make an effort
// to delete them, but if we are interrupted for some reason we may not be able
// to do so. Therefore the user should arrange for garbage collection.
func NewSyncer(
	appendThreshold int64,
	tmpObjectPrefix string,
	bucket gcs.Bucket) (os Syncer) {
	// Create the object creators.
	fullCreator := &fullObjectCreator{
		bucket: bucket,
	}

	appendCreator := newAppendObjectCreator(
		tmpObjectPrefix,
		bucket)

	// And the syncer.
	os = newSyncer(appendThreshold, fullCreator, appendCreator)

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
	srcObject *gcs.Object,
	mtime time.Time,
	r io.Reader) (o *gcs.Object, err error) {
	req := &gcs.CreateObjectRequest{
		Name:                       srcObject.Name,
		GenerationPrecondition:     &srcObject.Generation,
		MetaGenerationPrecondition: &srcObject.MetaGeneration,
		Contents:                   r,
		Metadata: map[string]string{
			MtimeMetadataKey: mtime.Format(time.RFC3339Nano),
		},
	}

	o, err = oc.bucket.CreateObject(ctx, req)
	if err != nil {
		// Don't mangle precondition errors.
		if _, ok := err.(*gcs.PreconditionError); ok {
			return
		}

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
		srcObject *gcs.Object,
		mtime time.Time,
		r io.Reader) (o *gcs.Object, err error)
}

// Create a syncer that stats the mutable content to see if it's dirty before
// calling through to one of two object creators if the content is dirty:
//
// *   fullCreator accepts the source object and the full contents with which it
//     should be overwritten.
//
// *   appendCreator accepts the source object and the contents that should be
//     "appended" to it.
//
// appendThreshold controls the source object length at which we consider it
// worthwhile to make the append optimization. It should be set to a value on
// the order of the bandwidth to GCS times three times the round trip latency
// to GCS (for a small create, a compose, and a delete).
func newSyncer(
	appendThreshold int64,
	fullCreator objectCreator,
	appendCreator objectCreator) (os Syncer) {
	os = &syncer{
		appendThreshold: appendThreshold,
		fullCreator:     fullCreator,
		appendCreator:   appendCreator,
	}

	return
}

type syncer struct {
	appendThreshold int64
	fullCreator     objectCreator
	appendCreator   objectCreator
}

func (os *syncer) SyncObject(
	ctx context.Context,
	srcObject *gcs.Object,
	content TempFile) (o *gcs.Object, err error) {
	// Stat the content.
	sr, err := content.Stat()
	if err != nil {
		err = fmt.Errorf("Stat: %w", err)
		return
	}

	// Make sure the dirty threshold makes sense.
	srcSize := int64(srcObject.Size)
	if sr.DirtyThreshold > srcSize {
		err = fmt.Errorf(
			"Stat returned weird DirtyThreshold field: %d vs. %d",
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
		err = fmt.Errorf("Wacky stat result: %#v", sr)
		return
	}

	// Canonicalize to UTC.
	mtime := sr.Mtime.UTC()

	// Otherwise, we need to create a new generation. If the source object is
	// long enough, hasn't been dirtied, and has a low enough component count,
	// then we can make the optimization of not rewriting its contents.
	if srcSize >= os.appendThreshold &&
		sr.DirtyThreshold == srcSize &&
		srcObject.ComponentCount < gcs.MaxComponentCount {
		_, err = content.Seek(srcSize, 0)
		if err != nil {
			err = fmt.Errorf("Seek: %w", err)
			return
		}

		o, err = os.appendCreator.Create(ctx, srcObject, mtime, content)
	} else {
		_, err = content.Seek(0, 0)
		if err != nil {
			err = fmt.Errorf("Seek: %w", err)
			return
		}

		o, err = os.fullCreator.Create(ctx, srcObject, mtime, content)
	}

	// Deal with errors.
	if err != nil {
		// Special case: don't mess with precondition errors.
		if _, ok := err.(*gcs.PreconditionError); ok {
			return
		}

		err = fmt.Errorf("Create: %w", err)
		return
	}

	// Destroy the temp file.
	content.Destroy()

	return
}

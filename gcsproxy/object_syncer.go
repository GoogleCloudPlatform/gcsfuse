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

	"github.com/googlecloudplatform/gcsfuse/mutable"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Safe for concurrent access.
type ObjectSyncer interface {
	// Given an object record and content that was originally derived from that
	// object's contents (and potentially modified):
	//
	// *   If the temp file has not been modified, return a nil new object.
	//
	// *   Otherwise, write out a new generation in the bucket (failing with
	//     *gcs.PreconditionError if the source generation is no longer current).
	//
	// In the second case, the mutable.TempFile is destroyed. Otherwise,
	// including when this function fails, it is guaranteed to still be valid.
	SyncObject(
		ctx context.Context,
		srcObject *gcs.Object,
		content mutable.TempFile) (o *gcs.Object, err error)
}

// Create an object syncer that syncs into the supplied bucket.
//
// When the source object has been changed only by appending, and the source
// object's size is at least appendThreshold, we will "append" to it by writing
// out a temporary blob and composing it with the source object.
//
// Temporary blobs have names beginning with tmpObjectPrefix. We make an effort
// to delete them, but if we are interrupted for some reason we may not be able
// to do so. Therefore the user should arrange for garbage collection.
func NewObjectSyncer(
	appendThreshold int64,
	tmpObjectPrefix string,
	bucket gcs.Bucket) (os ObjectSyncer) {
	// Create the object creators.
	fullCreator := &fullObjectCreator{
		bucket: bucket,
	}

	appendCreator := newAppendObjectCreator(
		tmpObjectPrefix,
		bucket)

	// And the object syncer.
	os = newObjectSyncer(appendThreshold, fullCreator, appendCreator)

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
	r io.Reader) (o *gcs.Object, err error) {
	req := &gcs.CreateObjectRequest{
		Name: srcObject.Name,
		GenerationPrecondition: &srcObject.Generation,
		Contents:               r,
	}

	o, err = oc.bucket.CreateObject(ctx, req)
	if err != nil {
		// Don't mangle precondition errors.
		if _, ok := err.(*gcs.PreconditionError); ok {
			return
		}

		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// objectSyncer
////////////////////////////////////////////////////////////////////////

// An implementation detail of objectSyncer. See notes on
// newObjectSyncer.
type objectCreator interface {
	Create(
		ctx context.Context,
		srcObject *gcs.Object,
		r io.Reader) (o *gcs.Object, err error)
}

// Create an object syncer that stats the mutable content to see if it's dirty
// before calling through to one of two object creators if the content is dirty:
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
func newObjectSyncer(
	appendThreshold int64,
	fullCreator objectCreator,
	appendCreator objectCreator) (os ObjectSyncer) {
	os = &objectSyncer{
		appendThreshold: appendThreshold,
		fullCreator:     fullCreator,
		appendCreator:   appendCreator,
	}

	return
}

type objectSyncer struct {
	appendThreshold int64
	fullCreator     objectCreator
	appendCreator   objectCreator
}

func (os *objectSyncer) SyncObject(
	ctx context.Context,
	srcObject *gcs.Object,
	content mutable.Content) (rl lease.ReadLease, o *gcs.Object, err error) {
	// Stat the content.
	sr, err := content.Stat(ctx)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
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

	// Otherwise, we need to create a new generation. If the source object is
	// long enough, hasn't been dirtied, and has a low enough component count,
	// then we can make the optimization of not rewriting its contents.
	if srcSize >= os.appendThreshold &&
		sr.DirtyThreshold == srcSize &&
		srcObject.ComponentCount < gcs.MaxComponentCount {
		o, err = os.appendCreator.Create(
			ctx,
			srcObject,
			&mutableContentReader{
				Ctx:     ctx,
				Content: content,
				Offset:  srcSize,
			})
	} else {
		o, err = os.fullCreator.Create(
			ctx,
			srcObject,
			&mutableContentReader{
				Ctx:     ctx,
				Content: content,
			})
	}

	// Deal with errors.
	if err != nil {
		// Special case: don't mess with precondition errors.
		if _, ok := err.(*gcs.PreconditionError); ok {
			return
		}

		err = fmt.Errorf("Create: %v", err)
		return
	}

	// Yank out the contents.
	rl = content.Release().Downgrade()

	return
}

////////////////////////////////////////////////////////////////////////
// mutableContentReader
////////////////////////////////////////////////////////////////////////

// An io.Reader that wraps a mutable.Content object, reading starting from a
// base offset.
type mutableContentReader struct {
	Ctx     context.Context
	Content mutable.Content
	Offset  int64
}

func (mcr *mutableContentReader) Read(p []byte) (n int, err error) {
	n, err = mcr.Content.ReadAt(mcr.Ctx, p, mcr.Offset)
	mcr.Offset += int64(n)
	return
}

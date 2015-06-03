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

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/mutable"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Safe for concurrent access.
type ObjectSyncer interface {
	// Given an object record and content that was originally derived from that
	// object's contents (and potentially modified):
	//
	// *   If the content has not been modified, return a nil read lease and a
	//     nil new object.
	//
	// *   Otherwise, write out a new generation in the bucket (failing with
	//     *gcs.PreconditionError if the source generation is no longer current)
	//     and return a read lease for that object's contents.
	//
	// In the second case, the mutable.Content is destroyed. Otherwise, including
	// when this function fails, it is guaranteed to still be valid.
	SyncObject(
		ctx context.Context,
		srcObject *gcs.Object,
		content mutable.Content) (rl lease.ReadLease, o *gcs.Object, err error)
}

// Create an object syncer that syncs into the supplied bucket.
func NewObjectSyncer(
	bucket gcs.Bucket) (os ObjectSyncer) {
	os = &objectSyncer{
		bucket: bucket,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type objectSyncer struct {
	bucket gcs.Bucket
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
	if sr.DirtyThreshold > int64(srcObject.Size) {
		err = fmt.Errorf(
			"Stat returned weird DirtyThreshold field: %d vs. %d",
			sr.DirtyThreshold,
			srcObject.Size)

		return
	}

	// If the content hasn't been dirtied (i.e. it is the same size as the source
	// object, and no bytes within the source object have been dirtied), we're
	// done.
	if sr.Size == int64(srcObject.Size) &&
		sr.DirtyThreshold == sr.Size {
		return
	}

	// Otherwise, we need to create a new generation.
	o, err = os.bucket.CreateObject(
		ctx,
		&gcs.CreateObjectRequest{
			Name: srcObject.Name,
			Contents: &mutableContentReader{
				Ctx:     ctx,
				Content: content,
			},
			GenerationPrecondition: &srcObject.Generation,
		})

	if err != nil {
		// Special case: don't mess with precondition errors.
		if _, ok := err.(*gcs.PreconditionError); ok {
			return
		}

		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Yank out the contents.
	rl = content.Release().Downgrade()

	return
}

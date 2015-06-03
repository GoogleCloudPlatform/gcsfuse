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
	panic("TODO")
}

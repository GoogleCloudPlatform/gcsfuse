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
	"errors"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Given an object record and content that was originally derived from that
// object's contents (and potentially modified):
//
// *   If the content has not been modified, return a nil read proxy and a nil
//     new object.
//
// *   Otherwise, write out a new generation in the bucket (failing with
//     *gcs.PreconditionError if the source generation is no longer current)
//     and return a read proxy for that object's contents.
//
// In the second case, the MutableContent is destroyed. Otherwise, including
// when this function fails, it is guaranteed to still be valid.
func Sync(
	ctx context.Context,
	srcObject *gcs.Object,
	content MutableContent,
	bucket gcs.Bucket) (
	newProxy lease.ReadProxy, newObject *gcs.Object, err error) {
	// Stat the content.
	sr, err := content.Stat(ctx)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	// Make sure the dirty threshold makes sense.
	if sr.DirtyThreshold > int64(srcObject.Size) {
		err = fmt.Errorf(
			"Weird DirtyThreshold field: %d vs. %d",
			sr.DirtyThreshold,
			srcObject.Size)

		return
	}

	// If the content hasn't been dirtied, we're done.
	if sr.DirtyThreshold == int64(srcObject.Size) {
		return
	}

	// Otherwise, we need to create a new generation.
	err = errors.New("TODO")

	return
}

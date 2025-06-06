// Copyright 2023 Google LLC
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

package storageutil

import (
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"
)

// Repeatedly call bucket.ListObjects until there is nothing further to list,
// returning all objects and collapsed runs encountered.
//
// May modify *req.
func ListAll(
	ctx context.Context,
	bucket gcs.Bucket,
	req *gcs.ListObjectsRequest) (
	minObjects []*gcs.MinObject,
	runs []string,
	err error) {
	for {
		// Grab one set of results.
		var listing *gcs.Listing
		if listing, err = bucket.ListObjects(ctx, req); err != nil {
			return
		}

		// Accumulate the results.
		minObjects = append(minObjects, listing.MinObjects...)
		runs = append(runs, listing.CollapsedRuns...)

		// Are we done?
		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	return
}

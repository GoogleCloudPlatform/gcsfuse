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

package gcsutil

import (
	"fmt"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// List objects in the supplied bucket whose name starts with the given prefix.
// Write them into the supplied channel in an undefined order.
func ListPrefix(
	ctx context.Context,
	bucket gcs.Bucket,
	prefix string,
	objects chan<- *gcs.Object) (err error) {
	req := &gcs.ListObjectsRequest{
		Prefix: prefix,
	}

	// List until we run out.
	for {
		// Fetch the next batch.
		var listing *gcs.Listing
		listing, err = bucket.ListObjects(ctx, req)
		if err != nil {
			err = fmt.Errorf("ListObjects: %v", err)
			return
		}

		// Pass on each object.
		for _, o := range listing.Objects {
			select {
			case objects <- o:

				// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		// Are we done?
		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	return
}

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
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"golang.org/x/net/context"
)

// Delete all objects from the supplied bucket. Results are undefined if the
// bucket is being concurrently updated.
func DeleteAllObjects(
	ctx context.Context,
	bucket gcs.Bucket) error {
	bundle := syncutil.NewBundle(ctx)

	// List all of the objects in the bucket.
	objects := make(chan *gcs.Object, 100)
	bundle.Add(func(ctx context.Context) error {
		defer close(objects)
		return ListPrefix(ctx, bucket, "", objects)
	})

	// Strip everything but the name.
	objectNames := make(chan string, 10e3)
	bundle.Add(func(ctx context.Context) (err error) {
		defer close(objectNames)
		for o := range objects {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return

			case objectNames <- o.Name:
			}
		}

		return
	})

	// Delete the objects in parallel.
	const parallelism = 64
	for i := 0; i < parallelism; i++ {
		bundle.Add(func(ctx context.Context) error {
			for objectName := range objectNames {
				err := bucket.DeleteObject(
					ctx,
					&gcs.DeleteObjectRequest{
						Name: objectName,
					})

				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	return bundle.Join()
}

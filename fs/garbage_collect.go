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

package fs

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"github.com/jacobsa/gcloud/syncutil"
)

const tmpObjectPrefix = ".gcsfuse_tmp/"

func garbageCollectOnce(
	ctx context.Context,
	bucket gcs.Bucket) (objectsDeleted uint64, err error) {
	const stalenessThreshold = 30 * time.Minute
	b := syncutil.NewBundle(ctx)

	// List all objects with the temporary prefix.
	objects := make(chan *gcs.Object, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(objects)
		err = gcsutil.ListPrefix(ctx, bucket, tmpObjectPrefix, objects)
		if err != nil {
			err = fmt.Errorf("ListPrefix: %v", err)
			return
		}

		return
	})

	// Filter to the names of objects that are stale.
	now := time.Now()
	staleNames := make(chan string, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(staleNames)
		for o := range objects {
			if now.Sub(o.Updated) < stalenessThreshold {
				continue
			}

			select {
			case <-ctx.Done():
				err = ctx.Err()
				return

			case staleNames <- o.Name:
			}
		}

		return
	})

	// Delete those objects.
	b.Add(func(ctx context.Context) (err error) {
		for name := range staleNames {
			err = bucket.DeleteObject(ctx, name)
			if err != nil {
				err = fmt.Errorf("DeleteObject(%q): %v", name, err)
				return
			}

			atomic.AddUint64(&objectsDeleted, 1)
		}

		return
	})

	err = b.Join()
	return
}

// Periodically delete stale temporary objects from the supplied bucket.
func garbageCollect(
	ctx context.Context,
	bucket gcs.Bucket) {
	const period = 10 * time.Minute
	for _ = range time.Tick(period) {
		log.Println("Starting a garbage collection run.")

		startTime := time.Now()
		objectsDeleted, err := garbageCollectOnce(ctx, bucket)

		if err != nil {
			log.Printf(
				"Garbage collection failed after deleting %d objects in %v, "+
					"with error: %v",
				objectsDeleted,
				time.Since(startTime),
				err)
		} else {
			log.Printf(
				"Garbage collection succeeded after deleted %d objects in %v.",
				objectsDeleted,
				time.Since(startTime))
		}
	}
}

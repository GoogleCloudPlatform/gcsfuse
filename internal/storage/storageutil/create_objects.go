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
	"golang.org/x/sync/errgroup"
)

// Create multiple objects with some parallelism, with contents according to
// the supplied map from name to contents.
func CreateObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	input map[string][]byte) (err error) {
	group, ctx := errgroup.WithContext(ctx)

	// Feed ObjectInfo records into a channel.
	type record struct {
		name     string
		contents []byte
	}

	recordChan := make(chan record, len(input))
	for name, contents := range input {
		recordChan <- record{name, contents}
	}

	close(recordChan)

	// Create the objects in parallel.
	const parallelism = 64
	for i := 0; i < parallelism; i++ {
		group.Go(func() (err error) {
			for r := range recordChan {
				_, err = CreateObject(
					ctx, bucket,
					r.name,
					r.contents)

				if err != nil {
					return
				}
			}

			return
		})
	}

	err = group.Wait()
	return
}

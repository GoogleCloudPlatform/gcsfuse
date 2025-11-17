// Copyright 2024 Google LLC
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
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"
)

// DeleteObject deletes an object in the given bucket with the given name.
func DeleteObject(
	ctx context.Context,
	bucket gcs.Bucket,
	name string) error {
	req := &gcs.DeleteObjectRequest{
		Name:       name,
		Generation: 0,
	}

	return bucket.DeleteObject(ctx, req)
}

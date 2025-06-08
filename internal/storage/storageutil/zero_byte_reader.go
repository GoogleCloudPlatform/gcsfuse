// Copyright 2025 Google LLC
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
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func GetObjectSizeFromZeroByteReader(ctx context.Context, bh *storage.BucketHandle, objectName string) (int64, error) {
	// Get object handle
	obj := bh.Object(objectName)

	// Create a new reader
	reader, err := obj.NewRangeReader(ctx, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to create reader: %w", err)
	}
	err = reader.Close()

	// Return the size
	return reader.Attrs.Size, err
}

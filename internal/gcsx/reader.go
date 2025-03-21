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

package gcsx

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// RandomReader is an object that knows how to read ranges within a particular
// generation of a particular GCS object. Optimised for (large) sequential reads.
//
// Not safe for concurrent access.
//
// TODO - (raj-prince) - Rename this with appropriate name as it also started
// fulfilling the responsibility of reading object's content from cache.
type Reader interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// ReadAt returns the data from the requested offset and upto the size of input
	// byte array. It either populates input array i.e., p or returns a different
	// byte array. In case input array is populated, the same array will be returned
	// as part of response. Hence the callers should use the byte array returned
	// as part of response always.
	ReadAt(ctx context.Context, p []byte, offset int64) (objectData readers.ObjectData, err error)

	// Return the record for the object to which the reader is bound.
	Object() (o *gcs.MinObject)

	// Clean up any resources associated with the reader, which must not be used
	// again.
	Destroy()
}

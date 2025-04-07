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

// Reader is the base interface for all logical readers.
type Reader interface {
	// CheckInvariants performs internal consistency checks on the reader state.
	CheckInvariants()

	// ReadAt reads data into the provided byte slice starting at the given offset.
	// Returns an ObjectData struct with the actual data read and size.
	ReadAt(ctx context.Context, p []byte, offset int64) (readers.ObjectData, error)

	// Destroy is called to release any resources held by the reader.
	Destroy()
}

// The ReadManager interface extends the base Reader interface with an Object method.
// This is generally used in higher-level components that need access to object metadata.
// File handle will contain a ReadManager instance and will handle read operations.
type ReadManager interface {
	Reader

	// Object returns the underlying GCS object metadata associated with the reader.
	Object() *gcs.MinObject
}

// GCSReader defines an interface for reading data from a GCS object at a specific offset.
// It is intended for lower-level interactions with GCS-based readers.
type GCSReader interface {
	// ReadAt reads data into the provided request buffer, starting from the specified offset and ending at the specified end offset.
	// It returns an ObjectData response containing the data read and any error encountered.
	ReadAt(ctx context.Context, req *readers.GCSReaderReq) (readers.ObjectData, error)
}

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
	"errors"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// FallbackToAnotherReader is returned when data could not be retrieved
// from the current reader, indicating that the caller should attempt to fall back
// to an alternative reader.
var FallbackToAnotherReader = errors.New("fallback to another reader is required")

// ReadRequest encapsulates the parameters for a read operation.
type ReadRequest struct {
	// Buffer is provided by jacobsa/fuse and should be filled with data from the object.
	Buffer []byte

	// Offset specifies the starting position in the object from where data should be read.
	// Note: This value should not be modified by any reader. It is used by the
	// read manager to fall back to the next reader and to record the read operation
	// correctly.
	Offset int64

	// ReadInfo contains metadata about the read pattern.
	ReadInfo
}

// GCSReaderRequest represents the request parameters needed to read a data from a GCS object.
type GCSReaderRequest struct {
	// Buffer is provided by jacobsa/fuse and should be filled with data from the object.
	Buffer []byte

	// Offset specifies the starting position in the object from where data should be read.
	Offset int64

	// This determines GCS range request.
	EndOffset int64

	// This parameter specifies whether the reader needs to be discarded for a new reader.
	ForceCreateReader bool

	// ReadInfo contains metadata about the read pattern.
	*ReadInfo
}

// ReadResponse represents the response returned as part of a ReadAt call.
// It includes the actual data read and its size.
type ReadResponse struct {
	// Data contains slices of bytes read from the object. This is populated when
	// the reader returns data directly from its internal buffers.
	Data [][]byte

	// Size indicates how many bytes were read into DataBuf.
	Size int

	// Callback is a function to be executed after the read operation is completed.
	Callback func()
}

type Reader interface {
	// CheckInvariants performs internal consistency checks on the reader state.
	CheckInvariants()

	// ReadAt attempts to read data from the object. Depending on the
	// implementation, it may either populate the provided buffer `p` directly
	// or return data as a slice of byte slices in the `Data` field of the
	// ReadResponse. To indicate that the operation should be handled by an
	// alternative reader, return the error FallbackToAnotherReader.
	// If an error occurs, the size in ReadResponse will be zero.
	ReadAt(ctx context.Context, req *ReadRequest) (*ReadResponse, error)

	// Destroy is called to release any resources held by the reader.
	Destroy()
}

// ReadManager is generally used in higher-level components that need access to object metadata.
// File handle will contain a ReadManager instance and will handle read operations.
type ReadManager interface {
	Reader

	// Object returns the underlying GCS object metadata associated with the reader.
	Object() *gcs.MinObject
}

// GCSReader defines an interface for reading data from a GCS object.
// This interface is intended for lower-level interactions with GCS readers.
type GCSReader interface {
	// ReadAt reads data into the `Buffer` field of the provided `GCSReaderRequest`,
	// starting from the specified offset and ending at the specified end offset.
	// It returns a `ReadResponse` indicating the number of bytes successfully read and any error encountered.
	ReadAt(ctx context.Context, req *GCSReaderRequest) (*ReadResponse, error)
}

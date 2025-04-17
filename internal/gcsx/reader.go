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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// Provides methods to read data at a specific offset
type GCSReader interface {
	ReadAt(ctx context.Context, req *GCSReaderReq) (objectData ObjectData, err error)
}

// Base reader interface without Object()
type Reader interface {
	CheckInvariants()
	ReadAt(ctx context.Context, p []byte, offset int64) (objectData *ObjectData, err error)
	Destroy()
}

// Extended reader that also needs Object() method
type ReadManager interface {
	Reader
	Object() (o *gcs.MinObject)
}

type FallbackToAnotherReaderError struct{}

func (e *FallbackToAnotherReaderError) Error() string {
	return "fallback to another reader is not allowed"
}

// Usage
var FallbackToAnotherReader = &FallbackToAnotherReaderError{}

type GCSReaderReq struct {
	Buffer      []byte
	Offset      int64
	EndPosition int64
}

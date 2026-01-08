// Copyright 2026 Google LLC
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
	"fmt"
	"sync/atomic"
)

// MrdSimpleReader is a reader that uses an MRD Instance to read data from a GCS object.
// This reader is simpler than the GCSReader as it doesn't have complex logic
// to switch between sequential and random read strategies.
type MrdSimpleReader struct {
	mrdInstanceInUse atomic.Bool
	mrdInstance      *MrdInstance
}

// NewMrdSimpleReader creates a new MrdSimpleReader that uses the provided
// MrdInstance to manage MRD connections.
func NewMrdSimpleReader(mrdInstance *MrdInstance) *MrdSimpleReader {
	return &MrdSimpleReader{
		mrdInstance: mrdInstance,
	}
}

// ReadAt reads data into the provided request buffer starting at the specified
// offset. It retrieves an available MRD entry and uses it to download the
// requested byte range.
func (msr *MrdSimpleReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	// If the destination buffer is empty, there's nothing to read.
	if len(req.Buffer) == 0 {
		return ReadResponse{}, nil
	}

	// mrdInstance is set to nil in Destroy which will be called only after all active Read operations
	// have finished. Hence, not taking RLock to access it.
	if msr.mrdInstance == nil {
		return ReadResponse{}, fmt.Errorf("MrdSimpleReader: mrdInstance is nil")
	}

	if msr.mrdInstanceInUse.CompareAndSwap(false, true) {
		msr.mrdInstance.IncrementRefCount()
	}

	n, err := msr.mrdInstance.Read(ctx, req.Buffer, req.Offset)
	return ReadResponse{Size: n}, err
}

// Destroy cleans up the resources used by the reader, primarily by destroying
// the associated MrdInstance. This should be called when the reader is no
// longer needed.
func (msr *MrdSimpleReader) Destroy() {
	// No need to take lock as Destroy will only be called when file handle is being released
	// and there will be no read calls at that point.
	if msr.mrdInstance != nil {
		msr.mrdInstanceInUse.Store(false)
		msr.mrdInstance.DecrementRefCount()
		msr.mrdInstance = nil
	}
}

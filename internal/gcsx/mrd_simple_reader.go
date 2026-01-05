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
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// MrdSimpleReader is a reader that uses an MRD Instance to read data from a GCS object.
// This reader is simpler than the full `randomReader` as it doesn't have complex logic
//
//	to switch between sequential and random read strategies.
type MrdSimpleReader struct {
	// mu protects the internal state of the reader, specifically access to mrdInstance.
	mu               sync.RWMutex
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

// getValidEntry handles the logic of obtaining a usable entry from the MRDInstance,
// including initialization and recreation of entries if they are in a bad state.
func (msr *MrdSimpleReader) getValidEntry() (*MRDEntry, error) {
	msr.mu.RLock()
	defer msr.mu.RUnlock()

	if msr.mrdInstance == nil {
		return nil, fmt.Errorf("MrdSimpleReader: mrdInstance is nil")
	}

	if msr.mrdInstanceInUse.CompareAndSwap(false, true) {
		msr.mrdInstance.IncrementRefCount()
	}

	// Attempt to get an entry.
	entry := msr.mrdInstance.GetMRDEntry()

	// If no entry is available, the pool might not be initialized.
	var err error
	if entry == nil {
		err = msr.mrdInstance.EnsureMrdInstance()
		if err != nil {
			return nil, err
		}
		// After initialization, get the next available entry.
		entry = msr.mrdInstance.GetMRDEntry()
	} else {
		// If an entry was retrieved, check if it's usable.
		entry.mu.RLock()
		needsRecreation := entry.mrd == nil || entry.mrd.Error() != nil
		entry.mu.RUnlock()

		if needsRecreation {
			err = msr.mrdInstance.RecreateMRDEntry(entry)
			if err != nil {
				return nil, err
			}
			// After recreation, get the next available entry.
			entry = msr.mrdInstance.GetMRDEntry()
		}
	}

	return entry, err
}

// ReadAt reads data into the provided request buffer starting at the specified
// offset. It retrieves an available MRD entry and uses it to download the
// requested byte range. If an MRD entry is not available or is in an error
// state, it attempts to create or recreate one.
func (msr *MrdSimpleReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	// If the destination buffer is empty, there's nothing to read.
	if len(req.Buffer) == 0 {
		return ReadResponse{}, nil
	}

	entry, err := msr.getValidEntry()
	if err != nil {
		return ReadResponse{}, err
	}
	if entry == nil {
		return ReadResponse{}, fmt.Errorf("MrdSimpleReader: failed to get a valid MRD entry")
	}

	// Prepare the buffer for the read operation.
	buffer := bytes.NewBuffer(req.Buffer)
	buffer.Reset()
	done := make(chan readResult, 1)

	// Local mutex for the callback closure to prevent race conditions on the 'done' channel.
	var cbMu sync.Mutex
	defer func() {
		cbMu.Lock()
		// The channel must be closed only once.
		if done != nil {
			close(done)
			done = nil
		}
		cbMu.Unlock()
	}()

	// Lock the entry to safely access its MRD instance.
	entry.mu.RLock()
	mrd := entry.mrd
	if mrd == nil {
		entry.mu.RUnlock()
		return ReadResponse{}, fmt.Errorf("MrdSimpleReader: mrd is nil")
	}
	// Add the read request to the MRD instance. The read will be performed
	// asynchronously, and the callback will be invoked upon completion.
	mrd.Add(buffer, req.Offset, int64(len(req.Buffer)), func(offsetAddCallback int64, bytesReadAddCallback int64, e error) {
		defer func() {
			cbMu.Lock()
			// Send the result to the 'done' channel if it's still open.
			if done != nil {
				done <- readResult{bytesRead: int(bytesReadAddCallback), err: e}
			}
			cbMu.Unlock()
		}()

		// Wrap non-EOF errors for better context.
		if e != nil && e != io.EOF {
			e = fmt.Errorf("error in Add call: %w", e)
		}
	})
	entry.mu.RUnlock()

	var bytesRead int

	// Wait for the read to complete or for the context to be cancelled.
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case res := <-done:
		bytesRead = res.bytesRead
		err = res.err
	}

	if err != nil {
		return ReadResponse{}, err
	}

	return ReadResponse{Size: bytesRead}, nil
}

// Destroy cleans up the resources used by the reader, primarily by destroying
// the associated MrdInstance. This should be called when the reader is no
// longer needed.
func (msr *MrdSimpleReader) Destroy() {
	msr.mu.Lock()
	defer msr.mu.Unlock()
	if msr.mrdInstance != nil {
		msr.mrdInstanceInUse.Store(false)
		msr.mrdInstance.DecrementRefCount()
		msr.mrdInstance = nil
	}
}

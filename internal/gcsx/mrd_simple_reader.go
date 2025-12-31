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
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

type MrdSimpleReader struct {
	mu          sync.RWMutex
	mrdInstance *MrdInstance
}

func NewMrdSimpleReader(mrdInstance *MrdInstance) *MrdSimpleReader {
	return &MrdSimpleReader{
		mrdInstance: mrdInstance,
	}
}

func (msr *MrdSimpleReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	if len(req.Buffer) == 0 {
		return ReadResponse{}, nil
	}

	msr.mu.RLock()
	if msr.mrdInstance == nil {
		msr.mu.RUnlock()
		return ReadResponse{}, fmt.Errorf("MrdSimpleReader: mrdInstance is nil")
	}

	entry := msr.mrdInstance.GetMRDEntry()
	if entry == nil {
		msr.mrdInstance.EnsureMrdInstance()
		entry = msr.mrdInstance.GetMRDEntry()
	} else {
		entry.mu.RLock()
		needsRecreation := entry.mrd == nil || entry.mrd.Error() != nil
		entry.mu.RUnlock()

		if needsRecreation {
			msr.mrdInstance.RecreateMRDEntry(entry)
			entry = msr.mrdInstance.GetMRDEntry()
		}
	}

	if entry == nil {
		msr.mu.RUnlock()
		return ReadResponse{}, fmt.Errorf("MrdSimpleReader: failed to get a valid MRD entry")
	}
	msr.mu.RUnlock()

	buffer := bytes.NewBuffer(req.Buffer)
	buffer.Reset()
	done := make(chan readResult, 1)

	// Local mutex for the callback closure
	var cbMu sync.Mutex
	defer func() {
		cbMu.Lock()
		close(done)
		done = nil
		cbMu.Unlock()
	}()

	entry.mu.RLock()
	mrd := entry.mrd
	// Todo: Need to take a RLock here to ensure, we wait on before closing this MRD from MrdInstance
	mrd.Add(buffer, req.Offset, int64(len(req.Buffer)), func(offsetAddCallback int64, bytesReadAddCallback int64, e error) {
		defer func() {
			cbMu.Lock()
			if done != nil {
				done <- readResult{bytesRead: int(bytesReadAddCallback), err: e}
			}
			cbMu.Unlock()
		}()

		if e != nil && e != io.EOF {
			logger.Errorf("error in Add call: %v", e)
		}
	})
	entry.mu.RUnlock()

	var bytesRead int
	var err error

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

func (msr *MrdSimpleReader) Destroy() {
	msr.mu.Lock()
	defer msr.mu.Unlock()
	msr.mrdInstance.Destroy()
	msr.mrdInstance = nil
}

func (msr *MrdSimpleReader) CheckInvariants() {
}

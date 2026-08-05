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

package kernel_readers

import (
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// KernelRangeReaderInstance manages a reference to the latest gcs.MinObject
// for a standard/regional bucket, providing a thread-safe way to update
// it when the object is synced or mutated on GCS.
type KernelRangeReaderInstance struct {
	mu     sync.RWMutex
	object *gcs.MinObject
}

// NewKernelRangeReaderInstance creates a new KernelRangeReaderInstance.
func NewKernelRangeReaderInstance(obj *gcs.MinObject) *KernelRangeReaderInstance {
	return &KernelRangeReaderInstance{
		object: obj,
	}
}

// SetMinObject thread-safely updates the stored gcs.MinObject.
func (ki *KernelRangeReaderInstance) SetMinObject(minObj *gcs.MinObject) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	ki.object = minObj
}

// GetMinObject thread-safely returns a copy of the stored gcs.MinObject.
func (ki *KernelRangeReaderInstance) GetMinObject() *gcs.MinObject {
	ki.mu.RLock()
	defer ki.mu.RUnlock()
	return ki.object
}

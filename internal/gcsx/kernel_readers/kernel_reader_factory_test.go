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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func TestNewKernelReader_Zonal(t *testing.T) {
	mockBucket := new(storage.TestifyMockBucket)
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})

	reader := NewKernelReader(mockBucket, nil, nil, nil)

	assert.NotNil(t, reader)
	assert.Equal(t, "KernelMRDReader", reader.ReaderName())
	mockBucket.AssertExpectations(t)
}

func TestNewKernelReader_Standard(t *testing.T) {
	mockBucket := new(storage.TestifyMockBucket)
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: false})

	reader := NewKernelReader(mockBucket, nil, nil, nil)

	assert.NotNil(t, reader)
	assert.Equal(t, "KernelRangeReader", reader.ReaderName())
	mockBucket.AssertExpectations(t)
}

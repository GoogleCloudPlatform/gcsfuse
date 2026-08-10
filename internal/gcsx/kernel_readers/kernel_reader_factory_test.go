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
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func TestNewKernelReader(t *testing.T) {
	bucketTypes := []struct {
		name       string
		bucketType gcs.BucketType
		isNil      bool
	}{
		{
			name:       "Zonal",
			bucketType: gcs.BucketType{Zonal: true},
		},
		{
			name:       "PirloEnabled",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesEnabled},
		},
		{
			name:       "PirloDisabled",
			bucketType: gcs.BucketType{Pirlo: gcs.PirloStateRapidWritesDisabled},
		},
		{
			name:       "Regional",
			bucketType: gcs.BucketType{},
		},
		{
			name:       "HierarchicalRegional",
			bucketType: gcs.BucketType{Hierarchical: true},
		},
		{
			name:  "NilBucket",
			isNil: true,
		},
	}

	protocols := []struct {
		name     string
		protocol cfg.Protocol
	}{
		{name: "GRPC", protocol: cfg.GRPC},
		{name: "HTTP1", protocol: cfg.HTTP1},
		{name: "HTTP2", protocol: cfg.HTTP2},
		{name: "EmptyProtocol", protocol: ""},
	}

	for _, bt := range bucketTypes {
		for _, proto := range protocols {
			testName := fmt.Sprintf("%s_%s", bt.name, proto.name)
			t.Run(testName, func(t *testing.T) {
				var mockBucket gcs.Bucket
				if !bt.isNil {
					mb := new(storage.TestifyMockBucket)
					mb.On("BucketType").Return(bt.bucketType)
					mockBucket = mb
				}

				reader := NewKernelReader(mockBucket, nil, nil, nil, proto.protocol)

				assert.NotNil(t, reader)
				isRapid := !bt.isNil && bt.bucketType.IsRapid()
				if isRapid || proto.protocol == cfg.GRPC {
					assert.Equal(t, "KernelMRDReader", reader.ReaderName())
				} else {
					assert.Equal(t, "KernelRangeReader", reader.ReaderName())
				}

				if mb, ok := mockBucket.(*storage.TestifyMockBucket); ok {
					mb.AssertExpectations(t)
				}
			})
		}
	}
}

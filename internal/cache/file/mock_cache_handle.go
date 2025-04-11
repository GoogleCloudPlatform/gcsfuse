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

package file

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
)

// MockCacheHandle is a testify mock for the CacheHandle interface
type MockCacheHandle struct {
	mock.Mock
}

func (m *MockCacheHandle) Read(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, offset int64, dst []byte) (int, bool, error) {
	args := m.Called(ctx, bucket, object, offset, dst)
	return args.Int(0), args.Bool(1), args.Error(2)
}

func (m *MockCacheHandle) IsSequential(currentOffset int64) bool {
	args := m.Called(currentOffset)
	return args.Bool(0)
}

func (m *MockCacheHandle) Close() error {
	args := m.Called()
	return args.Error(0)
}

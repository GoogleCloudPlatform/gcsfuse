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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
)

// MockFileCacheHandler is a testify mock for the FileCacheHandler interface
type MockFileCacheHandler struct {
	mock.Mock
}

func (m *MockFileCacheHandler) GetCacheHandle(object *gcs.MinObject, bucket gcs.Bucket, cacheForRangeRead bool, initialOffset int64) (CacheHandleInterface, error) {
	args := m.Called(object, bucket, cacheForRangeRead, initialOffset)
	if ch, ok := args.Get(0).(CacheHandleInterface); ok {
		return ch, nil
	}
	return nil, args.Error(1)
}

func (m *MockFileCacheHandler) InvalidateCache(objectName string, bucketName string) error {
	args := m.Called(objectName, bucketName)
	return args.Error(0)
}

func (m *MockFileCacheHandler) Destroy() error {
	args := m.Called()
	return args.Error(0)
}

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

	"github.com/stretchr/testify/mock"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
)

type MockRandomReader struct {
	RandomReader
	mock.Mock
}

func (m *MockRandomReader) ReadAt(ctx context.Context, dst []byte, offset int64) (ObjectData, error) {
	args := m.Called(ctx, dst, offset)
	return args.Get(0).(ObjectData), args.Error(1)
}

func (m *MockRandomReader) Object() *gcs.MinObject {
	args := m.Called()
	return args.Get(0).(*gcs.MinObject)
}

func (m *MockRandomReader) Destroy() {
	m.Called()
}

func (m *MockRandomReader) CheckInvariants() {
	m.Called()
}

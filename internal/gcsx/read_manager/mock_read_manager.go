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

package read_manager

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
)

type MockReadManager struct {
	gcsx.ReadManager
	mock.Mock
}

func (m *MockReadManager) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (*gcsx.ReadResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcsx.ReadResponse), args.Error(1)
}

func (m *MockReadManager) Object() *gcs.MinObject {
	args := m.Called()
	return args.Get(0).(*gcs.MinObject)
}

func (m *MockReadManager) Destroy() {
	m.Called()
}

func (m *MockReadManager) CheckInvariants() {
	m.Called()
}

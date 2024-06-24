// Copyright 2024 Google Inc. All Rights Reserved.
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

package storage

import (
	"context"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
)

// MockStorageControlClient creates a mock version of the StorageControlClient.
type MockStorageControlClient struct {
	StorageControlClient
	mock.Mock
}

// Implement the GetStorageLayout method for the mock.
func (m *MockStorageControlClient) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*controlpb.StorageLayout), args.Error(1)
}

// Implement the DeleteFolder method for the mock.
func (m *MockStorageControlClient) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) error {
	args := m.Called(ctx, req, opts)
	return args.Error(0)
}

func (m *MockStorageControlClient) GetFolder(ctx context.Context,
	req *controlpb.GetFolderRequest,
	opts ...gax.CallOption) (*controlpb.Folder, error) {
	args := m.Called(ctx, req, opts)

	// Needed to assert folder in only those cases where folder is present
	if folder, ok := args.Get(0).(*controlpb.Folder); ok {
		return folder, nil
	}

	return nil, args.Error(1)
}

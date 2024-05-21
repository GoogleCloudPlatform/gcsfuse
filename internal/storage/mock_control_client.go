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
	 mock.Mock
}

// Implement the GetStorageLayout method for the mock.
func (m *MockStorageControlClient) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption)(*controlpb.StorageLayout, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*controlpb.StorageLayout), args.Error(1)
}

//// MockStorageControlClientInterface is a mock of StorageControlClientInterface interface.
//type MockStorageControlClientInterface struct {
//	ctrl     *gomock.Controller
//	recorder *MockStorageControlClientInterfaceMockRecorder
//}
//
//// MockStorageControlClientInterfaceMockRecorder is the mock recorder for MockStorageControlClientInterface.
//type MockStorageControlClientInterfaceMockRecorder struct {
//	mock *MockStorageControlClientInterface
//}
//
//// NewMockStorageControlClientInterface creates a new mock instance.
//func NewMockStorageControlClientInterface(ctrl *gomock.Controller) *MockStorageControlClientInterface {
//	mock := &MockStorageControlClientInterface{ctrl: ctrl}
//	mock.recorder = &MockStorageControlClientInterfaceMockRecorder{mock}
//	return mock
//}
//
//// EXPECT returns an object that allows the caller to indicate expected use.
//func (m *MockStorageControlClientInterface) EXPECT() *MockStorageControlClientInterfaceMockRecorder {
//	return m.recorder
//}
//
//// GetStorageLayout mocks base method.
//func (m *MockStorageControlClientInterface) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
//	m.ctrl.T.Helper()
//	ret := m.ctrl.Call(m, "GetStorageLayout", ctx, req, opts)
//	ret0, _ := ret[0].(*controlpb.StorageLayout)
//	ret1, _ := ret[1].(error)
//	return ret0, ret1
//}
//
//// GetStorageLayout indicates an expected call of GetStorageLayout.
//func (mr *MockStorageControlClientInterfaceMockRecorder) GetStorageLayout(ctx, req interface{}, opts ...interface{}) *gomock.Call {
//	mr.mock.ctrl.T.Helper()
//	varargs := append([]interface{}{ctx, req}, opts...)
//	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageLayout", reflect.TypeOf((*MockStorageControlClientInterface)(nil).GetStorageLayout), varargs...)
//}
//
//
//// MockStorageControlClient creates a mock version of the StorageControlClient.
////type MockStorageControlClient struct {
////	mock.Mock
////}
////
////// Implement the GetStorageLayout method for the mock.
////func (m *MockStorageControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...interface{}) (*controlpb.StorageLayout, error) {
////	args := m.Called(ctx, req, opts)
////
////	return args.Get(0).(*controlpb.StorageLayout), args.Error(1)
////}
////
////
////func (m *MockStorageControlClient) Close() error {
////	args := m.Called()
////	return args.Get(0).(error)
////}
////
////func (m *MockStorageControlClient) setGoogleClientInfo(...string) {}
////
////func (m *MockStorageControlClient) Connection() *grpc.ClientConn {
////	args := m.Called()
////	return args.Get(0).(*grpc.ClientConn)
////}
////
////func (m *MockStorageControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts gax.CallOption) (*controlpb.Folder, error) {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*controlpb.Folder), args.Error(1)
////}
////
////func (m *MockStorageControlClient) DeleteFolder(ctx context.Context, req *controlpb.DeleteFolderRequest, opts gax.CallOption) error {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(error)
////}
////func (m *MockStorageControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts gax.CallOption) (*controlpb.Folder, error) {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*controlpb.Folder), args.Error(1)
////}
////func (m *MockStorageControlClient) ListFolders(ctx context.Context,req *controlpb.ListFoldersRequest, opts gax.CallOption) *control.FolderIterator {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*control.FolderIterator)
////}
////func (m *MockStorageControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts gax.CallOption) (*control.RenameFolderOperation, error) {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*control.RenameFolderOperation), args.Error(1)
////}
////
////func (m *MockStorageControlClient) RenameFolderOperation(name string) *control.RenameFolderOperation {
////	args := m.Called(name)
////	return args.Get(0).(*control.RenameFolderOperation)
////}
////
////func (m *MockStorageControlClient) CreateManagedFolder(ctx context.Context, req *controlpb.CreateManagedFolderRequest, opts gax.CallOption) (*controlpb.ManagedFolder, error) {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*controlpb.ManagedFolder), args.Error(1)
////}
////
////func (m *MockStorageControlClient) DeleteManagedFolder(ctx context.Context, req *controlpb.DeleteManagedFolderRequest, opts gax.CallOption) error {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(error)
////}
////
////func (m *MockStorageControlClient) GetManagedFolder(ctx context.Context, req *controlpb.GetManagedFolderRequest, opts gax.CallOption) (*controlpb.ManagedFolder, error) {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*controlpb.ManagedFolder), args.Error(1)
////}
////
////func (m *MockStorageControlClient) ListManagedFolders(ctx context.Context, req *controlpb.ListManagedFoldersRequest, opts gax.CallOption) *control.ManagedFolderIterator {
////	args := m.Called(ctx, req, opts)
////	return args.Get(0).(*control.ManagedFolderIterator)
////}
////
////// Mock FetchAndSetBucketType function
////func mockFetchAndSetBucketType(mockClient *MockStorageControlClient) gcs.BucketType {
////	if mockClient == nil {
////		return gcs.NonHierarchical
////	}
////
////	stoargeLayout, err := mockClient.GetStorageLayout(context.Background(), &controlpb.GetStorageLayoutRequest{
////		Name: "projects/_/buckets/" + TestBucketName + "/storageLayout",
////	}, nil...)
////
////	if err != nil {
////		return gcs.Unknown
////	}
////
////	if stoargeLayout.GetHierarchicalNamespace() != nil && stoargeLayout.GetHierarchicalNamespace().Enabled {
////		return gcs.Hierarchical
////	}
////
////	return gcs.NonHierarchical
////}

package storage

import (
	"context"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
)

// TODO: Rename to mock bucket once deprecated ogle mock bucket is removed from all usages in unit tests
type TestifyMockBucket struct {
	mock.Mock
}

func (m *TestifyMockBucket) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *TestifyMockBucket) BucketType() gcs.BucketType {
	args := m.Called()
	return args.Get(0).(gcs.BucketType)
}

func (m *TestifyMockBucket) NewReader(ctx context.Context, req *gcs.ReadObjectRequest) (io.ReadCloser, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *TestifyMockBucket) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.Object), args.Error(1)
}

func (m *TestifyMockBucket) CopyObject(ctx context.Context, req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.Object), args.Error(1)
}

func (m *TestifyMockBucket) ComposeObjects(ctx context.Context, req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.Object), args.Error(1)
}

func (m *TestifyMockBucket) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.MinObject), args.Get(1).(*gcs.ExtendedObjectAttributes), args.Error(2)
}

func (m *TestifyMockBucket) ListObjects(ctx context.Context, req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.Listing), args.Error(1)
}

func (m *TestifyMockBucket) UpdateObject(ctx context.Context, req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*gcs.Object), args.Error(1)
}

func (m *TestifyMockBucket) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *TestifyMockBucket) DeleteFolder(ctx context.Context, folderName string) error {
	args := m.Called(ctx, folderName)
	return args.Error(0)
}

func (m *TestifyMockBucket) GetFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	args := m.Called(ctx, folderName)
	if args.Get(0) != nil {
		return args.Get(0).(*gcs.Folder), nil
	}
	return nil, args.Error(1)
}

func (m *TestifyMockBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error) {
	args := m.Called(ctx, folderName, destinationFolderId)
	return args.Get(0).(*gcs.Folder), args.Error(1)
}

func (m *TestifyMockBucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	args := m.Called(ctx, folderName)
	return args.Get(0).(*gcs.Folder), args.Error(1)
}

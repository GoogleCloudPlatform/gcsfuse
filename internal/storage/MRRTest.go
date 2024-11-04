package storage

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/mock"
)

type mockStorageBucketHandle struct {
	*storage.BucketHandle // Embedding the original bucket handle
	mock.Mock
}

type mockObjectHandle struct {
	mock.Mock
}

func (mbh *mockStorageBucketHandle) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	panic("unimplemented")
}

func (mbh *mockStorageBucketHandle) Delete(ctx context.Context) error {
	panic("unimplemented")
}
func (mbh *mockStorageBucketHandle) BucketName() string {
	panic("unimplemented")
}

func (mbh *mockStorageBucketHandle) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	panic("unimplemented")
}

func (mbh *mockStorageBucketHandle) Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	panic("unimplemented")
}

func (mbh *mockStorageBucketHandle) Objects(ctx context.Context, q *storage.Query) *storage.ObjectIterator {
	panic("unimplemented")
}

func (m *mockStorageBucketHandle) Object(name string) *storage.ObjectHandle {
	args := m.Called()
	moh := args.Get(0).(storage.ObjectHandle)
	return &moh
}

func TestForNewMRR(t *testing.T) {

	mockStorageBucketHandle := new(mockStorageBucketHandle)
	mockObjectHandle := new(mockObjectHandle)

	// Set up the mock behavior
	mockStorageBucketHandle.On("Object", "objName").Return(&mockObjectHandle)

	bh := &bucketHandle{
		bucket:        mockStorageBucketHandle,
		bucketName:    "",
		controlClient: nil,
	}
	// Test any method of bh
	bh.NewReader()

}

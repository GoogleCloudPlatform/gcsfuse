package storage

import (
	"context"

	"cloud.google.com/go/storage"
)

type BucketHandleInterface interface {
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error

	Delete(ctx context.Context) error

	BucketName() string

	Attrs(ctx context.Context) (*storage.BucketAttrs, error)

	Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error)

	Objects(ctx context.Context, q *storage.Query) *storage.ObjectIterator

	Object(name string) *storage.ObjectHandle
}

type StorageBucketHandleWrapper struct {
	bucket *storage.BucketHandle
}

func (sbh *StorageBucketHandleWrapper) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	return sbh.Create(ctx, projectID, attrs)
}

func (sbh *StorageBucketHandleWrapper) Delete(ctx context.Context) error {
	return sbh.Delete(ctx)
}

func (sbh *StorageBucketHandleWrapper) BucketName() string {
	return sbh.BucketName()
}

func (sbh *StorageBucketHandleWrapper) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	return sbh.Attrs(ctx)
}

func (sbh *StorageBucketHandleWrapper) Update(ctx context.Context, uattrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	return sbh.Update(ctx, uattrs)
}

func (sbh *StorageBucketHandleWrapper) Objects(ctx context.Context, q *storage.Query) *storage.ObjectIterator {
	return sbh.Objects(ctx, q)
}

func (sbh *StorageBucketHandleWrapper) Object(name string) *storage.ObjectHandle {
	return sbh.Object(name)
}

func NewStorageBucketHandleWrapper(bucket *storage.BucketHandle) *StorageBucketHandleWrapper {
	return &StorageBucketHandleWrapper{
		bucket,
	}
}

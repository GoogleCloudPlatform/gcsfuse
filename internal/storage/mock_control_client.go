package storage

import (
	"context"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/mock"
)

// MockStorageControlClient creates a mock version of the StorageControlClient.
type MockStorageControlClient struct {
	mock.Mock
}

// Implement the GetStorageLayout method for the mock.
func (m *MockStorageControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...interface{}) (*controlpb.StorageLayout, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*controlpb.StorageLayout), args.Error(1)
}

// Mock getBucketType function
func mockGetBucketType(controlClient *MockStorageControlClient, bucketName string) gcs.BucketTypes {
	if controlClient == nil {
		return gcs.NonHierarchical
	}

	stoargeLayout, err := controlClient.GetStorageLayout(context.Background(), &controlpb.GetStorageLayoutRequest{
		Name: "projects/_/buckets/" + bucketName + "/storageLayout",
	}, nil...)

	if err != nil {
		return gcs.Unknown
	}

	if stoargeLayout.GetHierarchicalNamespace() != nil && stoargeLayout.GetHierarchicalNamespace().Enabled {
		return gcs.Hierarchical
	}

	return gcs.NonHierarchical
}

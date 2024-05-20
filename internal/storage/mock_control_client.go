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

// Mock FetchAndSetBucketType function
func mockFetchAndSetBucketType(mockClient *MockStorageControlClient) gcs.BucketType {
	if mockClient == nil {
		return gcs.NonHierarchical
	}

	stoargeLayout, err := mockClient.GetStorageLayout(context.Background(), &controlpb.GetStorageLayoutRequest{
		Name: "projects/_/buckets/" + TestBucketName + "/storageLayout",
	}, nil...)

	if err != nil {
		return gcs.Unknown
	}

	if stoargeLayout.GetHierarchicalNamespace() != nil && stoargeLayout.GetHierarchicalNamespace().Enabled {
		return gcs.Hierarchical
	}

	return gcs.NonHierarchical
}

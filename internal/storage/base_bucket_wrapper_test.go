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

package storage

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func TestBaseBucketWrapper_Name(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	mockBucket.On("Name").Return("test-bucket")
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	result := wrapper.Name()
	
	assert.Equal(t, "test-bucket", result)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_BucketType(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	expectedType := gcs.BucketType{Hierarchical: true, Zonal: false}
	mockBucket.On("BucketType").Return(expectedType)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	result := wrapper.BucketType()
	
	assert.Equal(t, expectedType, result)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_NewReaderWithReadHandle(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.ReadObjectRequest{Name: "test-object"}
	
	mockBucket.On("NewReaderWithReadHandle", ctx, req).Return(nil, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	_, err := wrapper.NewReaderWithReadHandle(ctx, req)
	
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_NewMultiRangeDownloader(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.MultiRangeDownloaderRequest{Name: "test-object"}
	
	mockBucket.On("NewMultiRangeDownloader", ctx, req).Return(nil, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	_, err := wrapper.NewMultiRangeDownloader(ctx, req)
	
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_CreateObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.CreateObjectRequest{Name: "test-object"}
	expectedObj := &gcs.Object{Name: "test-object"}
	
	mockBucket.On("CreateObject", ctx, req).Return(expectedObj, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	result, err := wrapper.CreateObject(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, result)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_DeleteObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.DeleteObjectRequest{Name: "test-object"}
	
	mockBucket.On("DeleteObject", ctx, req).Return(nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	err := wrapper.DeleteObject(ctx, req)
	
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_StatObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.StatObjectRequest{Name: "test-object"}
	expectedMinObj := &gcs.MinObject{Name: "test-object"}
	expectedExtAttrs := &gcs.ExtendedObjectAttributes{}
	
	mockBucket.On("StatObject", ctx, req).Return(expectedMinObj, expectedExtAttrs, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	minObj, extAttrs, err := wrapper.StatObject(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedMinObj, minObj)
	assert.Equal(t, expectedExtAttrs, extAttrs)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_ListObjects(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.ListObjectsRequest{Prefix: "test-"}
	expectedListing := &gcs.Listing{}
	
	mockBucket.On("ListObjects", ctx, req).Return(expectedListing, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	listing, err := wrapper.ListObjects(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedListing, listing)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_DeleteFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder/"
	
	mockBucket.On("DeleteFolder", ctx, folderName).Return(nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	err := wrapper.DeleteFolder(ctx, folderName)
	
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_GetFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder/"
	expectedFolder := &gcs.Folder{Name: folderName}
	
	mockBucket.On("GetFolder", ctx, folderName).Return(expectedFolder, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	folder, err := wrapper.GetFolder(ctx, folderName)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_CreateFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder/"
	expectedFolder := &gcs.Folder{Name: folderName}
	
	mockBucket.On("CreateFolder", ctx, folderName).Return(expectedFolder, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	folder, err := wrapper.CreateFolder(ctx, folderName)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_RenameFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "old-folder/"
	destFolderId := "new-folder-id"
	expectedFolder := &gcs.Folder{Name: "new-folder/"}
	
	mockBucket.On("RenameFolder", ctx, folderName, destFolderId).Return(expectedFolder, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	folder, err := wrapper.RenameFolder(ctx, folderName, destFolderId)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_GCSName(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	minObj := &gcs.MinObject{Name: "test-object"}
	
	mockBucket.On("GCSName", minObj).Return("test-object")
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	result := wrapper.GCSName(minObj)
	
	assert.Equal(t, "test-object", result)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_CopyObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.CopyObjectRequest{SrcName: "src", DstName: "dst"}
	expectedObj := &gcs.Object{Name: "dst"}
	
	mockBucket.On("CopyObject", ctx, req).Return(expectedObj, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	obj, err := wrapper.CopyObject(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_ComposeObjects(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.ComposeObjectsRequest{DstName: "composed"}
	expectedObj := &gcs.Object{Name: "composed"}
	
	mockBucket.On("ComposeObjects", ctx, req).Return(expectedObj, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	obj, err := wrapper.ComposeObjects(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_UpdateObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.UpdateObjectRequest{Name: "test-object"}
	expectedObj := &gcs.Object{Name: "test-object"}
	
	mockBucket.On("UpdateObject", ctx, req).Return(expectedObj, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	obj, err := wrapper.UpdateObject(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestBaseBucketWrapper_MoveObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.MoveObjectRequest{SrcName: "src", DstName: "dst"}
	expectedObj := &gcs.Object{Name: "dst"}
	
	mockBucket.On("MoveObject", ctx, req).Return(expectedObj, nil)
	
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	obj, err := wrapper.MoveObject(ctx, req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

// TestBaseBucketWrapper_InterfaceCompliance verifies that baseBucketWrapper
// implements the gcs.Bucket interface at runtime.
func TestBaseBucketWrapper_InterfaceCompliance(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	wrapper := &baseBucketWrapper{wrapped: mockBucket}
	
	// This will fail at compile time if baseBucketWrapper doesn't implement gcs.Bucket
	var _ gcs.Bucket = wrapper
	
	// Also verify at runtime
	_, ok := interface{}(wrapper).(gcs.Bucket)
	assert.True(t, ok, "baseBucketWrapper should implement gcs.Bucket interface")
}

// TestBaseBucketWrapper_EmbeddingPattern demonstrates how to use the wrapper
// in a custom bucket implementation.
func TestBaseBucketWrapper_EmbeddingPattern(t *testing.T) {
	// Create a custom bucket that embeds baseBucketWrapper
	type customBucket struct {
		baseBucketWrapper
		customField string
	}
	
	mockBucket := &TestifyMockBucket{}
	mockBucket.On("Name").Return("test-bucket")
	
	custom := &customBucket{
		baseBucketWrapper: baseBucketWrapper{wrapped: mockBucket},
		customField:       "custom-value",
	}
	
	// Verify that inherited methods work
	assert.Equal(t, "test-bucket", custom.Name())
	assert.Equal(t, "custom-value", custom.customField)
	
	// Verify it implements the interface
	var _ gcs.Bucket = custom
	
	mockBucket.AssertExpectations(t)
}

// customBucketWithOverride demonstrates overriding specific methods
type customBucketWithOverride struct {
	baseBucketWrapper
	nameOverride string
}

// Override only the Name method
func (cb *customBucketWithOverride) Name() string {
	return cb.nameOverride
}

// TestBaseBucketWrapper_SelectiveOverride demonstrates overriding specific methods
// while keeping others delegated.
func TestBaseBucketWrapper_SelectiveOverride(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	mockBucket.On("BucketType").Return(gcs.BucketType{})
	
	custom := &customBucketWithOverride{
		baseBucketWrapper: baseBucketWrapper{wrapped: mockBucket},
		nameOverride:      "overridden-name",
	}
	
	// Name is overridden
	assert.Equal(t, "overridden-name", custom.Name())
	
	// BucketType is still delegated to wrapped bucket
	custom.BucketType()
	mockBucket.AssertExpectations(t)
}

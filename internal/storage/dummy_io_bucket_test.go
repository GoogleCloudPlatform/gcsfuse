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
	"errors"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

func TestNewDummyIOBucket(t *testing.T) {
	// Test that NewDummyIOBucket doesn't panic and returns a non-nil bucket
	bucket := NewDummyIOBucket(nil)
	assert.NotNil(t, bucket)
}

func TestDummyIOBucket_Name(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	mockBucket.On("Name").Return("test-bucket")

	result := mockBucket.Name()
	assert.Equal(t, "test-bucket", result)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_BucketType(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	expectedType := gcs.BucketType{Hierarchical: false, Zonal: false}
	mockBucket.On("BucketType").Return(expectedType)

	result := mockBucket.BucketType()
	assert.Equal(t, expectedType, result)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.DeleteObjectRequest{Name: "test-object"}

	mockBucket.On("DeleteObject", ctx, req).Return(nil)

	err := mockBucket.DeleteObject(ctx, req)
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteObject_Error(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.DeleteObjectRequest{Name: "test-object"}
	expectedErr := errors.New("delete failed")

	mockBucket.On("DeleteObject", ctx, req).Return(expectedErr)

	err := mockBucket.DeleteObject(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_StatObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.StatObjectRequest{Name: "test-object"}
	expectedMinObj := &gcs.MinObject{Name: "test-object"}
	expectedExtAttrs := &gcs.ExtendedObjectAttributes{}

	mockBucket.On("StatObject", ctx, req).Return(expectedMinObj, expectedExtAttrs, nil)

	minObj, extAttrs, err := mockBucket.StatObject(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedMinObj, minObj)
	assert.Equal(t, expectedExtAttrs, extAttrs)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_ListObjects(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.ListObjectsRequest{Prefix: "test-"}
	expectedListing := &gcs.Listing{}

	mockBucket.On("ListObjects", ctx, req).Return(expectedListing, nil)

	listing, err := mockBucket.ListObjects(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedListing, listing)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_CopyObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.CopyObjectRequest{
		SrcName: "source-object",
		DstName: "dest-object",
	}
	expectedObj := &gcs.Object{Name: "dest-object"}

	mockBucket.On("CopyObject", ctx, req).Return(expectedObj, nil)

	obj, err := mockBucket.CopyObject(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_DeleteFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder"

	mockBucket.On("DeleteFolder", ctx, folderName).Return(nil)

	err := mockBucket.DeleteFolder(ctx, folderName)
	assert.NoError(t, err)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_GetFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "test-folder"
	expectedFolder := &gcs.Folder{Name: folderName}

	mockBucket.On("GetFolder", ctx, folderName).Return(expectedFolder, nil)

	folder, err := mockBucket.GetFolder(ctx, folderName)
	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_CreateFolder(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	folderName := "new-folder"
	expectedFolder := &gcs.Folder{Name: folderName}

	mockBucket.On("CreateFolder", ctx, folderName).Return(expectedFolder, nil)

	folder, err := mockBucket.CreateFolder(ctx, folderName)
	assert.NoError(t, err)
	assert.Equal(t, expectedFolder, folder)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_GCSName(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	obj := &gcs.MinObject{Name: "test-object"}
	expectedName := "gcs-name"

	mockBucket.On("GCSName", obj).Return(expectedName)

	name := mockBucket.GCSName(obj)
	assert.Equal(t, expectedName, name)
	mockBucket.AssertExpectations(t)
}

func TestDummyIOBucket_MoveObject(t *testing.T) {
	mockBucket := &TestifyMockBucket{}
	ctx := context.Background()
	req := &gcs.MoveObjectRequest{
		SrcName: "source-object",
		DstName: "dest-object",
	}
	expectedObj := &gcs.Object{Name: "dest-object"}

	mockBucket.On("MoveObject", ctx, req).Return(expectedObj, nil)

	obj, err := mockBucket.MoveObject(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, expectedObj, obj)
	mockBucket.AssertExpectations(t)
}

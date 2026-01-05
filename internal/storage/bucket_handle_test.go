// Copyright 2022 Google LLC
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
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const missingObjectName string = "test/foo"
const missingFolderName string = "missing"
const dstObjectName string = "gcsfuse/dst.txt"

var ContentType string = "ContentType"
var ContentEncoding string = "ContentEncoding"
var ContentLanguage string = "ContentLanguage"
var CacheControl string = "CacheControl"
var CustomTime string = "CustomTime"
var StorageClass string = "StorageClass"
var ContentDisposition string = "ContentDisposition"

// FakeGCSServer is not handling generation and metageneration checks for Delete flow and IncludeFoldersAsPrefixes check for ListObjects flow.
// Hence, we are not writing tests for these flows.
// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/fsouza/fake-gcs-server/fakestorage/object.go#L515

func minObjectsToMinObjectNames(minObjects []*gcs.MinObject) (objectNames []string) {
	objectNames = make([]string, len(minObjects))
	for i, object := range minObjects {
		if object != nil {
			objectNames[i] = object.Name
		}
	}
	return
}

func createBucketHandle(testSuite *BucketHandleTest, resp *controlpb.StorageLayout) {
	var err error
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(resp, nil)
	testSuite.bucketHandle, err = testSuite.storageHandle.BucketHandle(context.Background(), TestBucketName, "", false)
	testSuite.bucketHandle.controlClient = testSuite.mockClient

	assert.NotNil(testSuite.T(), testSuite.bucketHandle)
	assert.Nil(testSuite.T(), err)
}

type BucketHandleTest struct {
	suite.Suite
	bucketHandle  *bucketHandle
	storageHandle StorageHandle
	fakeStorage   FakeStorage
	mockClient    *MockStorageControlClient
}

func TestBucketHandleTestSuite(testSuite *testing.T) {
	suite.Run(testSuite, new(BucketHandleTest))
}

func (testSuite *BucketHandleTest) SetupTest() {
	testSuite.mockClient = new(MockStorageControlClient)
	testSuite.fakeStorage = NewFakeStorageWithMockClient(testSuite.mockClient, cfg.HTTP2)
	testSuite.storageHandle = testSuite.fakeStorage.CreateStorageHandle()
}

func (testSuite *BucketHandleTest) TearDownTest() {
	testSuite.fakeStorage.ShutDown()
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithCompleteRead() {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestObject, string(buf[:]))
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithRangeRead() {
	start := uint64(2)
	limit := uint64(8)

	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: limit,
			},
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, limit-start)
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestObject[start:limit], string(buf[:]))
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithNilRange() {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name:  TestObjectName,
			Range: nil,
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestObject, string(buf[:]))
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithInValidObject() {
	var notFoundErr *gcs.NotFoundError

	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: missingObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})

	assert.NotNil(testSuite.T(), err)
	assert.True(testSuite.T(), errors.As(err, &notFoundErr))
	assert.Nil(testSuite.T(), rc)
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithValidGeneration() {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			Generation: TestObjectGeneration,
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestObject, string(buf[:]))
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithInvalidGeneration() {
	var notFoundErr *gcs.NotFoundError

	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			Generation: 222, // other than TestObjectGeneration, doesn'testSuite exist.
		})

	assert.NotNil(testSuite.T(), err)
	assert.True(testSuite.T(), errors.As(err, &notFoundErr))
	assert.Nil(testSuite.T(), rc)
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithCompressionEnabled() {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestGzipObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestGzipObjectCompressed)),
			},
			ReadCompressed: true,
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestGzipObjectCompressed))
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestGzipObjectCompressed, string(buf))
}

func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithCompressionDisabled() {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestGzipObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestGzipObjectCompressed)),
			},
			ReadCompressed: false,
		})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestGzipObjectDecompressed))
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), ContentInTestGzipObjectDecompressed, string(buf))
}

// Fakestorage doesn't support readHandle concept
func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithoutReadHandle() {
	rd, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			ReadHandle: nil,
		})

	assert.Nil(testSuite.T(), err)
	defer rd.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rd.Read(buf)
	assert.Nil(testSuite.T(), err)
	//assert.Equal(testSuite.T(), len(rd.ReadHandle()), 0)
	assert.Equal(testSuite.T(), ContentInTestObject, string(buf[:]))
}

// Fakestorage doesn't support readHandle concept
func (testSuite *BucketHandleTest) TestNewReaderWithReadHandleMethodWithReadHandle() {
	rd, err := testSuite.bucketHandle.NewReaderWithReadHandle(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			ReadHandle: []byte("opaque-handle"),
		})

	assert.Nil(testSuite.T(), err)
	defer rd.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rd.Read(buf)
	assert.Nil(testSuite.T(), err)
	//assert.Equal(testSuite.T(), len(rd.ReadHandle()), 0)
	assert.Equal(testSuite.T(), ContentInTestObject, string(buf[:]))
}

func (testSuite *BucketHandleTest) TestDeleteObjectMethodWithValidObject() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	err := testSuite.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
		})

	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteObjectMethodWithMissingObject() {
	err := testSuite.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       missingObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
		})

	assert.NotNil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteObjectMethodWithMissingGeneration() {
	err := testSuite.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			MetaGenerationPrecondition: nil,
		})

	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteObjectMethodWithZeroGeneration() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	// Note: fake-gcs-server doesn't respect Generation or other conditions in
	// delete operations. This unit test will be helpful when fake-gcs-server
	// start respecting these conditions, or we move to other testing framework.
	err := testSuite.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			Generation:                 0,
			MetaGenerationPrecondition: nil,
		})

	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestStatObjectMethodWithValidObject() {
	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})

	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestStatObjectMethodWithReturnExtendedObjectAttributesTrue() {
	m, e, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name:                           TestObjectName,
			ReturnExtendedObjectAttributes: true,
		})

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), m)
	assert.NotNil(testSuite.T(), e)
}

func (testSuite *BucketHandleTest) TestStatObjectMethodWithReturnExtendedObjectAttributesFalse() {
	m, e, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name:                           TestObjectName,
			ReturnExtendedObjectAttributes: false,
		})

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), m)
	assert.Nil(testSuite.T(), e)
}

func (testSuite *BucketHandleTest) TestStatObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: missingObjectName,
		})

	assert.True(testSuite.T(), errors.As(err, &notfound))
}

func (testSuite *BucketHandleTest) TestCopyObjectMethodWithValidObject() {
	_, err := testSuite.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       TestObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 TestObjectGeneration,
			SrcMetaGenerationPrecondition: nil,
		})

	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestCopyObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, err := testSuite.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       missingObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 TestObjectGeneration,
			SrcMetaGenerationPrecondition: nil,
		})

	assert.True(testSuite.T(), errors.As(err, &notfound))
}

func (testSuite *BucketHandleTest) TestCopyObjectMethodWithInvalidGeneration() {
	var notfound *gcs.NotFoundError

	_, err := testSuite.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       TestObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 222, // Other than testObjectGeneration, no other generation exists.
			SrcMetaGenerationPrecondition: nil,
		})

	assert.True(testSuite.T(), errors.As(err, &notfound))
}

func (testSuite *BucketHandleTest) TestCreateObjectMethodWithValidObject() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	content := "Creating a new object"
	obj, err := testSuite.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:     "test_object",
			Contents: strings.NewReader(content),
		})

	assert.Equal(testSuite.T(), obj.Name, "test_object")
	assert.Equal(testSuite.T(), int(obj.Size), len(content))
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestCreateObjectMethodWithGenerationAsZero() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	content := "Creating a new object"
	var generation int64 = 0
	obj, err := testSuite.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	assert.Equal(testSuite.T(), obj.Name, "test_object")
	assert.Equal(testSuite.T(), int(obj.Size), len(content))
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestCreateObjectMethodWithGenerationAsZeroWhenObjectAlreadyExists() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	content := "Creating a new object"
	var generation int64 = 0
	var precondition *gcs.PreconditionError
	obj, err := testSuite.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	assert.Equal(testSuite.T(), obj.Name, "test_object")
	assert.Equal(testSuite.T(), int(obj.Size), len(content))
	assert.Nil(testSuite.T(), err)

	obj, err = testSuite.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	assert.Nil(testSuite.T(), obj)
	assert.True(testSuite.T(), errors.As(err, &precondition))
}

func (testSuite *BucketHandleTest) TestCreateObjectMethodWhenGivenGenerationObjectNotExist() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	var precondition *gcs.PreconditionError
	content := "Creating a new object"
	var crc32 uint32 = 45
	var generation int64 = 786

	obj, err := testSuite.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			CRC32C:                 &crc32,
			GenerationPrecondition: &generation,
		})

	assert.Nil(testSuite.T(), obj)
	assert.True(testSuite.T(), errors.As(err, &precondition))
}

func (testSuite *BucketHandleTest) TestBucketHandle_CreateObjectChunkWriter() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	var generation0 int64 = 0
	var generationNon0 int64 = 786
	var metaGeneration0 int64 = 0
	var metaGenerationNon0 int64 = 987

	tests := []struct {
		name           string
		generation     *int64
		metageneration *int64
		objectName     string
		chunkSize      int
	}{
		{
			name:       "NilGeneration",
			generation: nil,
			objectName: "test_object_1",
			chunkSize:  1024 * 1024,
		},
		{
			name:       "GenerationAsZero",
			generation: &generation0,
			objectName: "test_object_2",
			chunkSize:  1024 * 1024,
		},
		{
			name:       "NonZeroGeneration",
			generation: &generationNon0,
			objectName: "test_object_3",
			chunkSize:  1000,
		},
		{
			name:           "NilMetaGeneration",
			metageneration: nil,
			objectName:     "test_object_1",
			chunkSize:      1024 * 1024,
		},
		{
			name:           "MetaGenerationAsZero",
			metageneration: &metaGeneration0,
			objectName:     "test_object_2",
			chunkSize:      1024 * 1024,
		},
		{
			name:           "NonZeroMetaGeneration",
			metageneration: &metaGenerationNon0,
			objectName:     "test_object_3",
			chunkSize:      1000,
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			progressFunc := func(_ int64) {}
			w, err := testSuite.bucketHandle.CreateObjectChunkWriter(context.Background(),
				&gcs.CreateObjectRequest{
					Name:                       tt.objectName,
					GenerationPrecondition:     tt.generation,
					MetaGenerationPrecondition: tt.metageneration,
				},
				tt.chunkSize,
				progressFunc,
			)

			require.NoError(t, err)
			objWr, ok := (w).(*ObjectWriter)
			require.True(t, ok)
			require.NotNil(t, objWr)
			assert.Equal(t, tt.objectName, objWr.ObjectName())
			assert.Equal(t, tt.chunkSize, objWr.ChunkSize)
			assert.Equal(t, reflect.ValueOf(progressFunc).Pointer(), reflect.ValueOf(objWr.ProgressFunc).Pointer())
		})
	}
}

func (testSuite *BucketHandleTest) TestBucketHandle_FinalizeUploadSuccess() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	var generation0 int64 = 0

	tests := []struct {
		name       string
		generation *int64
		objectName string
		chunkSize  int
	}{
		{
			name:       "NilGeneration",
			generation: nil,
			objectName: "test_object_1",
			chunkSize:  1024 * 1024,
		},
		{
			name:       "GenerationAsZero",
			generation: &generation0,
			objectName: "test_object_2",
			chunkSize:  1024 * 100,
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			wr := testSuite.createObjectChunkWriter(t, tt.objectName, tt.generation, tt.chunkSize)

			o, err := testSuite.bucketHandle.FinalizeUpload(context.Background(), wr)

			assert.NoError(t, err)
			assert.NotNil(testSuite.T(), o)
		})
	}
}

func (testSuite *BucketHandleTest) TestFinalizeUploadWithGenerationAsZeroWhenObjectAlreadyExists() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	// Pre-create the object (creating writer and finalizing upload).
	objName := "pre_created_test_object"
	var generation int64 = 0
	wr := testSuite.createObjectChunkWriter(testSuite.T(), objName, &generation, 100)
	o, err := testSuite.bucketHandle.FinalizeUpload(context.Background(), wr)
	require.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), o)
	// Create Object Writer again when object already exists.
	wr = testSuite.createObjectChunkWriter(testSuite.T(), objName, &generation, 100)

	o, err = testSuite.bucketHandle.FinalizeUpload(context.Background(), wr)

	assert.Error(testSuite.T(), err)
	assert.IsType(testSuite.T(), &gcs.PreconditionError{}, err)
	assert.Nil(testSuite.T(), o)
}

func (testSuite *BucketHandleTest) createObjectChunkWriter(t *testing.T, objectName string, generation *int64, chunkSize int) gcs.Writer {
	t.Helper()
	progressFunc := func(_ int64) {}
	wr, err := testSuite.bucketHandle.CreateObjectChunkWriter(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   objectName,
			GenerationPrecondition: generation,
		},
		chunkSize,
		progressFunc,
	)
	require.NoError(t, err)
	objWr, ok := (wr).(*ObjectWriter)
	require.True(t, ok)
	require.NotNil(t, objWr)
	assert.Equal(t, objectName, objWr.ObjectName())
	assert.Equal(t, chunkSize, objWr.ChunkSize)
	assert.Equal(t, reflect.ValueOf(progressFunc).Pointer(), reflect.ValueOf(objWr.ProgressFunc).Pointer())

	return wr
}

func (testSuite *BucketHandleTest) TestFlushPendingWritesFails() {
	// These tests only run with HTTP client because fake storage server is not
	// integrated with GRPC.
	var generation0 int64 = 0
	tests := []struct {
		bucketType string
	}{
		{
			bucketType: "multiregion",
		},
		{
			bucketType: "zone",
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.bucketType, func(t *testing.T) {
			createBucketHandle(testSuite, &controlpb.StorageLayout{
				LocationType: tt.bucketType,
			})
			wr := testSuite.createObjectChunkWriter(t, TestObjectName, &generation0, 100)

			_, err := testSuite.bucketHandle.FlushPendingWrites(context.Background(), wr)

			require.Error(t, err)
			assert.ErrorContains(testSuite.T(), err, "Flush not supported unless client uses gRPC and Append is set to true")
		})
	}
}

func (testSuite *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsNoAcl() {
	proj := getProjectionValue(gcs.NoAcl)

	assert.Equal(testSuite.T(), storage.ProjectionNoACL, proj)
}

func (testSuite *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsFull() {
	proj := getProjectionValue(gcs.Full)

	assert.Equal(testSuite.T(), storage.ProjectionFull, proj)
}

func (testSuite *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsDefault() {
	proj := getProjectionValue(0)

	assert.Equal(testSuite.T(), storage.ProjectionFull, proj)
}

func (testSuite *BucketHandleTest) TestListObjectMethodWithPrefixObjectExist() {
	obj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "/",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(obj.MinObjects))
	assert.Equal(testSuite.T(), TestObjectGeneration, obj.MinObjects[0].Generation)
	assert.Equal(testSuite.T(), []string{TestObjectSubRootFolderName}, obj.CollapsedRuns)
}

func (testSuite *BucketHandleTest) TestListObjectMethodWithPrefixObjectDoesNotExist() {
	obj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "PrefixObjectDoesNotExist",
			Delimiter:                "/",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), obj.MinObjects)
	assert.Nil(testSuite.T(), obj.CollapsedRuns)
}

func (testSuite *BucketHandleTest) TestListObjectMethodWithIncludeTrailingDelimiterFalse() {
	obj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "/",
			IncludeTrailingDelimiter: false,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(obj.MinObjects))
	assert.Equal(testSuite.T(), []string{TestObjectSubRootFolderName}, obj.CollapsedRuns)
}

// If Delimiter is empty, all the objects will appear with same prefix.
func (testSuite *BucketHandleTest) TestListObjectMethodWithEmptyDelimiter() {
	obj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(obj.MinObjects))
	assert.Equal(testSuite.T(), TestObjectGeneration, obj.MinObjects[0].Generation)
	assert.Nil(testSuite.T(), obj.CollapsedRuns)
}

// We have 5 objects in fakeserver.
func (testSuite *BucketHandleTest) TestListObjectMethodForMaxResult() {
	fiveObj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: false,
			ContinuationToken:        "",
			MaxResults:               5,
			ProjectionVal:            0,
		})

	threeObj, err2 := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "/",
			IncludeTrailingDelimiter: false,
			ContinuationToken:        "",
			MaxResults:               3,
			ProjectionVal:            0,
		})

	// Validate that 5 objects are listed when MaxResults is passed 5.
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(fiveObj.MinObjects))
	assert.Nil(testSuite.T(), fiveObj.CollapsedRuns)

	// Note: The behavior is different in real GCS storage JSON API. In real API,
	// only 1 object and 1 collapsedRuns would have been returned if
	// IncludeTrailingDelimiter = false and 3 objects and 1 collapsedRuns if
	// IncludeTrailingDelimiter = true.
	// This is because fake storage doesn'testSuite support pagination and hence maxResults
	// has no affect.
	assert.Nil(testSuite.T(), err2)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(threeObj.MinObjects))
	assert.Equal(testSuite.T(), 1, len(threeObj.CollapsedRuns))
}

func (testSuite *BucketHandleTest) TestListObjectMethodWithMissingMaxResult() {
	// Validate that ee have 5 objects in fakeserver
	fiveObjWithMaxResults, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               100,
			ProjectionVal:            0,
		})
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 5, len(fiveObjWithMaxResults.MinObjects))

	fiveObjWithoutMaxResults, err2 := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			ProjectionVal:            0,
		})

	// Validate that all objects (5) are listed when MaxResults is not passed.
	assert.Nil(testSuite.T(), err2)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(fiveObjWithoutMaxResults.MinObjects))
	assert.Nil(testSuite.T(), fiveObjWithoutMaxResults.CollapsedRuns)
}

func (testSuite *BucketHandleTest) TestListObjectMethodWithZeroMaxResult() {
	// Validate that we have 5 objects in fakeserver
	fiveObj, err := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               100,
			ProjectionVal:            0,
		})
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), 5, len(fiveObj.MinObjects))

	fiveObjWithZeroMaxResults, err2 := testSuite.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               0,
			ProjectionVal:            0,
		})

	// Validate that all objects (5) are listed when MaxResults is 0. This has
	// same behavior as not passing MaxResults in request.
	assert.Nil(testSuite.T(), err2)
	assert.Equal(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestObjectName, TestGzipObjectName}, minObjectsToMinObjectNames(fiveObjWithZeroMaxResults.MinObjects))
	assert.Nil(testSuite.T(), fiveObjWithZeroMaxResults.CollapsedRuns)
}

// FakeGCSServer is not handling ContentType, ContentEncoding, ContentLanguage, CacheControl in updateflow
// Hence, we are not writing tests for these parameters
// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/fsouza/fake-gcs-server/fakestorage/object.go#L795
func (testSuite *BucketHandleTest) TestUpdateObjectMethodWithValidObject() {
	// Metadata value before updating object
	minObj, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), minObj)
	assert.Equal(testSuite.T(), MetaDataValue, minObj.Metadata[MetaDataKey])

	updatedMetaData := time.RFC3339Nano
	expectedMetaData := map[string]string{
		MetaDataKey: updatedMetaData,
	}

	updatedObj, err := testSuite.bucketHandle.UpdateObject(context.Background(),
		&gcs.UpdateObjectRequest{
			Name:                       TestObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
			ContentType:                &ContentType,
			ContentEncoding:            &ContentEncoding,
			ContentLanguage:            &ContentLanguage,
			CacheControl:               &CacheControl,
			Metadata: map[string]*string{
				MetaDataKey: &updatedMetaData,
			},
		})

	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), TestObjectName, updatedObj.Name)
	// Metadata value after updating object
	assert.Equal(testSuite.T(), expectedMetaData[MetaDataKey], updatedObj.Metadata[MetaDataKey])
}

func (testSuite *BucketHandleTest) TestUpdateObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, err := testSuite.bucketHandle.UpdateObject(context.Background(),
		&gcs.UpdateObjectRequest{
			Name:                       missingObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
			ContentType:                &ContentType,
			ContentEncoding:            &ContentEncoding,
			ContentLanguage:            &ContentLanguage,
			CacheControl:               &CacheControl,
			Metadata:                   nil,
		})

	assert.True(testSuite.T(), errors.As(err, &notfound))
}

// Read content of an object and return
func (testSuite *BucketHandleTest) readObjectContent(ctx context.Context, req *gcs.ReadObjectRequest) (buffer string) {
	rc, err := testSuite.bucketHandle.NewReaderWithReadHandle(ctx, &gcs.ReadObjectRequest{
		Name:  req.Name,
		Range: req.Range})

	assert.Nil(testSuite.T(), err)
	defer rc.Close()
	buf := make([]byte, req.Range.Limit)
	_, err = rc.Read(buf)
	assert.Nil(testSuite.T(), err)
	return string(buf[:])
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWithDstObjectExist() {
	// Reading content before composing it
	buffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestObject, buffer)
	// Checking if srcObject exists or not
	srcMinObj, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj)

	// Composing the object
	composedObj, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       TestObjectName,
			DstGenerationPrecondition:     nil,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: TestSubObjectName,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	assert.Nil(testSuite.T(), err)
	// Validation of srcObject to ensure that it is not effected.
	srcBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestSubObject, srcBuffer)
	// Reading content of destination object
	dstBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Destination object's content will get overwrite by srcObject.
	assert.Equal(testSuite.T(), srcBuffer, dstBuffer)
	assert.NotNil(testSuite.T(), composedObj)
	assert.Equal(testSuite.T(), srcMinObj.Size, composedObj.Size)
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWithOneSrcObject() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	var notfound *gcs.NotFoundError
	// Checking that dstObject does not exist
	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	assert.True(testSuite.T(), errors.As(err, &notfound))
	srcMinObj, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj)

	composedObj, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       dstObjectName,
			DstGenerationPrecondition:     nil,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: TestObjectName,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	assert.Nil(testSuite.T(), err)
	// Validation of srcObject to ensure that it is not effected.
	srcBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Reading content of dstObject
	dstBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	assert.Equal(testSuite.T(), srcBuffer, dstBuffer)
	assert.NotNil(testSuite.T(), composedObj)
	assert.Equal(testSuite.T(), srcMinObj.Size, composedObj.Size)
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWithTwoSrcObjects() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	var notfound *gcs.NotFoundError
	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	assert.True(testSuite.T(), errors.As(err, &notfound))
	srcMinObj1, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj1)
	srcMinObj2, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj2)

	composedObj, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       dstObjectName,
			DstGenerationPrecondition:     nil,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: TestObjectName,
				},
				{
					Name: TestSubObjectName,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	assert.Nil(testSuite.T(), err)
	// Validation of srcObject1 to ensure that it is not effected.
	srcBuffer1 := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	// Validation of srcObject2 to ensure that it is not effected.
	srcBuffer2 := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	// Reading content of dstObject
	dstBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Comparing content of destination object
	assert.Equal(testSuite.T(), srcBuffer1+srcBuffer2, dstBuffer)
	assert.NotNil(testSuite.T(), composedObj)
	assert.Equal(testSuite.T(), srcMinObj1.Size+srcMinObj2.Size, composedObj.Size)
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWhenSrcObjectDoesNotExist() {
	var notfound *gcs.NotFoundError
	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: missingObjectName,
		})
	// SrcObject does not exist
	assert.True(testSuite.T(), errors.As(err, &notfound))

	_, err = testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       TestObjectName,
			DstGenerationPrecondition:     nil,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: missingObjectName,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	// For fakeobject it is giving googleapi 500 error, where as in real mounting we are getting "404 not found error"
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWhenSourceIsNil() {
	_, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       TestObjectName,
			DstGenerationPrecondition:     nil,
			DstMetaGenerationPrecondition: nil,
			Sources:                       nil,
			ContentType:                   ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	// error : Error in composing object: storage: at least one source object must be specified
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestNameMethod() {
	name := testSuite.bucketHandle.Name()

	assert.Equal(testSuite.T(), TestBucketName, name)
}

func (testSuite *BucketHandleTest) TestIsStorageConditionsNotEmptyWithEmptyConditions() {
	assert.False(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{}))
}

func (testSuite *BucketHandleTest) TestIsStorageConditionsNotEmptyWithNonEmptyConditions() {
	// GenerationMatch is set.
	assert.True(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{GenerationMatch: 123}))

	// GenerationNotMatch is set.
	assert.True(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{GenerationNotMatch: 123}))

	// MetagenerationMatch is set.
	assert.True(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{MetagenerationMatch: 123}))

	// MetagenerationNotMatch is set.
	assert.True(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{MetagenerationNotMatch: 123}))

	// DoesNotExist is set.
	assert.True(testSuite.T(), isStorageConditionsNotEmpty(storage.Conditions{DoesNotExist: true}))
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWhenDstObjectDoesNotExist() {
	var notfound *gcs.NotFoundError
	_, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	assert.True(testSuite.T(), errors.As(err, &notfound))
	srcMinObj1, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj1)
	srcMinObj2, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj2)

	// Add DstGenerationPrecondition = 0 as the Destination object doesn'testSuite exist.
	// Note: fake-gcs-server doesn'testSuite respect precondition checks but still adding
	// to make sure that it works when precondition checks are supported or we
	// shift to different testing storage.
	var zeroPreCond int64 = 0
	composedObj, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       dstObjectName,
			DstGenerationPrecondition:     &zeroPreCond,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: TestObjectName,
				},
				{
					Name: TestSubObjectName,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})
	assert.Nil(testSuite.T(), err)

	// Validation of srcObject1 to ensure that it is not effected.
	srcBuffer1 := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestObject, srcBuffer1)

	// Validation of srcObject2 to ensure that it is not effected.
	srcBuffer2 := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestSubObject, srcBuffer2)

	// Reading content of dstObject
	dstBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Comparing content of destination object
	assert.Equal(testSuite.T(), srcBuffer1+srcBuffer2, dstBuffer)
	assert.NotNil(testSuite.T(), composedObj)
	assert.Equal(testSuite.T(), srcMinObj1.Size+srcMinObj2.Size, composedObj.Size)
}

func (testSuite *BucketHandleTest) TestComposeObjectMethodWithOneSrcObjectIsDstObject() {
	// Checking source object 1 exists. This will also be the destination object.
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	srcMinObj1, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj1)

	// Reading source object 1 content before composing it
	srcObj1Buffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestObject, srcObj1Buffer)

	// Checking source object 2 exists.
	srcMinObj2, _, err := testSuite.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), srcMinObj2)

	// Reading source object 2 content before composing it
	srcObj2Buffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestSubObject, srcObj2Buffer)

	// Note: fake-gcs-server doesn'testSuite respect precondition checks but still adding
	// to make sure that it works when precondition checks are supported or we
	// shift to different testing storage.
	var preCond int64 = srcMinObj1.Generation
	// Compose srcObj1 and srcObj2 back into srcObj1
	composedObj, err := testSuite.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       srcMinObj1.Name,
			DstGenerationPrecondition:     &preCond,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: srcMinObj1.Name,
				},
				{
					Name: srcMinObj2.Name,
				},
			},
			ContentType: ContentType,
			Metadata: map[string]string{
				MetaDataKey: MetaDataValue,
			},
			ContentLanguage:    ContentLanguage,
			ContentEncoding:    ContentEncoding,
			CacheControl:       CacheControl,
			ContentDisposition: ContentDisposition,
			CustomTime:         CustomTime,
			EventBasedHold:     true,
			StorageClass:       StorageClass,
			Acl:                nil,
		})

	assert.Nil(testSuite.T(), err)
	// Validation of src object 2 to ensure that it is not effected.
	srcObj2BufferAfterCompose := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestSubObject, srcObj2BufferAfterCompose)

	// Reading content of dstObject (composed back into src object 1)
	dstBuffer := testSuite.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	assert.Equal(testSuite.T(), ContentInTestObject+ContentInTestSubObject, dstBuffer)
	assert.NotNil(testSuite.T(), composedObj)
	assert.Equal(testSuite.T(), len(ContentInTestObject)+len(ContentInTestSubObject), int(composedObj.Size))
}

func (testSuite *BucketHandleTest) TestBucketTypeForHierarchicalNameSpaceTrue() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{Hierarchical: true}, testSuite.bucketHandle.bucketType, "Expected Hierarchical bucket type")
}

func (testSuite *BucketHandleTest) TestBucketTypeForZonalLocationType() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		LocationType: "zone",
	})

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{Zonal: true}, testSuite.bucketHandle.bucketType, "Expected Zonal bucket type")
}

func (testSuite *BucketHandleTest) TestBucketTypeForZonalLocationTypeAndHierarchicalNameSpaceTrue() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
		LocationType:          "zone",
	})

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{Hierarchical: true, Zonal: true}, testSuite.bucketHandle.bucketType, "Expected Zonal bucket type")
}

func (testSuite *BucketHandleTest) TestBucketTypeForHierarchicalNameSpaceFalse() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: false},
	})

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{}, testSuite.bucketHandle.bucketType, "Expected default bucket type")
}

func (testSuite *BucketHandleTest) TestBucketHandleWithError() {
	var x *controlpb.StorageLayout
	var err error
	// Test when the client returns an error.
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).Return(x, errors.New("mocked error"))
	testSuite.bucketHandle, err = testSuite.storageHandle.BucketHandle(context.Background(), TestBucketName, "", false)

	assert.Nil(testSuite.T(), testSuite.bucketHandle)
	assert.Contains(testSuite.T(), err.Error(), "mocked error")
}

func (testSuite *BucketHandleTest) TestBucketHandleWithRapidAppendsEnabled() {
	var err error
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).Return(&controlpb.StorageLayout{}, nil)
	testSuite.mockClient.On("getClient", mock.Anything, mock.Anything).Return(&storage.Client{}, nil)

	testSuite.bucketHandle, err = testSuite.storageHandle.BucketHandle(context.Background(), TestBucketName, "", false)

	assert.NotNil(testSuite.T(), testSuite.bucketHandle)
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestBucketTypeWithHierarchicalNamespaceIsNil() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{}, testSuite.bucketHandle.bucketType, "Expected default bucket type")
}

func (testSuite *BucketHandleTest) TestDefaultBucketTypeWithControlClientNil() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{})
	var nilControlClient *control.StorageControlClient = nil
	testSuite.bucketHandle.controlClient = nilControlClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), &gcs.BucketType{}, testSuite.bucketHandle.bucketType, "Expected default bucket type")
}

func (testSuite *BucketHandleTest) TestDeleteFolderWhenFolderExitForHierarchicalBucket() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	ctx := context.Background()
	deleteFolderReq := controlpb.DeleteFolderRequest{Name: fmt.Sprintf(FullFolderPathHNS, TestBucketName, TestFolderName)}
	testSuite.mockClient.On("DeleteFolder", ctx, &deleteFolderReq, mock.Anything).Return(nil)
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	err := testSuite.bucketHandle.DeleteFolder(ctx, TestFolderName)

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteFolderWhenFolderNotExistForHierarchicalBucket() {
	ctx := context.Background()
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	deleteFolderReq := controlpb.DeleteFolderRequest{Name: fmt.Sprintf(FullFolderPathHNS, TestBucketName, missingFolderName)}
	testSuite.mockClient.On("DeleteFolder", mock.Anything, &deleteFolderReq, mock.Anything).Return(errors.New("mock error"))
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	err := testSuite.bucketHandle.DeleteFolder(ctx, missingFolderName)

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestGetFolderWhenFolderExistsForHierarchicalBucket() {
	ctx := context.Background()
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	folderPath := fmt.Sprintf(FullFolderPathHNS, TestBucketName, TestFolderName)
	getFolderReq := controlpb.GetFolderRequest{Name: folderPath}
	mockFolder := controlpb.Folder{
		Name: folderPath,
	}
	testSuite.mockClient.On("GetFolder", ctx, &getFolderReq, mock.Anything).Return(&mockFolder, nil)
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	result, err := testSuite.bucketHandle.GetFolder(ctx, &gcs.GetFolderRequest{Name: TestFolderName})

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), TestFolderName, result.Name)
}

func (testSuite *BucketHandleTest) TestGetFolderWhenFolderDoesNotExistsForHierarchicalBucket() {
	ctx := context.Background()
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	folderPath := fmt.Sprintf(FullFolderPathHNS, TestBucketName, missingFolderName)
	getFolderReq := controlpb.GetFolderRequest{Name: folderPath}
	testSuite.mockClient.On("GetFolder", ctx, &getFolderReq, mock.Anything).Return(nil, status.Error(codes.NotFound, "folder not found"))
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	result, err := testSuite.bucketHandle.GetFolder(ctx, &gcs.GetFolderRequest{Name: missingFolderName})

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.Nil(testSuite.T(), result)
	assert.ErrorContains(testSuite.T(), err, "folder not found")
}

func (testSuite *BucketHandleTest) TestRenameFolderWithError() {
	ctx := context.Background()
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	renameFolderReq := controlpb.RenameFolderRequest{Name: fmt.Sprintf(FullFolderPathHNS, TestBucketName, TestFolderName), DestinationFolderId: TestRenameFolder}
	testSuite.mockClient.On("RenameFolder", mock.Anything, &renameFolderReq, mock.Anything).Return(nil, errors.New("mock error"))
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	_, err := testSuite.bucketHandle.RenameFolder(ctx, TestFolderName, TestRenameFolder)

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestCreateFolderWithError() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	createFolderReq := controlpb.CreateFolderRequest{Parent: fmt.Sprintf(FullBucketPathHNS, TestBucketName), FolderId: TestFolderName, Recursive: true}
	testSuite.mockClient.On("CreateFolder", context.Background(), &createFolderReq, mock.Anything).Return(nil, errors.New("mock error"))
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	folder, err := testSuite.bucketHandle.CreateFolder(context.Background(), TestFolderName)

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), folder)
}

func (testSuite *BucketHandleTest) TestCreateFolderWithGivenName() {
	createBucketHandle(testSuite, &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
	})
	mockFolder := controlpb.Folder{
		Name: fmt.Sprintf(FullFolderPathHNS, TestBucketName, TestFolderName),
	}
	createFolderReq := controlpb.CreateFolderRequest{Parent: fmt.Sprintf(FullBucketPathHNS, TestBucketName), FolderId: TestFolderName, Recursive: true}
	testSuite.mockClient.On("CreateFolder", context.Background(), &createFolderReq, mock.Anything).Return(&mockFolder, nil)
	testSuite.bucketHandle.bucketType = &gcs.BucketType{Hierarchical: true}

	folder, err := testSuite.bucketHandle.CreateFolder(context.Background(), TestFolderName)

	testSuite.mockClient.AssertExpectations(testSuite.T())
	assert.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), gcs.GCSFolder(TestBucketName, &mockFolder), folder)
}

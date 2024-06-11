// Copyright 2022 Google Inc. All Rights Reserved.
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
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const missingObjectName string = "test/foo"
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

func objectsToObjectNames(objects []*gcs.Object) (objectNames []string) {
	objectNames = make([]string, len(objects))
	for i, object := range objects {
		if object != nil {
			objectNames[i] = object.Name
		}
	}
	return
}

type BucketHandleTest struct {
	suite.Suite
	bucketHandle  *bucketHandle
	storageHandle StorageHandle
	fakeStorage   FakeStorage
}

func TestBucketHandleTestSuite(testSuite *testing.T) {
	suite.Run(testSuite, new(BucketHandleTest))
}

func (testSuite *BucketHandleTest) SetupTest() {
	testSuite.fakeStorage = NewFakeStorage()
	testSuite.storageHandle = testSuite.fakeStorage.CreateStorageHandle()
	testSuite.bucketHandle = testSuite.storageHandle.BucketHandle(TestBucketName, "")

	assert.NotNil(testSuite.T(), testSuite.bucketHandle)
}

func (testSuite *BucketHandleTest) TearDownTest() {
	testSuite.fakeStorage.ShutDown()
}

func (testSuite *BucketHandleTest) TestNewReaderMethodWithCompleteRead() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestNewReaderMethodWithRangeRead() {
	start := uint64(2)
	limit := uint64(8)

	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestNewReaderMethodWithNilRange() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestNewReaderMethodWithInValidObject() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: missingObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), rc)
}

func (testSuite *BucketHandleTest) TestNewReaderMethodWithValidGeneration() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestNewReaderMethodWithInvalidGeneration() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			Generation: 222, // other than TestObjectGeneration, doesn'testSuite exist.
		})

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), rc)
}

func (testSuite *BucketHandleTest) TestNewReaderMethodWithCompressionEnabled() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestNewReaderMethodWithCompressionDisabled() {
	rc, err := testSuite.bucketHandle.NewReader(context.Background(),
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

func (testSuite *BucketHandleTest) TestDeleteObjectMethodWithValidObject() {
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

	assert.Equal(testSuite.T(), "gcs.NotFoundError: storage: object doesn't exist", err.Error())
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
	// Note: fake-gcs-server doesn'testSuite respect Generation or other conditions in
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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestObjectName, TestGzipObjectName}, objectsToObjectNames(obj.Objects))
	assert.Equal(testSuite.T(), TestObjectGeneration, obj.Objects[0].Generation)
	assert.ElementsMatch(testSuite.T(), []string{TestObjectSubRootFolderName}, obj.CollapsedRuns)
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
	assert.Nil(testSuite.T(), obj.Objects)
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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectName, TestGzipObjectName}, objectsToObjectNames(obj.Objects))
	assert.ElementsMatch(testSuite.T(), []string{TestObjectSubRootFolderName}, obj.CollapsedRuns)
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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestGzipObjectName, TestObjectName}, objectsToObjectNames(obj.Objects))
	assert.Equal(testSuite.T(), TestObjectGeneration, obj.Objects[0].Generation)
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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestGzipObjectName, TestObjectName}, objectsToObjectNames(fiveObj.Objects))
	assert.Nil(testSuite.T(), fiveObj.CollapsedRuns)

	// Note: The behavior is different in real GCS storage JSON API. In real API,
	// only 1 object and 1 collapsedRuns would have been returned if
	// IncludeTrailingDelimiter = false and 3 objects and 1 collapsedRuns if
	// IncludeTrailingDelimiter = true.
	// This is because fake storage doesn'testSuite support pagination and hence maxResults
	// has no affect.
	assert.Nil(testSuite.T(), err2)
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestGzipObjectName, TestObjectName}, objectsToObjectNames(threeObj.Objects))
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
	assert.Equal(testSuite.T(), 5, len(fiveObjWithMaxResults.Objects))

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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestGzipObjectName, TestObjectName}, objectsToObjectNames(fiveObjWithoutMaxResults.Objects))
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
	assert.Equal(testSuite.T(), 5, len(fiveObj.Objects))

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
	assert.ElementsMatch(testSuite.T(), []string{TestObjectRootFolderName, TestObjectSubRootFolderName, TestSubObjectName, TestGzipObjectName, TestObjectName}, objectsToObjectNames(fiveObjWithZeroMaxResults.Objects))
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
	rc, err := testSuite.bucketHandle.NewReader(ctx, &gcs.ReadObjectRequest{
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

func (testSuite *BucketHandleTest) TestBucketTypeMethod() {
	bucketType := testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.NonHierarchical, bucketType)
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
	mockClient := new(MockStorageControlClient)
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{
			HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
		}, nil)
	testSuite.bucketHandle.controlClient = mockClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.Hierarchical, testSuite.bucketHandle.bucketType, "Expected Hierarchical bucket type")
}

func (testSuite *BucketHandleTest) TestBucketTypeForHierarchicalNameSpaceFalse() {
	mockClient := new(MockStorageControlClient)
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{
			HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: false},
		}, nil)
	testSuite.bucketHandle.controlClient = mockClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.NonHierarchical, testSuite.bucketHandle.bucketType, "Expected NonHierarchical bucket type")
}

func (testSuite *BucketHandleTest) TestBucketTypeWithError() {
	var x *controlpb.StorageLayout
	mockClient := new(MockStorageControlClient)
	// Test when the client returns an error.
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(x, errors.New("mocked error"))
	testSuite.bucketHandle.controlClient = mockClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.Unknown, testSuite.bucketHandle.bucketType, "Expected Unknown when there's an error")
}

func (testSuite *BucketHandleTest) TestBucketTypeWithHierarchicalNamespaceIsNil() {
	mockClient := new(MockStorageControlClient)
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	testSuite.bucketHandle.controlClient = mockClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.NonHierarchical, testSuite.bucketHandle.bucketType, "Expected NonHierarchical bucket type")
}

func (testSuite *BucketHandleTest) TestDefaultBucketTypeWithControlClientNil() {
	var nilControlClient *control.StorageControlClient = nil
	testSuite.bucketHandle.controlClient = nilControlClient

	testSuite.bucketHandle.BucketType()

	assert.Equal(testSuite.T(), gcs.NonHierarchical, testSuite.bucketHandle.bucketType, "Expected Hierarchical bucket type")
}

func (testSuite *BucketHandleTest) TestDeleteFolderWhenFolderExitForHierarchicalBucket() {
	ctx := context.Background()
	mockClient := new(MockStorageControlClient)
	mockClient.On("DeleteFolder", ctx, &controlpb.DeleteFolderRequest{Name: "projects/_/buckets/" + TestBucketName + "/folders/" + TestObjectName}, mock.Anything).
		Return(nil)
	testSuite.bucketHandle.controlClient = mockClient
	testSuite.bucketHandle.bucketType = gcs.Hierarchical

	err := testSuite.bucketHandle.DeleteFolder(ctx, TestObjectName)

	mockClient.AssertExpectations(testSuite.T())
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteFolderWhenFolderExistButSameObjectNotExistInHierarchicalBucket() {
	ctx := context.Background()
	mockClient := new(MockStorageControlClient)
	mockClient.On("DeleteFolder", ctx, &controlpb.DeleteFolderRequest{Name: "projects/_/buckets/" + TestBucketName + "/folders/" + missingObjectName}, mock.Anything).
		Return(nil)
	testSuite.bucketHandle.controlClient = mockClient
	testSuite.bucketHandle.bucketType = gcs.Hierarchical

	err := testSuite.bucketHandle.DeleteFolder(ctx, missingObjectName)

	mockClient.AssertExpectations(testSuite.T())
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BucketHandleTest) TestDeleteFolderWhenFolderNotExistForHierarchicalBucket() {
	ctx := context.Background()
	mockClient := new(MockStorageControlClient)
	mockClient.On("DeleteFolder", mock.Anything, &controlpb.DeleteFolderRequest{Name: "projects/_/buckets/" + TestBucketName + "/folders/" + missingObjectName}, mock.Anything).
		Return(errors.New("mock error"))
	testSuite.bucketHandle.controlClient = mockClient
	testSuite.bucketHandle.bucketType = gcs.Hierarchical

	err := testSuite.bucketHandle.DeleteFolder(ctx, missingObjectName)

	mockClient.AssertExpectations(testSuite.T())
	assert.Equal(testSuite.T(), "DeleteFolder: mock error", err.Error())
}

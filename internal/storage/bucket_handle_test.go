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
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
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

// FakeGCSServer is not handling generation and metageneration checks for Delete flow.
// Hence, we are not writing tests for these flows.
// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/fsouza/fake-gcs-server/fakestorage/object.go#L515

func TestBucketHandle(t *testing.T) { RunTests(t) }

type BucketHandleTest struct {
	bucketHandle  *bucketHandle
	storageHandle StorageHandle
	fakeStorage   FakeStorage
}

var _ SetUpInterface = &BucketHandleTest{}
var _ TearDownInterface = &BucketHandleTest{}

func init() { RegisterTestSuite(&BucketHandleTest{}) }

func (t *BucketHandleTest) SetUp(_ *TestInfo) {
	t.fakeStorage = NewFakeStorage()
	t.storageHandle = t.fakeStorage.CreateStorageHandle()
	t.bucketHandle = t.storageHandle.BucketHandle(TestBucketName, "")

	AssertNe(nil, t.bucketHandle)
}

func (t *BucketHandleTest) TearDown() {
	t.fakeStorage.ShutDown()
}

func (t *BucketHandleTest) TestNewReaderMethodWithCompleteRead() {
	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	ExpectEq(string(buf[:]), ContentInTestObject)
}

func (t *BucketHandleTest) TestNewReaderMethodWithRangeRead() {
	start := uint64(2)
	limit := uint64(8)

	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: limit,
			},
		})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, limit-start)
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	ExpectEq(string(buf[:]), ContentInTestObject[start:limit])
}

func (t *BucketHandleTest) TestNewReaderMethodWithNilRange() {
	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name:  TestObjectName,
			Range: nil,
		})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	ExpectEq(string(buf[:]), ContentInTestObject[:])
}

func (t *BucketHandleTest) TestNewReaderMethodWithInValidObject() {
	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: missingObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})

	AssertNe(nil, err)
	AssertEq(nil, rc)
}

func (t *BucketHandleTest) TestNewReaderMethodWithValidGeneration() {
	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			Generation: TestObjectGeneration,
		})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, len(ContentInTestObject))
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	ExpectEq(string(buf[:]), ContentInTestObject)
}

func (t *BucketHandleTest) TestNewReaderMethodWithInvalidGeneration() {
	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
			Generation: 222, // other than TestObjectGeneration, doesn't exist.
		})

	AssertNe(nil, err)
	AssertEq(nil, rc)
}

func (t *BucketHandleTest) TestDeleteObjectMethodWithValidObject() {
	err := t.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
		})

	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestDeleteObjectMethodWithMissingObject() {
	err := t.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       missingObjectName,
			Generation:                 TestObjectGeneration,
			MetaGenerationPrecondition: nil,
		})

	AssertEq("gcs.NotFoundError: storage: object doesn't exist", err.Error())
}

func (t *BucketHandleTest) TestDeleteObjectMethodWithMissingGeneration() {
	err := t.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			MetaGenerationPrecondition: nil,
		})

	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestDeleteObjectMethodWithZeroGeneration() {
	// Note: fake-gcs-server doesn't respect Generation or other conditions in
	// delete operations. This unit test will be helpful when fake-gcs-server
	// start respecting these conditions, or we move to other testing framework.
	err := t.bucketHandle.DeleteObject(context.Background(),
		&gcs.DeleteObjectRequest{
			Name:                       TestObjectName,
			Generation:                 0,
			MetaGenerationPrecondition: nil,
		})

	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestStatObjectMethodWithValidObject() {
	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})

	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestStatObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: missingObjectName,
		})

	AssertTrue(errors.As(err, &notfound))
}

func (t *BucketHandleTest) TestCopyObjectMethodWithValidObject() {
	_, err := t.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       TestObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 TestObjectGeneration,
			SrcMetaGenerationPrecondition: nil,
		})

	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestCopyObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, err := t.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       missingObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 TestObjectGeneration,
			SrcMetaGenerationPrecondition: nil,
		})

	AssertTrue(errors.As(err, &notfound))
}

func (t *BucketHandleTest) TestCopyObjectMethodWithInvalidGeneration() {
	var notfound *gcs.NotFoundError

	_, err := t.bucketHandle.CopyObject(context.Background(),
		&gcs.CopyObjectRequest{
			SrcName:                       TestObjectName,
			DstName:                       dstObjectName,
			SrcGeneration:                 222, // Other than testObjectGeneration, no other generation exists.
			SrcMetaGenerationPrecondition: nil,
		})

	AssertTrue(errors.As(err, &notfound))
}

func (t *BucketHandleTest) TestCreateObjectMethodWithValidObject() {
	content := "Creating a new object"
	obj, err := t.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:     "test_object",
			Contents: strings.NewReader(content),
		})

	AssertEq(obj.Name, "test_object")
	AssertEq(obj.Size, len(content))
	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestCreateObjectMethodWithGenerationAsZero() {
	content := "Creating a new object"
	var generation int64 = 0
	obj, err := t.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	AssertEq(obj.Name, "test_object")
	AssertEq(obj.Size, len(content))
	AssertEq(nil, err)
}

func (t *BucketHandleTest) TestCreateObjectMethodWithGenerationAsZeroWhenObjectAlreadyExists() {
	content := "Creating a new object"
	var generation int64 = 0
	var precondition *gcs.PreconditionError
	obj, err := t.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	AssertEq(obj.Name, "test_object")
	AssertEq(obj.Size, len(content))
	AssertEq(nil, err)

	obj, err = t.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			GenerationPrecondition: &generation,
		})

	AssertEq(nil, obj)
	AssertTrue(errors.As(err, &precondition))
}

func (t *BucketHandleTest) TestCreateObjectMethodWhenGivenGenerationObjectNotExist() {
	var precondition *gcs.PreconditionError
	content := "Creating a new object"
	var crc32 uint32 = 45
	var generation int64 = 786

	obj, err := t.bucketHandle.CreateObject(context.Background(),
		&gcs.CreateObjectRequest{
			Name:                   "test_object",
			Contents:               strings.NewReader(content),
			CRC32C:                 &crc32,
			GenerationPrecondition: &generation,
		})

	AssertEq(nil, obj)
	AssertTrue(errors.As(err, &precondition))
}

func (t *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsNoAcl() {
	proj := getProjectionValue(gcs.NoAcl)

	AssertEq(storage.ProjectionNoACL, proj)
}

func (t *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsFull() {
	proj := getProjectionValue(gcs.Full)

	AssertEq(storage.ProjectionFull, proj)
}

func (t *BucketHandleTest) TestGetProjectValueWhenGcloudProjectionIsDefault() {
	proj := getProjectionValue(0)

	AssertEq(storage.ProjectionFull, proj)
}

func (t *BucketHandleTest) TestListObjectMethodWithPrefixObjectExist() {
	obj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "/",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	AssertEq(nil, err)
	AssertEq(3, len(obj.Objects))
	AssertEq(1, len(obj.CollapsedRuns))
	AssertEq(TestObjectRootFolderName, obj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, obj.Objects[1].Name)
	AssertEq(TestObjectName, obj.Objects[2].Name)
	AssertEq(TestObjectGeneration, obj.Objects[0].Generation)
	AssertEq(TestObjectSubRootFolderName, obj.CollapsedRuns[0])
}

func (t *BucketHandleTest) TestListObjectMethodWithPrefixObjectDoesNotExist() {
	obj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "PrefixObjectDoesNotExist",
			Delimiter:                "/",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	AssertEq(nil, err)
	AssertEq(nil, obj.Objects)
	AssertEq(nil, obj.CollapsedRuns)
}

func (t *BucketHandleTest) TestListObjectMethodWithIncludeTrailingDelimiterFalse() {
	obj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "/",
			IncludeTrailingDelimiter: false,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	AssertEq(nil, err)
	AssertEq(2, len(obj.Objects))
	AssertEq(1, len(obj.CollapsedRuns))
	AssertEq(TestObjectRootFolderName, obj.Objects[0].Name)
	AssertEq(TestObjectName, obj.Objects[1].Name)
	AssertEq(TestObjectSubRootFolderName, obj.CollapsedRuns[0])
}

// If Delimiter is empty, all the objects will appear with same prefix.
func (t *BucketHandleTest) TestListObjectMethodWithEmptyDelimiter() {
	obj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "gcsfuse/",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "ContinuationToken",
			MaxResults:               7,
			ProjectionVal:            0,
		})

	AssertEq(nil, err)
	AssertEq(4, len(obj.Objects))
	AssertEq(TestObjectRootFolderName, obj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, obj.Objects[1].Name)
	AssertEq(TestSubObjectName, obj.Objects[2].Name)
	AssertEq(TestObjectName, obj.Objects[3].Name)
	AssertEq(TestObjectGeneration, obj.Objects[0].Generation)
	AssertEq(nil, obj.CollapsedRuns)
}

// We have 4 objects in fakeserver.
func (t *BucketHandleTest) TestListObjectMethodForMaxResult() {
	fourObj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               4,
			ProjectionVal:            0,
		})

	twoObj, err2 := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               2,
			ProjectionVal:            0,
		})

	// Validate that 4 objects are listed when MaxResults is passed 4.
	AssertEq(nil, err)
	AssertEq(4, len(fourObj.Objects))
	AssertEq(TestObjectRootFolderName, fourObj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, fourObj.Objects[1].Name)
	AssertEq(TestSubObjectName, fourObj.Objects[2].Name)
	AssertEq(TestObjectName, fourObj.Objects[3].Name)
	AssertEq(nil, fourObj.CollapsedRuns)

	// Validate that 2 objects are listed when MaxResults is passed 2.
	AssertEq(nil, err2)
	AssertEq(2, len(twoObj.Objects))
	AssertEq(TestObjectRootFolderName, twoObj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, twoObj.Objects[1].Name)
	AssertEq(nil, twoObj.CollapsedRuns)
}

func (t *BucketHandleTest) TestListObjectMethodWithMissingMaxResult() {
	// Validate that ee have 4 objects in fakeserver
	fourObjWithMaxResults, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               100,
			ProjectionVal:            0,
		})
	AssertEq(nil, err)
	AssertEq(4, len(fourObjWithMaxResults.Objects))

	fourObjWithoutMaxResults, err2 := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			ProjectionVal:            0,
		})

	// Validate that all objects (4) are listed when MaxResults is not passed.
	AssertEq(nil, err2)
	AssertEq(4, len(fourObjWithoutMaxResults.Objects))
	AssertEq(TestObjectRootFolderName, fourObjWithoutMaxResults.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, fourObjWithoutMaxResults.Objects[1].Name)
	AssertEq(TestSubObjectName, fourObjWithoutMaxResults.Objects[2].Name)
	AssertEq(TestObjectName, fourObjWithoutMaxResults.Objects[3].Name)
	AssertEq(nil, fourObjWithoutMaxResults.CollapsedRuns)
}

func (t *BucketHandleTest) TestListObjectMethodWithZeroMaxResult() {
	// Validate that ee have 4 objects in fakeserver
	fourObj, err := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               100,
			ProjectionVal:            0,
		})
	AssertEq(nil, err)
	AssertEq(4, len(fourObj.Objects))

	fourObjWithZeroMaxResults, err2 := t.bucketHandle.ListObjects(context.Background(),
		&gcs.ListObjectsRequest{
			Prefix:                   "",
			Delimiter:                "",
			IncludeTrailingDelimiter: true,
			ContinuationToken:        "",
			MaxResults:               0,
			ProjectionVal:            0,
		})

	// Validate that all objects (4) are listed when MaxResults is 0. This has
	// same behavior as not passing MaxResults in request.
	AssertEq(nil, err2)
	AssertEq(4, len(fourObjWithZeroMaxResults.Objects))
	AssertEq(TestObjectRootFolderName, fourObjWithZeroMaxResults.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, fourObjWithZeroMaxResults.Objects[1].Name)
	AssertEq(TestSubObjectName, fourObjWithZeroMaxResults.Objects[2].Name)
	AssertEq(TestObjectName, fourObjWithZeroMaxResults.Objects[3].Name)
	AssertEq(nil, fourObjWithZeroMaxResults.CollapsedRuns)
}

// FakeGCSServer is not handling ContentType, ContentEncoding, ContentLanguage, CacheControl in updateflow
// Hence, we are not writing tests for these parameters
// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/fsouza/fake-gcs-server/fakestorage/object.go#L795
func (t *BucketHandleTest) TestUpdateObjectMethodWithValidObject() {
	// Metadata value before updating object
	obj, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})

	AssertEq(nil, err)
	AssertEq(MetaDataValue, obj.Metadata[MetaDataKey])

	updatedMetaData := time.RFC3339Nano
	expectedMetaData := map[string]string{
		MetaDataKey: updatedMetaData,
	}

	updatedObj, err := t.bucketHandle.UpdateObject(context.Background(),
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

	AssertEq(nil, err)
	AssertEq(TestObjectName, updatedObj.Name)
	// Metadata value after updating object
	AssertEq(expectedMetaData[MetaDataKey], updatedObj.Metadata[MetaDataKey])
}

func (t *BucketHandleTest) TestUpdateObjectMethodWithMissingObject() {
	var notfound *gcs.NotFoundError

	_, err := t.bucketHandle.UpdateObject(context.Background(),
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

	AssertTrue(errors.As(err, &notfound))
}

// Read content of an object and return
func (t *BucketHandleTest) readObjectContent(ctx context.Context, req *gcs.ReadObjectRequest) (buffer string) {
	rc, err := t.bucketHandle.NewReader(ctx, &gcs.ReadObjectRequest{
		Name:  req.Name,
		Range: req.Range})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, req.Range.Limit)
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	return string(buf[:])
}

func (t *BucketHandleTest) TestComposeObjectMethodWithDstObjectExist() {
	// Reading content before composing it
	buffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	ExpectEq(ContentInTestObject, buffer)
	// Checking if srcObject exists or not
	srcObj, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj)

	// Composing the object
	composedObj, err := t.bucketHandle.ComposeObjects(context.Background(),
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

	AssertEq(nil, err)
	// Validation of srcObject to ensure that it is not effected.
	srcBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	ExpectEq(ContentInTestSubObject, srcBuffer)
	// Reading content of destination object
	dstBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Destination object's content will get overwrite by srcObject.
	ExpectEq(srcBuffer, dstBuffer)
	AssertNe(nil, composedObj)
	AssertEq(srcObj.Size, composedObj.Size)
}

func (t *BucketHandleTest) TestComposeObjectMethodWithOneSrcObject() {
	var notfound *gcs.NotFoundError
	// Checking that dstObject does not exist
	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	AssertTrue(errors.As(err, &notfound))
	srcObj, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj)

	composedObj, err := t.bucketHandle.ComposeObjects(context.Background(),
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

	AssertEq(nil, err)
	// Validation of srcObject to ensure that it is not effected.
	srcBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Reading content of dstObject
	dstBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	ExpectEq(srcBuffer, dstBuffer)
	AssertNe(nil, composedObj)
	AssertEq(srcObj.Size, composedObj.Size)
}

func (t *BucketHandleTest) TestComposeObjectMethodWithTwoSrcObjects() {
	var notfound *gcs.NotFoundError
	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	AssertTrue(errors.As(err, &notfound))
	srcObj1, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj1)
	srcObj2, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj2)

	composedObj, err := t.bucketHandle.ComposeObjects(context.Background(),
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

	AssertEq(nil, err)
	// Validation of srcObject1 to ensure that it is not effected.
	srcBuffer1 := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	// Validation of srcObject2 to ensure that it is not effected.
	srcBuffer2 := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	// Reading content of dstObject
	dstBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Comparing content of destination object
	ExpectEq(srcBuffer1+srcBuffer2, dstBuffer)
	AssertNe(nil, composedObj)
	AssertEq(srcObj1.Size+srcObj2.Size, composedObj.Size)
}

func (t *BucketHandleTest) TestComposeObjectMethodWhenSrcObjectDoesNotExist() {
	var notfound *gcs.NotFoundError
	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: missingObjectName,
		})
	// SrcObject does not exist
	AssertTrue(errors.As(err, &notfound))

	_, err = t.bucketHandle.ComposeObjects(context.Background(),
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
	AssertNe(nil, err)
}

func (t *BucketHandleTest) TestComposeObjectMethodWhenSourceIsNil() {
	_, err := t.bucketHandle.ComposeObjects(context.Background(),
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
	AssertNe(nil, err)
}

func (t *BucketHandleTest) TestNameMethod() {
	name := t.bucketHandle.Name()

	AssertEq(TestBucketName, name)
}

func (t *BucketHandleTest) TestIsStorageConditionsNotEmptyWithEmptyConditions() {
	AssertFalse(isStorageConditionsNotEmpty(storage.Conditions{}))
}

func (t *BucketHandleTest) TestIsStorageConditionsNotEmptyWithNonEmptyConditions() {
	// GenerationMatch is set.
	AssertTrue(isStorageConditionsNotEmpty(storage.Conditions{GenerationMatch: 123}))

	// GenerationNotMatch is set.
	AssertTrue(isStorageConditionsNotEmpty(storage.Conditions{GenerationNotMatch: 123}))

	// MetagenerationMatch is set.
	AssertTrue(isStorageConditionsNotEmpty(storage.Conditions{MetagenerationMatch: 123}))

	// MetagenerationNotMatch is set.
	AssertTrue(isStorageConditionsNotEmpty(storage.Conditions{MetagenerationNotMatch: 123}))

	// DoesNotExist is set.
	AssertTrue(isStorageConditionsNotEmpty(storage.Conditions{DoesNotExist: true}))
}

func (t *BucketHandleTest) TestComposeObjectMethodWhenDstObjectDoesNotExist() {
	var notfound *gcs.NotFoundError
	_, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: dstObjectName,
		})
	AssertTrue(errors.As(err, &notfound))
	srcObj1, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj1)
	srcObj2, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj2)

	// Add DstGenerationPrecondition = 0 as the Destination object doesn't exist.
	// Note: fake-gcs-server doesn't respect precondition checks but still adding
	// to make sure that it works when precondition checks are supported or we
	// shift to different testing storage.
	var zeroPreCond int64 = 0
	composedObj, err := t.bucketHandle.ComposeObjects(context.Background(),
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
	AssertEq(nil, err)

	// Validation of srcObject1 to ensure that it is not effected.
	srcBuffer1 := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	ExpectEq(ContentInTestObject, srcBuffer1)

	// Validation of srcObject2 to ensure that it is not effected.
	srcBuffer2 := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	ExpectEq(ContentInTestSubObject, srcBuffer2)

	// Reading content of dstObject
	dstBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: dstObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	// Comparing content of destination object
	ExpectEq(srcBuffer1+srcBuffer2, dstBuffer)
	AssertNe(nil, composedObj)
	AssertEq(srcObj1.Size+srcObj2.Size, composedObj.Size)
}

func (t *BucketHandleTest) TestComposeObjectMethodWithOneSrcObjectIsDstObject() {
	// Checking source object 1 exists. This will also be the destination object.
	srcObj1, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj1)

	// Reading source object 1 content before composing it
	srcObj1Buffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestObject)),
			},
		})
	ExpectEq(ContentInTestObject, srcObj1Buffer)

	// Checking source object 2 exists.
	srcObj2, err := t.bucketHandle.StatObject(context.Background(),
		&gcs.StatObjectRequest{
			Name: TestSubObjectName,
		})
	AssertEq(nil, err)
	AssertNe(nil, srcObj2)

	// Reading source object 2 content before composing it
	srcObj2Buffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	ExpectEq(ContentInTestSubObject, srcObj2Buffer)

	// Note: fake-gcs-server doesn't respect precondition checks but still adding
	// to make sure that it works when precondition checks are supported or we
	// shift to different testing storage.
	var preCond int64 = srcObj1.Generation
	// Compose srcObj1 and srcObj2 back into srcObj1
	composedObj, err := t.bucketHandle.ComposeObjects(context.Background(),
		&gcs.ComposeObjectsRequest{
			DstName:                       srcObj1.Name,
			DstGenerationPrecondition:     &preCond,
			DstMetaGenerationPrecondition: nil,
			Sources: []gcs.ComposeSource{
				{
					Name: srcObj1.Name,
				},
				{
					Name: srcObj2.Name,
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

	AssertEq(nil, err)
	// Validation of src object 2 to ensure that it is not effected.
	srcObj2BufferAfterCompose := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestSubObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(ContentInTestSubObject)),
			},
		})
	ExpectEq(ContentInTestSubObject, srcObj2BufferAfterCompose)

	// Reading content of dstObject (composed back into src object 1)
	dstBuffer := t.readObjectContent(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(composedObj.Size),
			},
		})
	ExpectEq(ContentInTestObject+ContentInTestSubObject, dstBuffer)
	AssertNe(nil, composedObj)
	AssertEq(len(ContentInTestObject)+len(ContentInTestSubObject), composedObj.Size)
}

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
	var err error
	t.fakeStorage = NewFakeStorage()
	t.storageHandle = t.fakeStorage.CreateStorageHandle()
	t.bucketHandle, err = t.storageHandle.BucketHandle(TestBucketName)

	AssertEq(nil, err)
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

	AssertEq("storage: object doesn't exist", err.Error())
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

func (t *BucketHandleTest) TestCreateObjectMethodWhenGivenGenerationObjectNotExist() {
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
	AssertTrue(strings.Contains(err.Error(), "Error 412: Precondition failed"))
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

	AssertEq(nil, err)
	AssertEq(4, len(fourObj.Objects))
	AssertEq(TestObjectRootFolderName, fourObj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, fourObj.Objects[1].Name)
	AssertEq(TestSubObjectName, fourObj.Objects[2].Name)
	AssertEq(TestObjectName, fourObj.Objects[3].Name)
	AssertEq(nil, fourObj.CollapsedRuns)

	AssertEq(nil, err2)
	AssertEq(2, len(twoObj.Objects))
	AssertEq(TestObjectRootFolderName, twoObj.Objects[0].Name)
	AssertEq(TestObjectSubRootFolderName, twoObj.Objects[1].Name)
	AssertEq(nil, twoObj.CollapsedRuns)
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

func (t *BucketHandleTest) TestNameMethod() {
	name := t.bucketHandle.Name()

	AssertEq(TestBucketName, name)
}

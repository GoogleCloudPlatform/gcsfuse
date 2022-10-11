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

	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

const missingObjectName string = "test/foo"
const dstObjectName string = "gcsfuse/dst.txt"

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

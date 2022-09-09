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
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

const missingObjectName string = "test/foo"

func TestBucketHandle(t *testing.T) { RunTests(t) }

type BucketHandleTest struct {
	fakeStorageServer *fakestorage.Server
	bucketHandle      *bucketHandle
}

var _ SetUpInterface = &BucketHandleTest{}
var _ TearDownInterface = &BucketHandleTest{}

func init() { RegisterTestSuite(&BucketHandleTest{}) }

func (t *BucketHandleTest) SetUp(_ *TestInfo) {
	var err error
	t.fakeStorageServer, err = CreateFakeStorageServer([]fakestorage.Object{GetTestFakeStorageObject()})
	AssertEq(nil, err)

	storageClient := &storageClient{client: t.fakeStorageServer.Client()}
	t.bucketHandle, err = storageClient.BucketHandle(TestBucketName)
	AssertEq(nil, err)
	AssertNe(nil, t.bucketHandle)
}

func (t *BucketHandleTest) TearDown() {
	t.fakeStorageServer.Stop()
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

func (t *BucketHandleTest) TestNewReaderMethodWithGeneration() {
	// Modify the default object with different content
	updatedContent := "Some Modification"
	defaultObj := GetTestFakeStorageObject()
	defaultObj.Generation = 2
	defaultObj.Content = []byte(updatedContent)
	CreateObject(t.fakeStorageServer, defaultObj)

	rc, err := t.bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: TestObjectName,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(updatedContent)),
			},
			Generation: 2,
		})

	AssertEq(nil, err)
	defer rc.Close()
	buf := make([]byte, len(updatedContent))
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	ExpectEq(string(buf[:]), updatedContent)
}

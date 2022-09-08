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

const fileContent string = "hello gcsfuse"
const validFilePathInBucket string = "some/object/file.txt"
const invalidFilePathInBucket string = "test/foo"

func TestBucketHandle(t *testing.T) { RunTests(t) }

type BucketHandleTest struct {
	fakeStorageServer *fakestorage.Server
}

var _ SetUpInterface = &BucketHandleTest{}
var _ TearDownInterface = &BucketHandleTest{}

func init() { RegisterTestSuite(&BucketHandleTest{}) }

func (t *BucketHandleTest) SetUp(_ *TestInfo) {
	var err error
	t.fakeStorageServer, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{
			{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName: validBucketName,
					Name:       validFilePathInBucket,
				},
				Content: []byte(fileContent),
			},
		},
		Host: "127.0.0.1",
		Port: 8081,
	})
	AssertEq(nil, err)
}

func (t *BucketHandleTest) TearDown() {
	t.fakeStorageServer.Stop()
}

func (t *BucketHandleTest) TestNewReaderMethodWithValidObject() {
	fakeClient := t.fakeStorageServer.Client()
	storageClient := &storageClient{client: fakeClient}
	bucketHandle, err := storageClient.BucketHandle(validBucketName)
	AssertEq(nil, err)
	AssertNe(nil, bucketHandle)

	rc, err := bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: validFilePathInBucket,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(fileContent)),
			},
		})

	AssertEq(nil, err)
	buf := make([]byte, len(fileContent))
	_, err = rc.Read(buf)
	AssertEq(nil, err)
	content := string(buf[:])
	ExpectEq(content, fileContent)
}

func (t *BucketHandleTest) TestNewReaderMethodWithInValidObject() {
	fakeClient := t.fakeStorageServer.Client()
	storageClient := &storageClient{client: fakeClient}
	bucketHandle, err := storageClient.BucketHandle(validBucketName)
	AssertEq(nil, err)
	AssertNe(nil, bucketHandle)

	rc, err := bucketHandle.NewReader(context.Background(),
		&gcs.ReadObjectRequest{
			Name: invalidFilePathInBucket,
			Range: &gcs.ByteRange{
				Start: uint64(0),
				Limit: uint64(len(fileContent)),
			},
		})

	AssertNe(nil, err)
	AssertEq(nil, rc)
}

func (t *BucketHandleTest) TestNewReaderMethodWithInValidObject() {

}
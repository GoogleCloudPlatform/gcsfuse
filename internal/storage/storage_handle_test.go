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
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/oauth2"
)

const validBucketName string = "will-be-present-in-fake-server"
const invalidBucketName string = "will-not-be-present-in-fake-server"

func getDefaultStorageClientConfig() (clientConfig storageClientConfig) {
	return storageClientConfig{
		disableHTTP2:        true,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond,
		maxRetryDuration:    30 * time.Second,
		retryMultiplier:     2,
	}
}

func TestStorageHandle(t *testing.T) { RunTests(t) }

type StorageHandleTest struct {
	fakeStorageServer *fakestorage.Server
}

var _ SetUpInterface = &StorageHandleTest{}
var _ TearDownInterface = &StorageHandleTest{}

func init() { RegisterTestSuite(&StorageHandleTest{}) }

func (t *StorageHandleTest) SetUp(_ *TestInfo) {
	var err error
	t.fakeStorageServer, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{
			{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName: validBucketName,
				},
			},
		},
		Host: "127.0.0.1",
		Port: 8081,
	})
	AssertEq(nil, err)
}

func (t *StorageHandleTest) TearDown() {
	t.fakeStorageServer.Stop()
}

func (t *StorageHandleTest) invokeAndVerifyStorageHandle(sc storageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)
	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketExists() {
	fakeClient := t.fakeStorageServer.Client()
	storageClient := &storageClient{client: fakeClient}

	bucketHandle, err := storageClient.BucketHandle(validBucketName)

	AssertEq(nil, err)
	AssertNe(nil, bucketHandle)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExist() {
	fakeClient := t.fakeStorageServer.Client()
	storageClient := &storageClient{client: fakeClient}

	bucketHandle, err := storageClient.BucketHandle(invalidBucketName)

	AssertNe(nil, err)
	AssertEq(nil, bucketHandle)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := getDefaultStorageClientConfig() // by default http2 disabled

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Enabled() {
	sc := getDefaultStorageClientConfig()
	sc.disableHTTP2 = false

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := getDefaultStorageClientConfig()
	sc.maxConnsPerHost = 0

	t.invokeAndVerifyStorageHandle(sc)
}

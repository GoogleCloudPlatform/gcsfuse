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

	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/oauth2"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

func getDefaultStorageClientConfig() (clientConfig StorageClientConfig) {
	return StorageClientConfig{
		ClientProtocol:      mountpkg.HTTP1,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 100,
		TokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		HttpClientTimeout:   800 * time.Millisecond,
		MaxRetryDuration:    30 * time.Second,
		RetryMultiplier:     2,
		UserAgent:           "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) (GCP:gcsfuse)",
	}
}

func TestStorageHandle(t *testing.T) { RunTests(t) }

type StorageHandleTest struct {
	fakeStorage FakeStorage
}

var _ SetUpInterface = &StorageHandleTest{}
var _ TearDownInterface = &StorageHandleTest{}

func init() { RegisterTestSuite(&StorageHandleTest{}) }

func (t *StorageHandleTest) SetUp(_ *TestInfo) {
	var err error
	t.fakeStorage = NewFakeStorage()
	AssertEq(nil, err)
}

func (t *StorageHandleTest) TearDown() {
	t.fakeStorage.ShutDown()
}

func (t *StorageHandleTest) invokeAndVerifyStorageHandle(sc StorageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)
	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject() {
	storageHandle := t.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, "")

	AssertNe(nil, bucketHandle)
	AssertEq(TestBucketName, bucketHandle.bucketName)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := t.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, "")

	AssertEq(nil, bucketHandle.Bucket)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := t.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, projectID)

	AssertNe(nil, bucketHandle)
	AssertEq(TestBucketName, bucketHandle.bucketName)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithNonEmptyBillingProject() {
	storageHandle := t.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, projectID)

	AssertEq(nil, bucketHandle.Bucket)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := getDefaultStorageClientConfig() // by default http1 enabled

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Enabled() {
	sc := getDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP2

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := getDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSet() {
	sc := getDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	t.invokeAndVerifyStorageHandle(sc)
}

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
	"net/url"
	"testing"

	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

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

func (t *StorageHandleTest) invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc storageutil.StorageClientConfig) {
	sc.DisableAuth = true
	handleCreated, err := NewStorageHandle(context.Background(), sc)
	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (t *StorageHandleTest) invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc storageutil.StorageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	AssertEq(nil, handleCreated)
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

func (t *StorageHandleTest) TestNewStorageHandleHttp2DisabledWhenDisableAuthIsTrue() {
	sc := storageutil.GetDefaultStorageClientConfig() // by default http1 enabled

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2DisabledWhenDisableAuthIsFalse() {
	sc := storageutil.GetDefaultStorageClientConfig() // by default http1 enabled

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2EnabledWhenDisableAuthIsTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP2

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2EnabledWhenDisableAuthIsFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP2

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHostWhenDisableAuthIsTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHostWhenDisableAuthIsFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSetWhenDisableAuthIsTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSetWhenDisableAuthIsFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithCustomEndpointWhenDisableAuthIsTrue() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	AssertEq(nil, err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNilAndDisableAuthIsTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNilAndDisableAuthIsFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNotNilAndDisableAuthIsFalse() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	AssertEq(nil, err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenKeyFileIsEmptyAndDisableAuthTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.KeyFile = ""

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalseAndDisableAuthTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ReuseTokenFromUrl = false

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalseAndDisableAuthFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ReuseTokenFromUrl = false

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSetAndDisableAuthTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.TokenUrl = storageutil.CustomTokenUrl

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSetAndDisableAuthFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.TokenUrl = storageutil.CustomTokenUrl

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabledAndDisableAuthTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsTrue(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabledAndDisableAuthFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true

	t.invokeAndVerifyStorageHandleWhenDisableAuthIsFalse(sc)
}

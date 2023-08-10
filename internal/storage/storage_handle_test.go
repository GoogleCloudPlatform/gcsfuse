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
	"net/http"
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

func getDefaultStorageClientConfig() (clientConfig StorageClientConfig) {
	return StorageClientConfig{
		MaxRetryDuration: 30 * time.Second,
		RetryMultiplier:  2,
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

func (t *StorageHandleTest) TestNewStorageHandleWithCustomEndpointOption() {
	sc := getDefaultStorageClientConfig()
	sc.ClientOptions = append(sc.ClientOptions, option.WithEndpoint(CustomEndpoint))

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithoutHttpClientOption() {
	sc := getDefaultStorageClientConfig()
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base:   nil,
			Source: nil,
		},
		Timeout: 20,
	}
	sc.ClientOptions = append(sc.ClientOptions, option.WithHTTPClient(httpClient))

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithHttpClientAndEndpointOption() {
	sc := getDefaultStorageClientConfig()
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base:   nil,
			Source: nil,
		},
		Timeout: 20,
	}
	sc.ClientOptions = append(sc.ClientOptions, option.WithHTTPClient(httpClient))
	sc.ClientOptions = append(sc.ClientOptions, option.WithEndpoint(CustomEndpoint))

	t.invokeAndVerifyStorageHandle(sc)
}

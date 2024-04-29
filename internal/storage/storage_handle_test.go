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
	"fmt"
	"net/url"
	"strings"
	"testing"

	mountpkg "github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	. "github.com/jacobsa/ogletest"
	"github.com/stretchr/testify/suite"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

type StorageHandleTest struct {
	suite.Suite
	fakeStorage FakeStorage
}

func TestStorageHandleSuite(t *testing.T) {
	suite.Run(t, new(StorageHandleTest))
}

func (suite *StorageHandleTest) SetupSuite() {
	var err error
	suite.fakeStorage = NewFakeStorage()
	AssertEq(nil, err)
}

func (suite *StorageHandleTest) TearDownSuite() {
	suite.fakeStorage.ShutDown()
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, "")

	AssertNe(nil, bucketHandle)
	AssertEq(TestBucketName, bucketHandle.bucketName)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, "")

	AssertEq(nil, bucketHandle.Bucket)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, projectID)

	AssertNe(nil, bucketHandle)
	AssertEq(TestBucketName, bucketHandle.bucketName)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithNonEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, projectID)

	AssertEq(nil, bucketHandle.Bucket)
}

func (suite *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := storageutil.GetDefaultStorageClientConfig() // by default http1 enabled

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleHttp2Enabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP2

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithCustomEndpoint() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	AssertEq(nil, err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

// This will fail while fetching the token-source, since key-file doesn't exist.
func (suite *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNil() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
	AssertEq(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenKeyFileIsEmpty() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.KeyFile = ""

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ReuseTokenFromUrl = false

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.TokenUrl = storageutil.CustomTokenUrl

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithInvalidClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true
	sc.ClientProtocol = "test-protocol"

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	AssertNe(nil, err)
	AssertEq(nil, handleCreated)
	AssertTrue(strings.Contains(err.Error(), "invalid client-protocol requested: test-protocol"))
}

func (suite *StorageHandleTest) TestCreateGRPCClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	AssertEq(nil, err)
	AssertNe(nil, storageClient)
}

func (suite *StorageHandleTest) TestCreateHTTPClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	AssertEq(nil, err)
	AssertNe(nil, storageClient)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	storageClient, err := NewStorageHandle(context.Background(), sc)

	AssertEq(nil, err)
	AssertNe(nil, storageClient)
}

func (suite *StorageHandleTest) TestCreateGRPCClientHandle_WithHTTPClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP1

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	AssertNe(nil, err)
	AssertEq(nil, storageClient)
	AssertTrue(strings.Contains(err.Error(), fmt.Sprintf("client-protocol requested is not GRPC: %s", mountpkg.HTTP1)))
}

func (suite *StorageHandleTest) TestCreateHTTPClientHandle_WithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	AssertNe(nil, err)
	AssertEq(nil, storageClient)
	AssertTrue(strings.Contains(err.Error(), fmt.Sprintf("client-protocol requested is not HTTP1 or HTTP2: %s", mountpkg.GRPC)))
}

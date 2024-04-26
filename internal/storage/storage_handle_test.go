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
	suite.Nil(err)
}

func (suite *StorageHandleTest) TearDownSuite() {
	suite.fakeStorage.ShutDown()
}

func (suite *StorageHandleTest) invokeAndVerifyStorageHandle(sc storageutil.StorageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)
	suite.Nil(err)
	suite.NotNil(handleCreated)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject()  {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, "")

	suite.NotNil(bucketHandle)
	suite.Equal(TestBucketName, bucketHandle.bucketName)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, "")

	suite.Nil(bucketHandle.Bucket)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, projectID)

	suite.NotNil(bucketHandle)
	suite.Equal(TestBucketName, bucketHandle.bucketName)
}

func (suite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithNonEmptyBillingProject() {
	storageHandle := suite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, projectID)

	suite.Nil(bucketHandle.Bucket)
}

func (suite *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := storageutil.GetDefaultStorageClientConfig() // by default http1 enabled

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleHttp2Enabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP2

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithCustomEndpoint() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	suite.Nil(err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url

	suite.invokeAndVerifyStorageHandle(sc)
}

// This will fail while fetching the token-source, since key-file doesn't exist.
func (suite *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNil() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	suite.NotNil(err)
	suite.Contains(err.Error(), fmt.Sprintf("no such file or directory"))
	suite.Nil(handleCreated)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenKeyFileIsEmpty() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.KeyFile = ""

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ReuseTokenFromUrl = false

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.TokenUrl = storageutil.CustomTokenUrl

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithInvalidClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true
	sc.ClientProtocol = "test-protocol"

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	suite.NotNil(err)
	suite.Nil(handleCreated)
	AssertTrue(strings.Contains(err.Error(), "invalid client-protocol requested: test-protocol"))
}

func (suite *StorageHandleTest) TestCreateGRPCClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	suite.Nil(err)
	suite.NotNil(storageClient)
}

func (suite *StorageHandleTest) TestCreateHTTPClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	suite.Nil(err)
	suite.NotNil(storageClient)
}

func (suite *StorageHandleTest) TestNewStorageHandleWithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	suite.invokeAndVerifyStorageHandle(sc)
}

func (suite *StorageHandleTest) TestCreateGRPCClientHandle_WithHTTPClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.HTTP1

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	suite.NotNil(err)
	suite.Nil(storageClient)
	suite.Contains(fmt.Sprintf("client-protocol requested is not GRPC: %s", mountpkg.HTTP1), err.Error())
}

func (suite *StorageHandleTest) TestCreateHTTPClientHandle_WithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = mountpkg.GRPC

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	suite.NotNil(err)
	suite.Nil(storageClient)
	suite.Contains(fmt.Sprintf("client-protocol requested is not HTTP1 or HTTP2: %s", mountpkg.GRPC), err.Error())
}

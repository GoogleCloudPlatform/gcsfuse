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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

type StorageHandleTest struct {
	suite.Suite
	fakeStorage FakeStorage
}

func TestStorageHandleTestSuite(t *testing.T) {
	suite.Run(t, new(StorageHandleTest))
}

func (testSuite *StorageHandleTest) SetupTest() {
	var err error
	testSuite.fakeStorage = NewFakeStorage()
	assert.Nil(testSuite.T(), err)
}

func (testSuite *StorageHandleTest) TearDownTest() {
	testSuite.fakeStorage.ShutDown()
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Equal(testSuite.T(), TestBucketName, bucketHandle.bucketName)
	assert.Equal(testSuite.T(), gcs.Nil, bucketHandle.bucketType)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, "")

	assert.Nil(testSuite.T(), bucketHandle.Bucket)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(TestBucketName, projectID)

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Equal(testSuite.T(), TestBucketName, bucketHandle.bucketName)
	assert.Equal(testSuite.T(), gcs.Nil, bucketHandle.bucketType)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithNonEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	bucketHandle := storageHandle.BucketHandle(invalidBucketName, projectID)

	assert.Nil(testSuite.T(), bucketHandle.Bucket)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := storageutil.GetDefaultStorageClientConfig() // by default http1 enabled

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleHttp2EnabledAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = cfg.HTTP2
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "no such file or directory")
	assert.Nil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.MaxConnsPerHost = 0

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithCustomEndpointAndAuthEnabled() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	assert.Nil(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "no such file or directory")
	assert.Nil(testSuite.T(), handleCreated)
}

// This will fail while fetching the token-source, since key-file doesn't exist.
func (testSuite *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNilAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "no such file or directory")
	assert.Nil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenKeyFileIsEmpty() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.KeyFile = ""

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalse() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ReuseTokenFromUrl = false

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.TokenUrl = storageutil.CustomTokenUrl

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithInvalidClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ExperimentalEnableJsonRead = true
	sc.ClientProtocol = "test-protocol"

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), handleCreated)
	assert.Contains(testSuite.T(), err.Error(), "invalid client-protocol requested: test-protocol")
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig()

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientHandle_WithHTTPClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = cfg.HTTP1

	storageClient, err := createGRPCClientHandle(context.Background(), &sc)

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), storageClient)
	assert.Contains(testSuite.T(), err.Error(), fmt.Sprintf("client-protocol requested is not GRPC: %s", cfg.HTTP1))
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_WithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := createHTTPClientHandle(context.Background(), &sc)

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), storageClient)
	assert.Contains(testSuite.T(), err.Error(), fmt.Sprintf("client-protocol requested is not HTTP1 or HTTP2: %s", cfg.GRPC))
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientWithCustomEndpointNilAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil
	sc.AnonymousAccess = false
	sc.ClientProtocol = cfg.GRPC

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "no such file or directory")
	assert.Nil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientWithCustomEndpointAndAuthEnabled() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	assert.Nil(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url
	sc.AnonymousAccess = false
	sc.ClientProtocol = cfg.GRPC

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "GRPC client doesn't support auth for custom-endpoint. Please set anonymous-access: true via config-file.")
	assert.Nil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestCreateStorageHandleWithEnableHNSTrue() {
	sc := storageutil.GetDefaultStorageClientConfig()
	sc.EnableHNS = true

	sh, err := NewStorageHandle(context.Background(), sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh)
}

func (testSuite *StorageHandleTest) TestCreateClientOptionForGRPCClient() {
	sc := storageutil.GetDefaultStorageClientConfig()

	clientOption, err := createClientOptionForGRPCClient(&sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
}

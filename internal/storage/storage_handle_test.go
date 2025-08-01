// Copyright 2022 Google LLC
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
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

var keyFile = "storageutil/testdata/key.json"

type StorageHandleTest struct {
	suite.Suite
	fakeStorage FakeStorage
	mockClient  *MockStorageControlClient
	ctx         context.Context
}

func TestStorageHandleTestSuite(t *testing.T) {
	suite.Run(t, new(StorageHandleTest))
}

func (testSuite *StorageHandleTest) SetupTest() {
	testSuite.mockClient = new(MockStorageControlClient)
	testSuite.fakeStorage = NewFakeStorageWithMockClient(testSuite.mockClient, cfg.HTTP2)
	testSuite.ctx = context.Background()
}

func (testSuite *StorageHandleTest) TearDownTest() {
	testSuite.fakeStorage.ShutDown()
}

func (testSuite *StorageHandleTest) mockStorageLayout(bucketType gcs.BucketType) {
	storageLayout := &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: false},
		LocationType:          "nil",
	}

	if bucketType.Zonal {
		storageLayout.HierarchicalNamespace = &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true}
		storageLayout.LocationType = "zone"
	}

	if bucketType.Hierarchical {
		storageLayout.HierarchicalNamespace = &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true}
		storageLayout.LocationType = "multiregion"
	}

	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).Return(storageLayout, nil)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockStorageLayout(gcs.BucketType{})
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, TestBucketName, "", false)

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), TestBucketName, bucketHandle.bucketName)
	assert.False(testSuite.T(), bucketHandle.bucketType.Zonal)
	assert.False(testSuite.T(), bucketHandle.bucketType.Hierarchical)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("bucket does not exist"))
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, invalidBucketName, "", false)

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bucketHandle)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, TestBucketName, projectID, false)

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), TestBucketName, bucketHandle.bucketName)
	assert.False(testSuite.T(), bucketHandle.bucketType.Zonal)
	assert.True(testSuite.T(), bucketHandle.bucketType.Hierarchical)
	// verify the billing account set.
	testHandle := bucketHandle
	assert.Equal(testSuite.T(), bucketHandle.bucket, testHandle.bucket.UserProject(projectID))
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithNonEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("bucket does not exist"))
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, invalidBucketName, projectID, false)

	assert.Nil(testSuite.T(), bucketHandle)
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile) // by default http1 enabled

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleHttp2EnabledAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTP2
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.MaxConnsPerHost = 0

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenUserAgentIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.UserAgent = "gcsfuse/unknown (Go version go1.20-pre3 cl/474093167 +a813be86df) appName (GPN:Gcsfuse-DLC)"

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithCustomEndpointAndAuthEnabled() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	assert.Nil(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = url.String()
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

// This will fail while fetching the token-source, since key-file doesn't exist.
func (testSuite *StorageHandleTest) TestNewStorageHandleWhenCustomEndpointIsNilAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = ""
	sc.AnonymousAccess = false

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenAnonymousAccessTrue() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.KeyFile = ""
	sc.AnonymousAccess = true

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenReuseTokenUrlFalse() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ReuseTokenFromUrl = false

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenTokenUrlIsSet() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.TokenUrl = storageutil.CustomTokenUrl

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWhenJsonReadEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ExperimentalEnableJsonRead = true

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithBillingProject() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableHNS = true

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, projectID)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
	storageClient, ok := handleCreated.(*storageClient)
	assert.NotNil(testSuite.T(), storageClient)
	assert.True(testSuite.T(), ok)
	// Confirm that the returned storage-handle's control-client is of type storageControlClientWithBillingProject
	// and its billing-project is same as the one passed while
	// creating the storage-handle.
	storageControlClient, ok := storageClient.storageControlClient.(*storageControlClientWithBillingProject)
	assert.NotNil(testSuite.T(), storageControlClient)
	assert.True(testSuite.T(), ok)
	assert.Equal(testSuite.T(), storageControlClient.billingProject, projectID)
	assert.NotNil(testSuite.T(), storageControlClient.raw)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithInvalidClientProtocol() {
	fakeStorage := NewFakeStorageWithMockClient(testSuite.mockClient, "test-protocol")
	testSuite.mockStorageLayout(gcs.BucketType{})
	sh := fakeStorage.CreateStorageHandle()
	assert.NotNil(testSuite.T(), sh)
	bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, projectID, false)

	assert.Nil(testSuite.T(), bh)
	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "invalid client-protocol requested: test-protocol")
}

func (testSuite *StorageHandleTest) TestNewStorageHandleDirectPathDetector() {
	testCases := []struct {
		name           string
		clientProtocol cfg.Protocol
	}{
		{
			name:           "grpcWithNonNilDirectPathDetector",
			clientProtocol: cfg.GRPC,
		},
		{
			name:           "http1WithNilDirectPathDetector",
			clientProtocol: cfg.HTTP1,
		},
		{
			name:           "http2WithNilDirectPathDetector",
			clientProtocol: cfg.HTTP2,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ExperimentalEnableJsonRead = true
			sc.ClientProtocol = tc.clientProtocol

			handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")
			assert.Nil(testSuite.T(), err)
			assert.NotNil(testSuite.T(), handleCreated)

			storageClient, ok := handleCreated.(*storageClient)
			assert.True(testSuite.T(), ok)

			assert.NotNil(testSuite.T(), storageClient.directPathDetector)
		})
	}
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := createGRPCClientHandle(testSuite.ctx, &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientHandleWithBidiConfig() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := createGRPCClientHandle(testSuite.ctx, &sc, true)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientProtocol() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.GRPC

	storageClient, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_WithReadStallRetry() {
	testCases := []struct {
		name                 string
		enableReadStallRetry bool
	}{
		{
			name:                 "ReadStallRetryEnabled",
			enableReadStallRetry: true,
		},
		{
			name:                 "ReadStallRetryDisabled",
			enableReadStallRetry: false,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ReadStallRetryConfig.Enable = tc.enableReadStallRetry

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

			assert.Nil(testSuite.T(), err)
			assert.NotNil(testSuite.T(), storageClient)
		})
	}
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_ReadStallInitialReqTimeout() {
	testCases := []struct {
		name              string
		initialReqTimeout time.Duration
	}{
		{
			name:              "ShortTimeout",
			initialReqTimeout: 1 * time.Millisecond,
		},
		{
			name:              "LongTimeout",
			initialReqTimeout: 10 * time.Second,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ReadStallRetryConfig.Enable = true
			sc.ReadStallRetryConfig.InitialReqTimeout = tc.initialReqTimeout

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

			assert.Nil(testSuite.T(), err)
			assert.NotNil(testSuite.T(), storageClient)
		})
	}
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_ReadStallMinReqTimeout() {
	testCases := []struct {
		name          string
		minReqTimeout time.Duration
	}{
		{
			name:          "ShortTimeout",
			minReqTimeout: 1 * time.Millisecond,
		},
		{
			name:          "LongTimeout",
			minReqTimeout: 10 * time.Second,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ReadStallRetryConfig.Enable = true
			sc.ReadStallRetryConfig.MinReqTimeout = tc.minReqTimeout

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

			assert.Nil(testSuite.T(), err)
			assert.NotNil(testSuite.T(), storageClient)
		})
	}
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_ReadStallReqIncreaseRate() {
	testCases := []struct {
		name            string
		reqIncreaseRate float64
		expectErr       bool
	}{
		{
			name:            "NegativeRate",
			reqIncreaseRate: -0.5,
			expectErr:       true,
		},
		{
			name:            "ZeroRate",
			reqIncreaseRate: 0.0,
			expectErr:       true,
		},
		{
			name:            "PositiveRate",
			reqIncreaseRate: 1.5,
			expectErr:       false,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ReadStallRetryConfig.Enable = true
			sc.ReadStallRetryConfig.ReqIncreaseRate = tc.reqIncreaseRate

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

			if tc.expectErr {
				assert.NotNil(testSuite.T(), err)
			} else {
				assert.Nil(testSuite.T(), err)
				assert.NotNil(testSuite.T(), storageClient)
			}
		})
	}
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle_ReadStallReqTargetPercentile() {
	testCases := []struct {
		name                string
		reqTargetPercentile float64
		expectErr           bool
	}{
		{
			name:                "LowPercentile",
			reqTargetPercentile: 0.25, // 25th percentile
			expectErr:           false,
		},
		{
			name:                "MidPercentile",
			reqTargetPercentile: 0.50, // 50th percentile
			expectErr:           false,
		},
		{
			name:                "HighPercentile",
			reqTargetPercentile: 0.90, // 90th percentile
			expectErr:           false,
		},
		{
			name:                "InvalidPercentile-Low",
			reqTargetPercentile: -0.5, // Invalid percentile
			expectErr:           true,
		},
		{
			name:                "InvalidPercentile-High",
			reqTargetPercentile: 1.5, // Invalid percentile
			expectErr:           true,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ReadStallRetryConfig.Enable = true
			sc.ReadStallRetryConfig.ReqTargetPercentile = tc.reqTargetPercentile

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc)

			if tc.expectErr {
				assert.NotNil(testSuite.T(), err)
			} else {
				assert.Nil(testSuite.T(), err)
				assert.NotNil(testSuite.T(), storageClient)
			}
		})
	}
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientWithCustomEndpointNilAndAuthEnabled() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = ""
	sc.AnonymousAccess = false
	sc.ClientProtocol = cfg.GRPC

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientWithCustomEndpointAndAuthEnabled() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	assert.Nil(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = url.String()
	sc.AnonymousAccess = false
	sc.ClientProtocol = cfg.GRPC
	sc.TokenUrl = storageutil.CustomTokenUrl
	sc.KeyFile = ""

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithGRPCClientWithCustomEndpointAndAuthDisabled() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	assert.Nil(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = url.String()
	sc.ClientProtocol = cfg.GRPC
	sc.TokenUrl = storageutil.CustomTokenUrl
	sc.KeyFile = ""

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
}

func (testSuite *StorageHandleTest) TestCreateStorageHandleWithEnableHNSTrue() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableHNS = true

	sh, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh)
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithCustomEndpointAndEnableHNSTrue() {
	url, err := url.Parse(storageutil.CustomEndpoint)
	require.NoError(testSuite.T(), err)
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.CustomEndpoint = url.String()
	sc.EnableHNS = true

	sh, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh)
}

func (testSuite *StorageHandleTest) TestCreateClientOptionForGRPCClient() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)

	clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithoutGoogleLibAuth() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableGoogleLibAuth = false

	clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
}

func (testSuite *StorageHandleTest) Test_CreateHTTPClientHandle_WithoutGoogleLibAuth() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableGoogleLibAuth = false

	httpClient, err := createHTTPClientHandle(context.TODO(), &sc)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), httpClient)
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_AuthFailures() {
	tests := []struct {
		name          string
		modifyConfig  func(sc *storageutil.StorageClientConfig)
		expectError   bool
		expectNilOpts bool
	}{
		{
			name: "Invalid token URL with google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.TokenUrl = ":"
				sc.KeyFile = ""
				sc.EnableGoogleLibAuth = true
			},
		},
		{
			name: "Invalid token URL without google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.TokenUrl = ":"
				sc.KeyFile = ""
				sc.EnableGoogleLibAuth = false
			},
		},
		{
			name: "Invalid key file path with google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.KeyFile = "incorrect_path"
				sc.EnableGoogleLibAuth = true
			},
		},
		{
			name: "Invalid key file path without google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.KeyFile = "incorrect_path"
				sc.EnableGoogleLibAuth = false
			},
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ClientProtocol = cfg.GRPC
			tt.modifyConfig(&sc)

			clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

			assert.Error(t, err)
			assert.Nil(t, clientOption)
		})
	}
}

func (testSuite *StorageHandleTest) Test_CreateHTTPClientHandle_AuthFailures() {
	tests := []struct {
		name          string
		modifyConfig  func(sc *storageutil.StorageClientConfig)
		expectError   bool
		expectNilOpts bool
	}{
		{
			name: "Invalid token URL with google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.TokenUrl = ":"
				sc.KeyFile = ""
				sc.EnableGoogleLibAuth = true
			},
		},
		{
			name: "Invalid token URL without google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.TokenUrl = ":"
				sc.KeyFile = ""
				sc.EnableGoogleLibAuth = false
			},
		},
		{
			name: "Invalid key file path with google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.KeyFile = "incorrect_path"
				sc.EnableGoogleLibAuth = true
			},
		},
		{
			name: "Invalid Key File Path without google lib auth",
			modifyConfig: func(sc *storageutil.StorageClientConfig) {
				sc.KeyFile = "incorrect_path"
				sc.EnableGoogleLibAuth = false
			},
		},
	}

	for _, tt := range tests {
		testSuite.T().Run(tt.name, func(t *testing.T) {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			tt.modifyConfig(&sc)

			httpClient, err := createHTTPClientHandle(context.TODO(), &sc)

			assert.Error(t, err)
			assert.Nil(t, httpClient)
		})
	}
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithMaxRetryAttemptsNotZero() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.MaxRetryAttempts = 100

	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	if assert.NoError(testSuite.T(), err) {
		assert.NotNil(testSuite.T(), handleCreated)
	}
}

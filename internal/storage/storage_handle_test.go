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
	"net"
	"net/url"
	"testing"
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const invalidBucketName string = "will-not-be-present-in-fake-server"
const projectID string = "valid-project-id"

var keyFile = "storageutil/testdata/key.json"

// A fake implementation of control.StorageControlServer for testing.
type fakeStorageControlServer struct {
	controlpb.UnimplementedStorageControlServer
	// Last received request's peer address.
	remoteAddr net.Addr
}

func (s *fakeStorageControlServer) CreateFolder(ctx context.Context, in *controlpb.CreateFolderRequest) (*controlpb.Folder, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("peer not found")
	}
	s.remoteAddr = p.Addr
	return &controlpb.Folder{}, nil
}

type StorageHandleTest struct {
	suite.Suite
	fakeStorage  FakeStorage
	mockClient   *MockStorageControlClient
	clientConfig *storageutil.StorageClientConfig
	ctx          context.Context
}

func TestStorageHandleTestSuite(t *testing.T) {
	suite.Run(t, new(StorageHandleTest))
}

func (testSuite *StorageHandleTest) SetupTest() {
	testSuite.mockClient = new(MockStorageControlClient)
	sc := storageutil.GetDefaultStorageClientConfig("")
	testSuite.clientConfig = &sc
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

// Helpers

func (testSuite *StorageHandleTest) controlClientCallOptionsWithoutRetry() *control.StorageControlCallOptions {
	testSuite.T().Helper()
	return &control.StorageControlCallOptions{}
}

func (testSuite *StorageHandleTest) controlClientCallOptionsWithRetry() *control.StorageControlCallOptions {
	testSuite.T().Helper()
	clientConfig := &storageutil.StorageClientConfig{MaxRetrySleep: 100 * time.Microsecond, MaxRetryAttempts: 5}
	gaxRetryOptions := storageControlClientGaxRetryOptions(clientConfig)
	return &control.StorageControlCallOptions{
		CreateFolder: gaxRetryOptions,
		GetFolder:    gaxRetryOptions,
		DeleteFolder: gaxRetryOptions,
		RenameFolder: gaxRetryOptions,
	}
}

// Test functions

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

func (testSuite *StorageHandleTest) TestNewStorageHandleWithoutBillingProject() {
	// Arrange.
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableHNS = true

	// Act.
	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, "")

	// Assert.
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
	storageClient, ok := handleCreated.(*storageClient)
	assert.NotNil(testSuite.T(), storageClient)
	assert.True(testSuite.T(), ok)
	// Confirm that the returned storage-handle's control-client is of type storageControlClientWithRetry
	retrierControlClient, ok := storageClient.storageControlClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "retrierControlClient should be of type *storageControlClientWithRetry")
	require.NotNil(testSuite.T(), retrierControlClient, "retrierControlClient should not be nil")
	assert.True(testSuite.T(), retrierControlClient.enableRetriesOnStorageLayoutAPI, "enableRetriesOnStorageLayoutAPI should be true")
	assert.False(testSuite.T(), retrierControlClient.enableRetriesOnFolderAPIs, "enableRetriesOnFolderAPIs should be false")
	// Confirm that it has no underlying storageControlClientWithBillingProject in it.
	_, ok = retrierControlClient.raw.(*storageControlClientWithBillingProject)
	assert.False(testSuite.T(), ok, "raw should be of type *storageControlClientWithBillingProject")
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithBillingProject() {
	// Arrange.
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableHNS = true

	// Act.
	handleCreated, err := NewStorageHandle(testSuite.ctx, sc, projectID)

	// Assert.
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), handleCreated)
	storageClient, ok := handleCreated.(*storageClient)
	assert.NotNil(testSuite.T(), storageClient)
	assert.True(testSuite.T(), ok)
	retrierControlClient := storageClient.storageControlClient.(*storageControlClientWithRetry)
	// Confirm that the returned storage-handle's control-client is of type storageControlClientWithBillingProject
	// and its billing-project is same as the one passed while
	// creating the storage-handle.
	// Check that storageControlClient is wrapped correctly and billing project is set.
	billingProjectControlClient, ok := retrierControlClient.raw.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok, "raw should be of type *storageControlClientWithBillingProject")
	require.NotNil(testSuite.T(), billingProjectControlClient, "storageControlClientWithBillingProject should not be nil")
	assert.Equal(testSuite.T(), projectID, billingProjectControlClient.billingProject, "billingProject should match the provided projectID")
	assert.NotNil(testSuite.T(), billingProjectControlClient.raw, "raw client inside storageControlClientWithBillingProject should not be nil")
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

func (testSuite *StorageHandleTest) TestCreateGRPCClientWithSocketAddress() {
	// Start a local server to inspect incoming connections.
	server := &fakeStorageControlServer{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(testSuite.T(), err)
	serveErr := make(chan error, 1)
	grpcServer := grpc.NewServer()
	controlpb.RegisterStorageControlServer(grpcServer, server)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()
	// Configure the client to use a specific local IP address.
	testSuite.clientConfig.CustomEndpoint = listener.Addr().String()
	testSuite.clientConfig.SocketAddress = "127.0.0.1"
	testSuite.clientConfig.AnonymousAccess = true
	ctx := context.Background()
	clientOpts, err := createClientOptionForGRPCClient(ctx, testSuite.clientConfig, false)
	require.NoError(testSuite.T(), err)
	controlClient, err := storageutil.CreateGRPCControlClient(ctx, clientOpts, false)
	require.NoError(testSuite.T(), err)
	require.NotNil(testSuite.T(), controlClient)

	// Have the client connect to the test server.
	// This will not fail, as we have implemented the "CreateFolder" method.
	_, err = controlClient.CreateFolder(ctx, &controlpb.CreateFolderRequest{})
	assert.NoError(testSuite.T(), err)
	// Stop the server and check for any serving errors.
	grpcServer.Stop()
	err = <-serveErr
	if err != nil && err != grpc.ErrServerStopped {
		testSuite.T().Fatalf("grpcServer.Serve failed: %v", err)
	}
	// The defer call will also try to stop, which is fine.

	// Verify on the server side that the client's connection originates from the specified IP address.
	host, _, err := net.SplitHostPort(server.remoteAddr.String())
	require.NoError(testSuite.T(), err)
	assert.Equal(testSuite.T(), testSuite.clientConfig.SocketAddress, host)
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientWithInvalidSocketAddress() {
	// Configure the client to use an invalid local IP address.
	testSuite.clientConfig.SocketAddress = "invalid-address"
	testSuite.clientConfig.AnonymousAccess = true
	ctx := context.Background()

	// Attempt to create client options, which should fail.
	clientOpts, err := createClientOptionForGRPCClient(ctx, testSuite.clientConfig, false)

	assert.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), clientOpts)
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

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithTracing() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.TracingEnabled = true

	clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithTracingAddsOneOption() {
	scWithoutTracing := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithoutTracing.TracingEnabled = false
	optsWithoutTracing, err := createClientOptionForGRPCClient(context.TODO(), &scWithoutTracing, false)
	assert.Nil(testSuite.T(), err)
	scWithTracing := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithTracing.TracingEnabled = true

	optsWithTracing, err := createClientOptionForGRPCClient(context.TODO(), &scWithTracing, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), optsWithTracing)
	assert.Len(testSuite.T(), optsWithTracing, len(optsWithoutTracing)+1, "Enabling tracing should add exactly one client option.")
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

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_NilControlClient() {
	// Arrange
	sh := &storageClient{} // storageControlClient is nil by default

	// Act
	controlClient := sh.controlClientForBucketHandle(&gcs.BucketType{}, "")

	// Assert
	assert.Nil(testSuite.T(), controlClient)
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_ZonalBucket_NoBillingProject() {
	// Arrange
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh := &storageClient{
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		clientConfig:                             clientConfig,
	}
	bucketType := &gcs.BucketType{Zonal: true}

	// Act
	controlClient := sh.controlClientForBucketHandle(bucketType, "")

	// Assert
	require.NotNil(testSuite.T(), controlClient)
	retryWrapper, ok := controlClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "Expected a retry wrapper for zonal bucket")
	assert.Same(testSuite.T(), mockRawControlClientWithoutRetries, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for all APIs on zonal buckets")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs on zonal buckets")
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_ZonalBucket_WithBillingProject() {
	// Arrange
	billingProject := "test-project"
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh := &storageClient{
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		clientConfig:                             clientConfig,
	}
	bucketType := &gcs.BucketType{Zonal: true}

	// Act
	controlClient := sh.controlClientForBucketHandle(bucketType, billingProject)

	// Assert
	require.NotNil(testSuite.T(), controlClient)
	// Verify that the returned control-client wraps over the billing project, then storageControlClientWithRetry, in that order.
	billingProjectWrapper, ok := controlClient.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok, "Expected a billing project wrapper")
	assert.Equal(testSuite.T(), billingProject, billingProjectWrapper.billingProject)
	retryWrapper, ok := billingProjectWrapper.raw.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "Expected a retry wrapper for zonal bucket")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs on zonal buckets")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs on zonal buckets")
	assert.Same(testSuite.T(), mockRawControlClientWithoutRetries, retryWrapper.raw)
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_NonZonalBucket_WithoutBillingProject() {
	// Arrange
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh := &storageClient{
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		clientConfig:                             clientConfig,
	}
	bucketType := &gcs.BucketType{Zonal: false}

	// Act
	controlClient := sh.controlClientForBucketHandle(bucketType, "")

	// Assert
	require.NotNil(testSuite.T(), controlClient)
	// It should be the raw client with GAX retries, not the billing project wrapper.
	_, isBillingWrapper := controlClient.(*storageControlClientWithBillingProject)
	assert.False(testSuite.T(), isBillingWrapper)
	// It should be an enhanced storage control client with retries for GetStorageLayout.
	controlClientWithStorageLayoutRetries, ok := controlClient.(*storageControlClientWithRetry)
	assert.True(testSuite.T(), ok, "Expected a control client with retry")
	assert.NotNil(testSuite.T(), controlClientWithStorageLayoutRetries)
	assert.True(testSuite.T(), controlClientWithStorageLayoutRetries.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on non-zonal buckets")
	assert.False(testSuite.T(), controlClientWithStorageLayoutRetries.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs on non-zonal buckets")
	// Check if it's the GAX-retries-added client
	gaxClient, ok := controlClientWithStorageLayoutRetries.raw.(*control.StorageControlClient)
	require.True(testSuite.T(), ok)
	require.Same(testSuite.T(), mockRawControlClientWithRetries, gaxClient)
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_NonZonalBucket_WithBillingProject() {
	// Arrange
	billingProject := "test-project"
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockControlClient := withBillingProject(mockRawControlClientWithRetries, billingProject)
	mockControlClient = withRetryOnStorageLayout(mockControlClient, &clientConfig)
	sh := &storageClient{
		storageControlClient:                     mockControlClient,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		clientConfig:                             clientConfig,
	}
	bucketType := &gcs.BucketType{Zonal: false}

	// Act
	controlClient := sh.controlClientForBucketHandle(bucketType, billingProject)

	// Assert
	require.NotNil(testSuite.T(), controlClient)
	billingWrapper, ok := controlClient.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok, "Expected a billing project wrapper")
	assert.Equal(testSuite.T(), billingProject, billingWrapper.billingProject)
	// Check that the underlying control client is a storageControlClientWithRetry and also uses GAX retries.
	controlClientWithAllRetriesNonZB, ok := billingWrapper.raw.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), controlClientWithAllRetriesNonZB)
	assert.True(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on non-zonal buckets")
	assert.False(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs on non-zonal buckets")
	// Check that the inner client has GAX retries
	gaxClient, ok := controlClientWithAllRetriesNonZB.raw.(*control.StorageControlClient)
	require.True(testSuite.T(), ok)
	require.Same(testSuite.T(), mockRawControlClientWithRetries, gaxClient)
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_NonZonalBucket_ThenZonalBucket_WithoutBillingProject() {
	// Arrange
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh := &storageClient{
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		clientConfig:                             clientConfig,
	}

	// Act
	// create control-client for non-ZB, which should create a control-client with gax retries.
	bucketType := gcs.BucketType{Zonal: false}
	controlClientForNonZB := sh.controlClientForBucketHandle(&bucketType, "")

	// Assert
	require.NotNil(testSuite.T(), controlClientForNonZB)
	// Check that the raw control client is a storageControlClientWithRetry and also uses GAX retries.
	controlClientWithAllRetriesNonZB, ok := controlClientForNonZB.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), controlClientWithAllRetriesNonZB)
	assert.True(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on non-zonal buckets")
	assert.False(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs on non-zonal buckets")
	require.Same(testSuite.T(), mockRawControlClientWithRetries, controlClientWithAllRetriesNonZB.raw)

	// Act
	// create control-client for ZB afterwards, which should create a storageControlClientWithRetry a raw control.StorageControlClient without gax retries.
	bucketType = gcs.BucketType{Zonal: true}
	controlClientForZB := sh.controlClientForBucketHandle(&bucketType, "")

	// Assert
	require.NotNil(testSuite.T(), controlClientForZB)
	// Check that the control client is a storageControlClientWithRetry with all APIs retried.
	controlClientWithRetry, ok := controlClientForZB.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "Expected a control client with retry")
	assert.Same(testSuite.T(), mockRawControlClientWithoutRetries, controlClientWithRetry.raw)
	assert.True(testSuite.T(), controlClientWithRetry.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs on zonal buckets")
	assert.True(testSuite.T(), controlClientWithRetry.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on zonal buckets")
	require.Same(testSuite.T(), mockRawControlClientWithoutRetries, controlClientWithRetry.raw)
}

func (testSuite *StorageHandleTest) TestControlClientForBucketHandle_NonZonalBucket_ThenZonalBucket_WithBillingProject() {
	// Arrange
	billingProject := "test-project"
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	mockRawControlClientWithRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithRetry()}
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh := &storageClient{
		rawStorageControlClientWithoutGaxRetries: mockRawControlClientWithoutRetries,
		rawStorageControlClientWithGaxRetries:    mockRawControlClientWithRetries,
		clientConfig:                             clientConfig,
	}

	// Act
	// create control-client for non-ZB, which should create a control-client with gax retries.
	bucketType := gcs.BucketType{Zonal: false}
	controlClientForNonZB := sh.controlClientForBucketHandle(&bucketType, billingProject)

	// Assert
	require.NotNil(testSuite.T(), controlClientForNonZB)
	// Check that the control client is not a storageControlClientWithRetry and uses GAX retries.
	controlClientWithBillingProjectAndAllRetriesForNonZB, ok := controlClientForNonZB.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), controlClientWithBillingProjectAndAllRetriesForNonZB)
	assert.Equal(testSuite.T(), billingProject, controlClientWithBillingProjectAndAllRetriesForNonZB.billingProject)
	// Check that the underlying control client is a storageControlClientWithRetry and also uses GAX retries.
	controlClientWithAllRetriesNonZB, ok := controlClientWithBillingProjectAndAllRetriesForNonZB.raw.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), controlClientWithAllRetriesNonZB)
	assert.True(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on non-zonal buckets")
	assert.False(testSuite.T(), controlClientWithAllRetriesNonZB.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs on non-zonal buckets")
	require.Same(testSuite.T(), mockRawControlClientWithRetries, controlClientWithAllRetriesNonZB.raw)

	// Act
	// create control-client for ZB afterwards, which should create a storageControlClientWithRetry a raw control.StorageControlClient without gax retries.
	bucketType = gcs.BucketType{Zonal: true}
	controlClientForZB := sh.controlClientForBucketHandle(&bucketType, billingProject)

	// Assert
	require.NotNil(testSuite.T(), controlClientForZB)
	// Check that the control client contains a storageControlClientWithBillingProject.
	controlClientWithBillingProjectForZB, ok := controlClientForZB.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), controlClientWithBillingProjectForZB)
	// Check that the control client is a storageControlClientWithRetry with all APIs retried.
	controlClientWithRetry, ok := controlClientWithBillingProjectForZB.raw.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "Expected a control client with retry")
	require.NotNil(testSuite.T(), controlClientWithRetry)
	assert.True(testSuite.T(), controlClientWithRetry.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs on zonal buckets")
	assert.True(testSuite.T(), controlClientWithRetry.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout API on zonal buckets")
	assert.Same(testSuite.T(), mockRawControlClientWithoutRetries, controlClientWithRetry.raw)
}

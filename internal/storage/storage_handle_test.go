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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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

	if bucketType.RCU != gcs.RCUStateNone {
		storageLayout.RapidCacheInfo = &controlpb.StorageLayout_RapidCacheInfo{
			CacheType: rapidCacheUltraType,
		}
	}

	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).Return(storageLayout, nil)
}

// Helpers

func (testSuite *StorageHandleTest) controlClientCallOptionsWithoutRetry() *control.StorageControlCallOptions {
	testSuite.T().Helper()
	return &control.StorageControlCallOptions{}
}

// Test functions

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockStorageLayout(gcs.BucketType{})
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), TestBucketName, bucketHandle.bucketName)
	assert.False(testSuite.T(), bucketHandle.bucketType.Zonal)
	assert.False(testSuite.T(), bucketHandle.bucketType.Hierarchical)
}

func (testSuite *StorageHandleTest) TestBucketHandle_DisableGrpcReadChecksums() {
	tests := []struct {
		name                             string
		enableGrpcReadChecksums          bool
		bucketTypeRapid                  bool
		clientProtocol                   cfg.Protocol
		expectedDisableGrpcReadChecksums bool
	}{
		{
			name:                             "EnableGrpcReadChecksums is true, bucket is rapid, protocol is grpc",
			enableGrpcReadChecksums:          true,
			bucketTypeRapid:                  true,
			clientProtocol:                   cfg.GRPC,
			expectedDisableGrpcReadChecksums: false,
		},
		{
			name:                             "EnableGrpcReadChecksums is true, bucket is rapid, protocol is http1",
			enableGrpcReadChecksums:          true,
			bucketTypeRapid:                  true,
			clientProtocol:                   cfg.HTTP1,
			expectedDisableGrpcReadChecksums: false,
		},
		{
			name:                             "EnableGrpcReadChecksums is true, bucket is standard, protocol is grpc",
			enableGrpcReadChecksums:          true,
			bucketTypeRapid:                  false,
			clientProtocol:                   cfg.GRPC,
			expectedDisableGrpcReadChecksums: false,
		},
		{
			name:                             "EnableGrpcReadChecksums is true, bucket is standard, protocol is http1",
			enableGrpcReadChecksums:          true,
			bucketTypeRapid:                  false,
			clientProtocol:                   cfg.HTTP1,
			expectedDisableGrpcReadChecksums: true,
		},
		{
			name:                             "EnableGrpcReadChecksums is false, bucket is rapid, protocol is grpc",
			enableGrpcReadChecksums:          false,
			bucketTypeRapid:                  true,
			clientProtocol:                   cfg.GRPC,
			expectedDisableGrpcReadChecksums: true,
		},
		{
			name:                             "EnableGrpcReadChecksums is false, bucket is rapid, protocol is http1",
			enableGrpcReadChecksums:          false,
			bucketTypeRapid:                  true,
			clientProtocol:                   cfg.HTTP1,
			expectedDisableGrpcReadChecksums: true,
		},
		{
			name:                             "EnableGrpcReadChecksums is false, bucket is standard, protocol is grpc",
			enableGrpcReadChecksums:          false,
			bucketTypeRapid:                  false,
			clientProtocol:                   cfg.GRPC,
			expectedDisableGrpcReadChecksums: true,
		},
		{
			name:                             "EnableGrpcReadChecksums is false, bucket is standard, protocol is http1",
			enableGrpcReadChecksums:          false,
			bucketTypeRapid:                  false,
			clientProtocol:                   cfg.HTTP1,
			expectedDisableGrpcReadChecksums: true,
		},
	}

	for _, tc := range tests {
		testSuite.T().Run(tc.name, func(t *testing.T) {
			// Reinitialize mock client and fake storage for each subtest
			testSuite.mockClient = new(MockStorageControlClient)
			testSuite.fakeStorage = NewFakeStorageWithMockClient(testSuite.mockClient, cfg.HTTP2)
			storageHandle := testSuite.fakeStorage.CreateStorageHandle()
			sh := storageHandle.(*storageClient)
			sh.grpcClient = &storage.Client{}
			sh.clientConfig.EnableGrpcReadChecksums = tc.enableGrpcReadChecksums
			sh.clientConfig.ClientProtocol = tc.clientProtocol
			bucketType := gcs.BucketType{
				Zonal: tc.bucketTypeRapid,
			}
			testSuite.mockStorageLayout(bucketType)

			bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedDisableGrpcReadChecksums, bh.disableGrpcReadChecksums)
		})
	}
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExistWithEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("bucket does not exist"))
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, invalidBucketName, "")

	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bucketHandle)
}

func (testSuite *StorageHandleTest) TestBucketHandleWhenBucketExistsWithNonEmptyBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, TestBucketName, projectID)

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
	bucketHandle, err := storageHandle.BucketHandle(testSuite.ctx, invalidBucketName, projectID)

	assert.Nil(testSuite.T(), bucketHandle)
	assert.NotNil(testSuite.T(), err)
}

func (testSuite *StorageHandleTest) TestBucketHandle_ControlClientIsWrappedWithRetry() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	sh := storageHandle.(*storageClient)
	sh.clientConfig.EnableHNS = true
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh.rawStorageControlClient = mockRawControlClientWithoutRetries
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	controlClient := bucketHandle.controlClient
	require.NotNil(testSuite.T(), controlClient)
	retryWrapper, ok := controlClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "Expected a retry wrapper")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs)
}

func (testSuite *StorageHandleTest) TestBucketHandle_ControlClientIsWrappedWithBillingProject() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	sh := storageHandle.(*storageClient)
	sh.clientConfig.EnableHNS = true
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh.rawStorageControlClient = mockRawControlClientWithoutRetries
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := sh.BucketHandle(testSuite.ctx, TestBucketName, projectID)

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	controlClient := bucketHandle.controlClient
	require.NotNil(testSuite.T(), controlClient)
	billingProjectWrapper, ok := controlClient.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok, "Expected a billing project wrapper")
	assert.Equal(testSuite.T(), projectID, billingProjectWrapper.billingProject)
}

func (testSuite *StorageHandleTest) TestBucketHandle_FlatBucket_ClosesControlClient() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	sh := storageHandle.(*storageClient)
	sh.clientConfig.EnableHNS = true
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh.rawStorageControlClient = mockRawControlClientWithoutRetries
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: false})

	bucketHandle, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bucketHandle.controlClient, "Flat bucket handle should not retain a control client")
	assert.Nil(testSuite.T(), sh.rawStorageControlClient, "Flat bucket mount should close and clear rawStorageControlClient")
	assert.Nil(testSuite.T(), sh.storageControlClient, "Flat bucket mount should clear storageControlClient")
}

func (testSuite *StorageHandleTest) TestBucketHandle_HNSBucket_EnableHNSFalse_ClosesControlClient() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	sh := storageHandle.(*storageClient)
	sh.clientConfig.EnableHNS = false
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh.rawStorageControlClient = mockRawControlClientWithoutRetries
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.Nil(testSuite.T(), bucketHandle.controlClient, "When EnableHNS is false, control client must not be retained")
	assert.Nil(testSuite.T(), sh.rawStorageControlClient, "When EnableHNS is false, rawStorageControlClient must be closed and cleared")
	assert.Nil(testSuite.T(), sh.storageControlClient, "When EnableHNS is false, storageControlClient must be cleared")
}

func (testSuite *StorageHandleTest) TestBucketHandle_HNSBucket_RetainsControlClient() {
	storageHandle := testSuite.fakeStorage.CreateStorageHandle()
	sh := storageHandle.(*storageClient)
	sh.clientConfig.EnableHNS = true
	mockRawControlClientWithoutRetries := &control.StorageControlClient{CallOptions: testSuite.controlClientCallOptionsWithoutRetry()}
	sh.rawStorageControlClient = mockRawControlClientWithoutRetries
	testSuite.mockStorageLayout(gcs.BucketType{Hierarchical: true})

	bucketHandle, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	assert.NotNil(testSuite.T(), bucketHandle)
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), bucketHandle.controlClient, "HNS bucket handle must retain a control client")
	assert.NotNil(testSuite.T(), sh.rawStorageControlClient, "HNS bucket mount must retain rawStorageControlClient")
}

func (testSuite *StorageHandleTest) TestLookupBucketType_RapidCacheInfo() {
	testCases := []struct {
		name            string
		writeConfig     *cfg.WriteConfig
		rapidCacheInfo  *controlpb.StorageLayout_RapidCacheInfo
		expectedRCU     gcs.RCUState
		expectedIsRapid bool
	}{
		{
			name:            "rapid-cache-ultra with RapidWrite enabled",
			writeConfig:     &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			rapidCacheInfo:  &controlpb.StorageLayout_RapidCacheInfo{CacheType: rapidCacheUltraType},
			expectedRCU:     gcs.RCUStateRapidWritesEnabled,
			expectedIsRapid: true,
		},
		{
			name:            "rapid-cache-ultra with RapidWrite disabled",
			writeConfig:     &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyDisabled},
			rapidCacheInfo:  &controlpb.StorageLayout_RapidCacheInfo{CacheType: rapidCacheUltraType},
			expectedRCU:     gcs.RCUStateRapidWritesDisabled,
			expectedIsRapid: true,
		},
		{
			name:            "rapid-cache-ultra with nil WriteConfig defaults to disabled",
			writeConfig:     nil,
			rapidCacheInfo:  &controlpb.StorageLayout_RapidCacheInfo{CacheType: rapidCacheUltraType},
			expectedRCU:     gcs.RCUStateRapidWritesDisabled,
			expectedIsRapid: true,
		},
		{
			name:            "non-ultra rapid-cache is not rcu",
			writeConfig:     &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			rapidCacheInfo:  &controlpb.StorageLayout_RapidCacheInfo{CacheType: "rapid-cache"},
			expectedRCU:     gcs.RCUStateNone,
			expectedIsRapid: false,
		},
		{
			name:            "empty CacheType is not rcu",
			writeConfig:     &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			rapidCacheInfo:  &controlpb.StorageLayout_RapidCacheInfo{CacheType: ""},
			expectedRCU:     gcs.RCUStateNone,
			expectedIsRapid: false,
		},
		{
			name:            "nil RapidCacheInfo is not rcu",
			writeConfig:     &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			rapidCacheInfo:  nil,
			expectedRCU:     gcs.RCUStateNone,
			expectedIsRapid: false,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.WriteConfig = tc.writeConfig
			sh, err := NewStorageHandle(testSuite.ctx, sc, "")
			require.NoError(testSuite.T(), err)
			client := sh.(*storageClient)
			mockControlClient := new(MockStorageControlClient)
			client.storageControlClient = mockControlClient

			storageLayout := &controlpb.StorageLayout{
				LocationType: "region",
				HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{
					Enabled: false,
				},
				RapidCacheInfo: tc.rapidCacheInfo,
			}
			mockControlClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).Return(storageLayout, nil).Once()

			bt, err := client.lookupBucketType(TestBucketName)

			assert.NoError(testSuite.T(), err)
			assert.Equal(testSuite.T(), tc.expectedRCU, bt.RCU)
			assert.Equal(testSuite.T(), tc.expectedIsRapid, bt.IsRapid())
		})
	}
}

func (testSuite *StorageHandleTest) runLookupBucketTypeTest(onlyDirInput, expectedPrefixInRequest string) {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.OnlyDir = onlyDirInput
	sh, err := NewStorageHandle(testSuite.ctx, sc, "")
	require.NoError(testSuite.T(), err)
	client := sh.(*storageClient)
	client.storageControlClient = testSuite.mockClient
	storageLayout := &controlpb.StorageLayout{
		HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
		LocationType:          "regional",
	}
	// Verify that the prefix passed to GetStorageLayout matches expected value.
	testSuite.mockClient.On("GetStorageLayout", mock.Anything, mock.MatchedBy(func(req *controlpb.GetStorageLayoutRequest) bool {
		return req.Prefix == expectedPrefixInRequest && req.Name == fmt.Sprintf("projects/_/buckets/%s/storageLayout", TestBucketName)
	}), mock.Anything).Return(storageLayout, nil)

	bt, err := client.lookupBucketType(TestBucketName)

	assert.NoError(testSuite.T(), err)
	assert.True(testSuite.T(), bt.Hierarchical)
}

func (testSuite *StorageHandleTest) TestLookupBucketType_WithPrefix() {
	testSuite.runLookupBucketTypeTest("foo/bar", "foo/bar/")
}

func (testSuite *StorageHandleTest) TestLookupBucketType_MountConfigsMetadata() {
	testCases := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "HeaderSet",
			header:   "test-config-payload",
			expected: []string{"test-config-payload"},
		},
		{
			name:     "HeaderEmpty",
			header:   "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.MountConfigsHeader = tc.header
			sh, err := NewStorageHandle(testSuite.ctx, sc, "")
			require.NoError(testSuite.T(), err)
			client := sh.(*storageClient)
			mockClient := new(MockStorageControlClient)
			client.storageControlClient = mockClient
			var got []string
			mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					md, _ := metadata.FromOutgoingContext(args.Get(0).(context.Context))
					got = md.Get("x-goog-storage-go-features")
				}).
				Return(&controlpb.StorageLayout{}, nil)

			_, err = client.lookupBucketType(TestBucketName)

			assert.NoError(testSuite.T(), err)
			assert.Equal(testSuite.T(), tc.expected, got)
		})
	}
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
	// Confirm that the returned storage-handle's control-client is of type storageControlClientWithBillingProject
	// and its billing-project is same as the one passed while
	// creating the storage-handle.
	// Check that storageControlClient is wrapped correctly and billing project is set.
	billingProjectControlClient, ok := storageClient.storageControlClient.(*storageControlClientWithBillingProject)
	require.True(testSuite.T(), ok, "storageControlClient should be of type *storageControlClientWithBillingProject")
	require.NotNil(testSuite.T(), billingProjectControlClient, "storageControlClientWithBillingProject should not be nil")
	assert.Equal(testSuite.T(), projectID, billingProjectControlClient.billingProject, "billingProject should match the provided projectID")
	retrierControlClient, ok := billingProjectControlClient.raw.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "raw should be of type *storageControlClientWithRetry")
	require.NotNil(testSuite.T(), retrierControlClient, "retrierControlClient should not be nil")
	assert.NotNil(testSuite.T(), retrierControlClient.raw, "raw client inside storageControlClientWithRetry should not be nil")
}

func (testSuite *StorageHandleTest) TestNewStorageHandleWithInvalidClientProtocol() {
	fakeStorage := NewFakeStorageWithMockClient(testSuite.mockClient, "test-protocol")
	testSuite.mockStorageLayout(gcs.BucketType{})
	sh := fakeStorage.CreateStorageHandle()
	defer fakeStorage.ShutDown()
	assert.NotNil(testSuite.T(), sh)
	bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, projectID)

	assert.Nil(testSuite.T(), bh)
	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "invalid client-protocol requested: test-protocol")
}

func (testSuite *StorageHandleTest) TestUnSetDirectPathEnvVariable() {
	// Set the environment variable
	testSuite.T().Setenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS", "true")

	// Call the function
	unSetDirectPathEnvVariable()

	// Verify the environment variable is unset
	_, isSet := os.LookupEnv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS")
	assert.False(testSuite.T(), isSet)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandle() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandleWithAnonymousAccess() {
	sc := storageutil.GetDefaultStorageClientConfig("incorrect_path")
	sc.AnonymousAccess = true

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandleWithHTTPMtls_LibAuthTrue() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTPMtls
	sc.EnableGoogleLibAuth = true

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandleWithHTTPMtls_LibAuthFalse_ValidKey() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.HTTPMtls
	sc.EnableGoogleLibAuth = false

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), storageClient)
}

func (testSuite *StorageHandleTest) TestCreateHTTPClientHandleWithHTTPMtls_LibAuthFalse_InvalidKey() {
	sc := storageutil.GetDefaultStorageClientConfig("non-existent-key.json")
	sc.ClientProtocol = cfg.HTTPMtls
	sc.EnableGoogleLibAuth = false

	storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "while fetching tokenSource")
	assert.Nil(testSuite.T(), storageClient)
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
	testSuite.clientConfig.LocalSocketAddress = "127.0.0.1"
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
	assert.Equal(testSuite.T(), testSuite.clientConfig.LocalSocketAddress, host)
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientWithInvalidSocketAddress() {
	// Configure the client to use an invalid local IP address.
	testSuite.clientConfig.LocalSocketAddress = "invalid-address"
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

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

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

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

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

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

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

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

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

			storageClient, err := createHTTPClientHandle(testSuite.ctx, &sc, sc.ClientProtocol)

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

func (testSuite *StorageHandleTest) TestNewStorageHandleWithEnableMountRetries() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.EnableHNS = true
	sc.EnableMountRetries = true

	sh, err := NewStorageHandle(testSuite.ctx, sc, "")

	assert.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh)
	// Underlying storageClient controlClient should enable retries on mount when EnableMountRetries is true.
	scClient, ok := sh.(*storageClient)
	require.True(testSuite.T(), ok)
	require.NotNil(testSuite.T(), scClient.storageControlClient)
	sccwros, ok := scClient.storageControlClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok)
	assert.True(testSuite.T(), sccwros.enableRetriesOnMount)
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

	httpClient, err := createHTTPClientHandle(context.TODO(), &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), httpClient)
}

func (testSuite *StorageHandleTest) Test_CreateHTTPClientHandle_WithS2A() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.S2AAddress = "localhost:8080"
	sc.S2ASpiffeID = "spiffe://example.com/sa/test-sa"

	httpClient, err := createHTTPClientHandle(context.TODO(), &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), httpClient)
}

func (testSuite *StorageHandleTest) Test_CreateHTTPClientHandle_WithoutS2A_EmptyAddress() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.S2AAddress = ""
	sc.S2ASpiffeID = ""

	httpClient, err := createHTTPClientHandle(context.TODO(), &sc, sc.ClientProtocol)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), httpClient)
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithS2A() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.S2AAddress = "localhost:8080"
	sc.S2ASpiffeID = "spiffe://example.com/sa/test-sa"

	clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithoutS2A_EmptyAddress() {
	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.S2AAddress = ""
	sc.S2ASpiffeID = ""

	clientOption, err := createClientOptionForGRPCClient(context.TODO(), &sc, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), clientOption)
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

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithReadStallRetryAddsOneOption() {
	scWithoutReadStall := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithoutReadStall.ReadStallRetryConfig.Enable = false
	optsWithoutReadStall, err := createClientOptionForGRPCClient(context.TODO(), &scWithoutReadStall, false)
	assert.Nil(testSuite.T(), err)
	scWithReadStall := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithReadStall.ReadStallRetryConfig.Enable = true

	optsWithReadStall, err := createClientOptionForGRPCClient(context.TODO(), &scWithReadStall, false)

	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), optsWithReadStall)
	assert.Len(testSuite.T(), optsWithReadStall, len(optsWithoutReadStall)+1, "Enabling read stall retry should add exactly one client option.")
}

func (testSuite *StorageHandleTest) TestCreateGRPCClientHandle_WithReadStallRetry() {
	testCases := []struct {
		name      string
		modifyCfg func(*cfg.ReadStallGcsRetriesConfig)
		expectErr bool
	}{
		{
			name:      "ReadStallRetryDisabled",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) { c.Enable = false },
			expectErr: false,
		},
		{
			name:      "ReadStallRetryEnabled",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) { c.Enable = true },
			expectErr: false,
		},
		{
			name: "ShortTimeouts",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.InitialReqTimeout = 1 * time.Millisecond
				c.MinReqTimeout = 1 * time.Millisecond
			},
			expectErr: false,
		},
		{
			name: "LongTimeouts",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.InitialReqTimeout = 10 * time.Second
				c.MinReqTimeout = 10 * time.Second
			},
			expectErr: false,
		},
		{
			name: "PositiveIncreaseRate",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqIncreaseRate = 1.5
			},
			expectErr: false,
		},
		{
			name: "NegativeIncreaseRate",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqIncreaseRate = -0.5
			},
			expectErr: true,
		},
		{
			name: "ZeroIncreaseRate",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqIncreaseRate = 0.0
			},
			expectErr: true,
		},
		{
			name: "ValidTargetPercentile",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqTargetPercentile = 0.50
			},
			expectErr: false,
		},
		{
			name: "InvalidTargetPercentile_Low",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqTargetPercentile = -0.5
			},
			expectErr: true,
		},
		{
			name: "InvalidTargetPercentile_High",
			modifyCfg: func(c *cfg.ReadStallGcsRetriesConfig) {
				c.ReqTargetPercentile = 1.5
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sc := storageutil.GetDefaultStorageClientConfig(keyFile)
			sc.ClientProtocol = cfg.GRPC
			sc.ReadStallRetryConfig.Enable = true
			tc.modifyCfg(&sc.ReadStallRetryConfig)

			storageClient, err := createGRPCClientHandle(testSuite.ctx, &sc, false, false, TestBucketName, "")
			if storageClient != nil {
				defer func() { _ = storageClient.Close() }()
			}

			if tc.expectErr {
				assert.NotNil(testSuite.T(), err)
			} else {
				assert.Nil(testSuite.T(), err)
				assert.NotNil(testSuite.T(), storageClient)
			}
		})
	}
}

func (testSuite *StorageHandleTest) Test_CreateClientOptionForGRPCClient_WithGrpcMetrics() {
	oldProvider := otel.GetMeterProvider()
	defer otel.SetMeterProvider(oldProvider)

	sdkProvider := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(sdkProvider)

	scWithMetrics := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithMetrics.ClientProtocol = cfg.GRPC
	scWithMetrics.EnableGrpcMetrics = true

	optsWithMetrics, err := createClientOptionForGRPCClient(context.TODO(), &scWithMetrics, false)
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), optsWithMetrics)

	scWithoutMetrics := storageutil.GetDefaultStorageClientConfig(keyFile)
	scWithoutMetrics.ClientProtocol = cfg.GRPC
	scWithoutMetrics.EnableGrpcMetrics = false
	optsWithoutMetrics, err := createClientOptionForGRPCClient(context.TODO(), &scWithoutMetrics, false)
	require.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), optsWithoutMetrics)
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

			httpClient, err := createHTTPClientHandle(context.TODO(), &sc, sc.ClientProtocol)

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
func (testSuite *StorageHandleTest) TestBucketHandle_NonHNS_NilStorageControlClient_SucceedsForValidBucket() {
	sh := testSuite.fakeStorage.CreateStorageHandle().(*storageClient)
	sh.storageControlClient = nil // HNS disabled path
	sh.clientConfig.EnableMountRetries = true

	bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	require.NoError(testSuite.T(), err)
	require.NotNil(testSuite.T(), bh)
	assert.False(testSuite.T(), bh.bucketType.Hierarchical)
	assert.False(testSuite.T(), bh.bucketType.Zonal)
}

// createMockServer creates a test server that returns a fixed response and tracks the number of attempts.
func createMockServer(status int, body string) (*httptest.Server, *int) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	return server, &attempts
}

const testMaxRetryAttempts = 3

// setupMockStorageClientWithMountRetries initializes a storageClient configured with custom endpoint and mount retries enabled for testing.
func (testSuite *StorageHandleTest) setupMockStorageClientWithMountRetries(endpoint string) (*storageClient, func()) {
	sc, err := storage.NewClient(testSuite.ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	require.NoError(testSuite.T(), err)

	sh := &storageClient{
		httpClient:           sc,
		storageControlClient: nil,
		clientConfig: storageutil.StorageClientConfig{
			ClientProtocol:     cfg.HTTP1,
			EnableMountRetries: true,
			MaxRetrySleep:      time.Microsecond,
			RetryMultiplier:    1.0,
			MaxRetryAttempts:   testMaxRetryAttempts,
		},
	}

	cleanup := func() { _ = sc.Close() }
	return sh, cleanup
}

func (testSuite *StorageHandleTest) TestBucketHandle_NonHNS_AccessCheck_ExhaustsRetriesOnBucketNotFound() {
	server, attempts := createMockServer(http.StatusNotFound, `{"error": {"code": 404, "message": "bucket does not exist"}}`)
	defer server.Close()

	sh, cleanup := testSuite.setupMockStorageClientWithMountRetries(server.URL)
	defer cleanup()

	bh, err := sh.BucketHandle(testSuite.ctx, invalidBucketName, "")

	require.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), bh)
	assert.ErrorContains(testSuite.T(), err, "bucket access check failed")
	assert.Equal(testSuite.T(), testMaxRetryAttempts, *attempts, "expected max retry attempts to be exhausted")
}

func (testSuite *StorageHandleTest) TestBucketHandle_NonHNS_AccessCheck_ExhaustsRetriesOnForbidden() {
	server, attempts := createMockServer(http.StatusForbidden, `{"error": {"code": 403, "message": "Permission denied"}}`)
	defer server.Close()

	sh, cleanup := testSuite.setupMockStorageClientWithMountRetries(server.URL)
	defer cleanup()

	bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	require.Error(testSuite.T(), err)
	assert.Nil(testSuite.T(), bh)
	assert.ErrorContains(testSuite.T(), err, "bucket access check failed")
	assert.Equal(testSuite.T(), testMaxRetryAttempts, *attempts, "expected max retry attempts to be exhausted")
}

// TestBucketHandle_NonHNS_AccessCheck_WithPrefix verifies that when GCSFuse mounts
// a non-HNS bucket with a prefix restriction (--only-dir), the mount-time access
// check (which performs a Stat/Attrs call on a non-existent object to verify bucket
// existence and access) is correctly prefixed. This ensures the check does not
// attempt to access the root of the bucket, which would fail with 403 Permission Denied
// in prefix-restricted environments.
func (testSuite *StorageHandleTest) TestBucketHandle_NonHNS_AccessCheck_WithPrefix() {
	testObject := "gcsfuse-nonexistent-object-check"
	prefix := "foo/bar"
	// We expect GCSFuse to check: <prefix>/gcsfuse-nonexistent-object-check
	expectedObjectPath := prefix + "/" + testObject
	expectedRequestPath := "/b/" + TestBucketName + "/o/" + url.PathEscape(expectedObjectPath)
	// Set up a mock HTTP server to capture the outgoing GCS API request.
	requestPathChan := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPathChan <- r.URL.EscapedPath()
		// Mock a 404 response. For access verification, 404 (Not Found) means
		// we have permission to access the path (otherwise we would get 403 Forbidden).
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": 404, "message": "Not Found"}}`))
	}))
	defer server.Close()
	// Configure GCSFuse client with the mock server URL and only-dir prefix.
	scConfig := storageutil.StorageClientConfig{
		ClientProtocol:     cfg.HTTP1,
		EnableMountRetries: true, // Triggers verifyNonHNSBucketAccess during BucketHandle
		MaxRetrySleep:      time.Microsecond,
		RetryMultiplier:    1.0,
		MaxRetryAttempts:   1,
		OnlyDir:            prefix,
		CustomEndpoint:     server.URL,
		AnonymousAccess:    true, // Bypass real auth for mock server
	}
	sh, err := NewStorageHandle(testSuite.ctx, scConfig, "")
	require.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh)

	// Trigger bucket setup which runs the verification checks.
	bh, err := sh.BucketHandle(testSuite.ctx, TestBucketName, "")

	require.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), bh)
	// Assert that the request path captured by the mock server matches our expected prefixed path.
	select {
	case reqPath := <-requestPathChan:
		assert.Equal(testSuite.T(), expectedRequestPath, reqPath)
	case <-time.After(time.Second):
		testSuite.T().Fatal("Timeout waiting for request")
	}
}

func (testSuite *StorageHandleTest) TestGetClient_ProtocolSelection() {
	var lastReqClose atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReqClose.Store(r.Close)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": 404, "message": "Not Found"}}`))
	}))
	defer server.Close()
	testSuite.T().Setenv("STORAGE_EMULATOR_HOST", server.Listener.Addr().String())
	testSuite.T().Setenv("STORAGE_EMULATOR_HOST_GRPC", server.Listener.Addr().String())

	testCases := []struct {
		name                string
		isBucketRapid       bool
		clientProtocol      cfg.Protocol
		enableGrpcByDefault bool
		grpcPathStrategy    cfg.DirectPathStrategy
		customEndpoint      string
		expectErr           bool
		expectHTTP1Client   bool
		expectHTTP2Client   bool
		expectGRPCClient    bool
		expectBidiClient    bool
	}{
		{
			name:                "Regional_DefaultUnsetProtocol_UsesHTTP1",
			isBucketRapid:       false,
			clientProtocol:      "",
			enableGrpcByDefault: false,
			expectHTTP1Client:   true,
		},
		{
			name:                "Regional_ExplicitHTTP1_OverridesEnableGrpcByDefault",
			isBucketRapid:       false,
			clientProtocol:      cfg.HTTP1,
			enableGrpcByDefault: true,
			expectHTTP1Client:   true,
		},
		{
			name:                "Regional_ExplicitHTTP2_OverridesEnableGrpcByDefault",
			isBucketRapid:       false,
			clientProtocol:      cfg.HTTP2,
			enableGrpcByDefault: true,
			expectHTTP2Client:   true,
		},
		{
			name:                "Regional_ExplicitGRPC_SkipsDirectPathEnforcement",
			isBucketRapid:       false,
			clientProtocol:      cfg.GRPC,
			enableGrpcByDefault: false,
			expectGRPCClient:    true,
		},
		{
			name:                "Regional_ExplicitGRPC_WithEnableGrpcByDefault_SkipsDirectPathEnforcement",
			isBucketRapid:       false,
			clientProtocol:      cfg.GRPC,
			enableGrpcByDefault: true,
			expectGRPCClient:    true,
		},
		{
			name:                "Rapid_AlwaysUsesBidiGRPC",
			isBucketRapid:       true,
			clientProtocol:      cfg.HTTP1,
			enableGrpcByDefault: false,
			expectBidiClient:    true,
		},
		{
			name:                "Regional_EnableGrpcByDefault_DirectPathFails_FallsBackToHTTP1",
			isBucketRapid:       false,
			clientProtocol:      "",
			enableGrpcByDefault: true,
			grpcPathStrategy:    cfg.DirectPathWithFallback,
			expectHTTP1Client:   true,
		},
		{
			name:                "Regional_EnableGrpcByDefault_DirectPathOnly_ReturnsError",
			isBucketRapid:       false,
			clientProtocol:      "",
			enableGrpcByDefault: true,
			grpcPathStrategy:    cfg.DirectPathOnly,
			expectErr:           true,
		},
		{
			name:                "Regional_EnableGrpcByDefault_CustomEndpointBypassesGRPC_UsesHTTP1",
			isBucketRapid:       false,
			clientProtocol:      "",
			enableGrpcByDefault: true,
			grpcPathStrategy:    cfg.DirectPathOnly,
			customEndpoint:      server.URL,
			expectHTTP1Client:   true,
		},
	}

	for _, tc := range testCases {
		testSuite.Run(tc.name, func() {
			sh := &storageClient{
				clientConfig: storageutil.StorageClientConfig{
					ClientProtocol:      tc.clientProtocol,
					EnableGrpcByDefault: tc.enableGrpcByDefault,
					GrpcPathStrategy:    tc.grpcPathStrategy,
					AnonymousAccess:     true,
					CustomEndpoint:      tc.customEndpoint,
				},
			}
			defer func() {
				if sh.httpClient != nil {
					_ = sh.httpClient.Close()
				}
				if sh.grpcClient != nil {
					_ = sh.grpcClient.Close()
				}
				if sh.grpcClientWithBidiConfig != nil {
					_ = sh.grpcClientWithBidiConfig.Close()
				}
			}()

			// Empty bucket name makes verifyDirectPathConnectivity fail immediately without retry backoff.
			client, err := sh.getClient(testSuite.ctx, tc.isBucketRapid, "", "")

			assert.Equal(testSuite.T(), tc.clientProtocol, sh.clientConfig.ClientProtocol)
			if tc.expectErr {
				require.Error(testSuite.T(), err)
				assert.Nil(testSuite.T(), client)
				assert.Nil(testSuite.T(), sh.httpClient)
				return
			}
			require.NoError(testSuite.T(), err)
			require.NotNil(testSuite.T(), client)
			if tc.expectHTTP1Client {
				assert.Equal(testSuite.T(), sh.httpClient, client)
				assert.Nil(testSuite.T(), sh.grpcClient)
				_, _ = client.Bucket(TestBucketName).Object("test-obj").Attrs(testSuite.ctx)
				assert.False(testSuite.T(), lastReqClose.Load(), "HTTP/1.1 transport should keep connections alive (DisableKeepAlives=false)")
			}
			if tc.expectHTTP2Client {
				assert.Equal(testSuite.T(), sh.httpClient, client)
				assert.Nil(testSuite.T(), sh.grpcClient)
				_, _ = client.Bucket(TestBucketName).Object("test-obj").Attrs(testSuite.ctx)
				assert.True(testSuite.T(), lastReqClose.Load(), "HTTP/2 transport should set DisableKeepAlives=true")
			}
			if tc.expectGRPCClient {
				assert.Equal(testSuite.T(), sh.grpcClient, client)
				assert.Nil(testSuite.T(), sh.httpClient)
			}
			if tc.expectBidiClient {
				assert.Equal(testSuite.T(), sh.grpcClientWithBidiConfig, client)
				assert.Nil(testSuite.T(), sh.httpClient)
			}
		})
	}
}

func (testSuite *StorageHandleTest) TestGetClient_Regional_EnableGrpcByDefault_DirectPathSucceeds_UsesGRPC() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(testSuite.T(), err)
	grpcServer := grpc.NewServer(grpc.UnknownServiceHandler(func(srv any, stream grpc.ServerStream) error {
		return status.Error(codes.NotFound, "object not found")
	}))
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()
	testSuite.T().Setenv("STORAGE_EMULATOR_HOST_GRPC", listener.Addr().String())
	sh := &storageClient{
		clientConfig: storageutil.StorageClientConfig{
			ClientProtocol:      "",
			EnableGrpcByDefault: true,
			AnonymousAccess:     true,
		},
	}
	defer func() {
		if sh.grpcClient != nil {
			_ = sh.grpcClient.Close()
		}
	}()

	client, err := sh.getClient(testSuite.ctx, false, TestBucketName, "")

	require.NoError(testSuite.T(), err)
	assert.NotNil(testSuite.T(), sh.grpcClient)
	assert.Equal(testSuite.T(), sh.grpcClient, client)
	assert.Equal(testSuite.T(), cfg.Protocol(""), sh.clientConfig.ClientProtocol)
	assert.Nil(testSuite.T(), sh.httpClient)
}

func (testSuite *StorageHandleTest) TestGetClient_Regional_EnableGrpcByDefault_ReusesCachedGRPCClient() {
	cachedClient := &storage.Client{}
	sh := &storageClient{
		grpcClient: cachedClient,
		clientConfig: storageutil.StorageClientConfig{
			ClientProtocol:      "",
			EnableGrpcByDefault: true,
			AnonymousAccess:     true,
		},
	}

	client, err := sh.getClient(testSuite.ctx, false, "another-bucket", "")

	require.NoError(testSuite.T(), err)
	assert.Same(testSuite.T(), cachedClient, client)
	assert.Nil(testSuite.T(), sh.httpClient)
}

// Test_GrpcKeepaliveDefaultParams verifies that the production gRPC client
// keepalive parameters match the intended 30s ping interval and 20s ping ACK
// timeout (~50s total stall detection budget).
func (testSuite *StorageHandleTest) Test_GrpcKeepaliveDefaultParams() {
	assert.Equal(testSuite.T(), 30*time.Second, grpcKeepaliveTime)
	assert.Equal(testSuite.T(), 20*time.Second, grpcKeepaliveTimeout)
}

// stallBlackholeConn wraps the first client-side net.Conn and, once blackhole()
// is called after the initial object chunk is read, blocks all further Reads on
// that connection until Close() is called.
//
// Why this wrapper is necessary:
// If an in-process grpc.Server handler simply sleeps mid-stream, grpc-go's
// background HTTP/2 server transport goroutine (http2Server) remains active on
// the socket and automatically replies with HTTP/2 PING ACK frames whenever the
// client sends a keepalive PING. By blocking Read on the client's net.Conn once
// blackhole() is invoked, we prevent the client's HTTP/2 reader goroutine from
// receiving any frames (including PING ACKs), faithfully simulating a wedged L7
// proxy or blackholed network path where TCP stays open (no FIN/RST) while HTTP/2
// traffic stops.
type stallBlackholeConn struct {
	net.Conn
	blackholed  atomic.Bool
	intercepted atomic.Bool
	closed      chan struct{}
	closeOnce   sync.Once
}

func (c *stallBlackholeConn) blackhole() {
	c.blackholed.Store(true)
	// Interrupt any goroutine already blocked in c.Conn.Read(b) so it transitions
	// immediately into waiting on <-c.closed instead of reading a future PING ACK.
	_ = c.SetReadDeadline(time.Now())
}

func (c *stallBlackholeConn) Read(b []byte) (int, error) {
	if c.blackholed.Load() {
		c.intercepted.Store(true)
		<-c.closed
		return 0, net.ErrClosed
	}
	n, err := c.Conn.Read(b)
	if c.blackholed.Load() {
		c.intercepted.Store(true)
		<-c.closed
		return 0, net.ErrClosed
	}
	return n, err
}

func (c *stallBlackholeConn) Close() error {
	// Unblock any goroutine waiting in Read() when grpc-go closes the wedged transport.
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type fakeReadStallStorageServer interface{}

// registerReadStallStorageServer registers a minimal google.storage.v2.Storage
// gRPC service implementing the server-streaming ReadObject RPC:
//   - On the very first ReadObject attempt (gated by stalledOnce), it sends the
//     first stallAfterBytes (4 KiB) and then stalls for 25s. Sending the initial
//     chunk moves the RPC past response headers, where experimental.WithReadStallTimeout
//     no longer applies.
//   - On any subsequent ReadObject attempt after gRPC keepalive aborts the wedged
//     connection and the Go Storage SDK reopens the stream at the last received
//     byte offset, it immediately sends the remaining payload[offset:].
//
// We look up ReadObjectRequest and ReadObjectResponse via protoregistry.GlobalTypes
// (already linked into the binary by cloud.google.com/go/storage) so we do not need
// to import internal/generated storagepb packages directly, while still using the
// generated message structs for deterministic protobuf field serialization.
func registerReadStallStorageServer(t *testing.T, s *grpc.Server, payload []byte, stallAfterBytes int) {
	t.Helper()
	var srvPtr *fakeReadStallStorageServer
	reqType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectRequest")
	require.NoError(t, err)
	respType, err := protoregistry.GlobalTypes.FindMessageByName("google.storage.v2.ReadObjectResponse")
	require.NoError(t, err)

	sendChunk := func(stream grpc.ServerStream, chunk []byte) error {
		resp := respType.New()
		dataField := resp.Descriptor().Fields().ByName("checksummed_data")
		dataMsg := resp.NewField(dataField).Message()
		dataMsg.Set(dataMsg.Descriptor().Fields().ByName("content"), protoreflect.ValueOf(chunk))
		resp.Set(dataField, protoreflect.ValueOfMessage(dataMsg))

		metaField := resp.Descriptor().Fields().ByName("metadata")
		metaMsg := resp.NewField(metaField).Message()
		metaMsg.Set(metaMsg.Descriptor().Fields().ByName("name"), protoreflect.ValueOf("test-obj"))
		metaMsg.Set(metaMsg.Descriptor().Fields().ByName("size"), protoreflect.ValueOf(int64(len(payload))))
		resp.Set(metaField, protoreflect.ValueOfMessage(metaMsg))
		return stream.SendMsg(resp.Interface())
	}

	var stalledOnce atomic.Bool
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "google.storage.v2.Storage",
		HandlerType: srvPtr,
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "ReadObject",
				ServerStreams: true,
				Handler: func(srv any, stream grpc.ServerStream) error {
					req := reqType.New()
					if err := stream.RecvMsg(req.Interface()); err != nil {
						return err
					}
					offset := int(req.Get(req.Descriptor().Fields().ByName("read_offset")).Int())
					if stalledOnce.CompareAndSwap(false, true) {
						// Send initial chunk (past response headers, where WithReadStallTimeout
						// does not apply), then stall mid-stream for 25s (longer than the 16s
						// test context timeout, so the test would fail without keepalive).
						if err := sendChunk(stream, payload[:stallAfterBytes]); err != nil {
							return err
						}
						timer := time.NewTimer(25 * time.Second)
						defer timer.Stop()
						select {
						case <-timer.C:
						case <-stream.Context().Done():
							return stream.Context().Err()
						}
						return sendChunk(stream, payload[stallAfterBytes:])
					}
					// Resumed stream (offset == stallAfterBytes): send the remaining bytes immediately.
					return sendChunk(stream, payload[offset:])
				},
			},
		},
		Metadata: "google/storage/v2/storage.proto",
	}, struct{}{})
}

// Test_GrpcKeepaliveMidStreamStall verifies end-to-end recovery from a mid-stream
// gRPC read stall using the keepalive option wired in createClientOptionForGRPCClient:
//  1. Client opens a ReadObject stream via NewReader(ctx) for an 8 KiB object
//     (`read_offset = 0`), receiving the first 4 KiB chunk before NewReader returns.
//  2. Server pauses for 25s before sending the second 4 KiB chunk.
//  3. Right after NewReader(ctx) returns, the test calls blackhole() on the first
//     TCP connection so no further HTTP/2 frames (including PING ACKs) can reach
//     the client—without imposing any timing-sensitive read deadline during the
//     initial HTTP/2 handshake or first chunk read.
//  4. After grpcKeepaliveTime (10s), grpc-go sends an HTTP/2 PING frame; after
//     grpcKeepaliveTimeout (500ms) with no PING ACK, grpc-go closes the wedged
//     connection and fails the stream with codes.Unavailable.
//  5. The Go Storage SDK / GCSFuse retry loop dials a fresh connection and reopens
//     the stream at `read_offset = 4096`, reading the remaining 4 KiB to completion.
func (testSuite *StorageHandleTest) Test_GrpcKeepaliveMidStreamStall() {
	t := testSuite.T()
	// Override keepalive Time to 10s (the minimum enforced by grpc-go; values <10s
	// are clamped to 10s by grpc-go) and Timeout to 500ms so an unacknowledged
	// keepalive ping aborts the stalled transport in ~10.5s instead of ~50s.
	origTime, origTimeout := grpcKeepaliveTime, grpcKeepaliveTimeout
	grpcKeepaliveTime = 10 * time.Second
	grpcKeepaliveTimeout = 500 * time.Millisecond
	defer func() {
		grpcKeepaliveTime = origTime
		grpcKeepaliveTimeout = origTimeout
	}()

	const (
		totalSize       = 8 * 1024 // 8 KiB total object size
		stallAfterBytes = 4 * 1024 // Stall after the first 4 KiB chunk
	)
	payload := bytes.Repeat([]byte("A"), totalSize)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer()
	registerReadStallStorageServer(t, grpcServer, payload, stallAfterBytes)
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()

	// Bound the overall test to 16s: long enough for the ~10.5s keepalive timeout
	// and retry, but shorter than the server's 25s stall so the test fails if
	// keepalive does not abort the wedged connection.
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer cancel()

	var firstConn atomic.Pointer[stallBlackholeConn]
	dialer := grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		wrapped := &stallBlackholeConn{
			Conn:   conn,
			closed: make(chan struct{}),
		}
		if firstConn.CompareAndSwap(nil, wrapped) {
			return wrapped, nil
		}
		return conn, nil
	})

	sc := storageutil.GetDefaultStorageClientConfig(keyFile)
	sc.ClientProtocol = cfg.GRPC
	sc.CustomEndpoint = listener.Addr().String()
	sc.AnonymousAccess = true
	sc.GrpcConnPoolSize = 1
	sc.MaxRetrySleep = 10 * time.Millisecond
	sc.RetryMultiplier = 2.0
	sc.MetricHandle = metrics.NewNoopMetrics()

	clientOpts, err := createClientOptionForGRPCClient(ctx, &sc, false)
	require.NoError(t, err)
	clientOpts = append(clientOpts, option.WithGRPCDialOption(dialer))

	client, err := storage.NewGRPCClient(ctx, clientOpts...)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()
	setRetryConfig(ctx, client, &sc)

	r, err := client.Bucket(TestBucketName).Object("test-obj").NewReader(ctx)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	// NewReader(ctx) has already received the first 4 KiB ReadObjectResponse.
	// Blackhole the first connection now to simulate a mid-stream L7/network wedge.
	conn1 := firstConn.Load()
	require.NotNil(t, conn1)
	conn1.blackhole()

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, r)
	require.NoError(t, err)
	assert.True(t, conn1.intercepted.Load(), "expected blackholed first connection to intercept mid-stream reads")
	assert.Equal(t, payload, buf.Bytes())
}

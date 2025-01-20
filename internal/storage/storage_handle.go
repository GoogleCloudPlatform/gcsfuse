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
	"fmt"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"cloud.google.com/go/storage/experimental"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"golang.org/x/net/context"
	option "google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// Side effect to run grpc client with direct-path on gcp machine.
	_ "google.golang.org/grpc/balancer/rls"
	_ "google.golang.org/grpc/xds/googledirectpath"
)

const (
	// Used to modify the hidden options in go-sdk for read stall retry.
	// Ref: https://github.com/googleapis/google-cloud-go/blob/main/storage/option.go#L30
	dynamicReadReqIncreaseRateEnv   = "DYNAMIC_READ_REQ_INCREASE_RATE"
	dynamicReadReqInitialTimeoutEnv = "DYNAMIC_READ_REQ_INITIAL_TIMEOUT"

	zonalLocationType = "zone"
)

type StorageHandle interface {
	// In case of non-empty billingProject, this project is set as user-project for
	// all subsequent calls on the bucket. Calls with user-project will be billed
	// to that project rather than to the bucket's owning project.
	//
	// A user-project is required for all operations on Requester Pays buckets.
	BucketHandle(ctx context.Context, bucketName string, billingProject string) (bh *bucketHandle, err error)
}

type storageClient struct {
	httpClient           *storage.Client
	grpcClient           *storage.Client
	clientProtocol       cfg.Protocol
	storageControlClient StorageControlClient
	directPathDetector   *gRPCDirectPathDetector
}

type gRPCDirectPathDetector struct {
	clientOptions []option.ClientOption
}

// isDirectPathPossible checks if gRPC direct connectivity is available for a specific bucket
// from the environment where the client is running. A `nil` error represents Direct Connectivity was
// detected.
func (pd *gRPCDirectPathDetector) isDirectPathPossible(ctx context.Context, bucketName string) error {
	return storage.CheckDirectConnectivitySupported(ctx, bucketName, pd.clientOptions...)
}

// Return clientOpts for both gRPC client and control client.
func createClientOptionForGRPCClient(clientConfig *storageutil.StorageClientConfig) (clientOpts []option.ClientOption, err error) {
	// Add Custom endpoint option.
	if clientConfig.CustomEndpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(storageutil.StripScheme(clientConfig.CustomEndpoint)))
		// TODO(b/390799251): Check if this line can be merged with below anonymousAccess check.
		if clientConfig.AnonymousAccess {
			clientOpts = append(clientOpts, option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		}
	}

	if clientConfig.AnonymousAccess {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	} else {
		tokenSrc, tokenCreationErr := storageutil.CreateTokenSource(clientConfig)
		if tokenCreationErr != nil {
			err = fmt.Errorf("while fetching tokenSource: %w", tokenCreationErr)
			return
		}
		clientOpts = append(clientOpts, option.WithTokenSource(tokenSrc))
	}

	clientOpts = append(clientOpts, option.WithGRPCConnectionPool(clientConfig.GrpcConnPoolSize))
	clientOpts = append(clientOpts, option.WithUserAgent(clientConfig.UserAgent))
	// Turning off the go-sdk metrics exporter to prevent any problems.
	// TODO (kislaykishore) - to revisit here for monitoring support.
	clientOpts = append(clientOpts, storage.WithDisabledClientMetrics())
	return
}

// Followed https://pkg.go.dev/cloud.google.com/go/storage#hdr-Experimental_gRPC_API to create the gRPC client.
func createGRPCClientHandle(ctx context.Context, clientConfig *storageutil.StorageClientConfig) (sc *storage.Client, err error) {

	if err := os.Setenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS", "true"); err != nil {
		logger.Fatal("error setting direct path env var: %v", err)
	}

	var clientOpts []option.ClientOption
	clientOpts, err = createClientOptionForGRPCClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("error in getting clientOpts for gRPC client: %w", err)
	}

	sc, err = storage.NewGRPCClient(ctx, clientOpts...)
	if err != nil {
		err = fmt.Errorf("NewGRPCClient: %w", err)
	}

	// Unset the environment variable, since it's used only while creation of grpc client.
	if err := os.Unsetenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS"); err != nil {
		logger.Fatal("error while unsetting direct path env var: %v", err)
	}

	return
}

func createHTTPClientHandle(ctx context.Context, clientConfig *storageutil.StorageClientConfig) (sc *storage.Client, err error) {
	var clientOpts []option.ClientOption

	// Add WithHttpClient option.
	var httpClient *http.Client
	httpClient, err = storageutil.CreateHttpClient(clientConfig)
	if err != nil {
		err = fmt.Errorf("while creating http endpoint: %w", err)
		return
	}

	clientOpts = append(clientOpts, option.WithHTTPClient(httpClient))

	if clientConfig.AnonymousAccess {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	}

	// Create client with JSON read flow, if EnableJasonRead flag is set.
	if clientConfig.ExperimentalEnableJsonRead {
		clientOpts = append(clientOpts, storage.WithJSONReads())
	}

	// Add Custom endpoint option.
	if clientConfig.CustomEndpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(clientConfig.CustomEndpoint))
	}

	if clientConfig.ReadStallRetryConfig.Enable {
		// Hidden way to modify the increase rate for dynamic delay algorithm in go-sdk.
		// Ref: https://github.com/googleapis/google-cloud-go/blob/main/storage/option.go#L47
		// Temporarily we kept an option to change the increase-rate, will be removed
		// once we get a good default.
		err = os.Setenv(dynamicReadReqIncreaseRateEnv, strconv.FormatFloat(clientConfig.ReadStallRetryConfig.ReqIncreaseRate, 'f', -1, 64))
		if err != nil {
			logger.Warnf("Error while setting the env %s: %v", dynamicReadReqIncreaseRateEnv, err)
		}

		// Hidden way to modify the initial-timeout of the dynamic delay algorithm in go-sdk.
		// Ref: https://github.com/googleapis/google-cloud-go/blob/main/storage/option.go#L62
		// Temporarily we kept an option to change the initial-timeout, will be removed
		// once we get a good default.
		err = os.Setenv(dynamicReadReqInitialTimeoutEnv, clientConfig.ReadStallRetryConfig.InitialReqTimeout.String())
		if err != nil {
			logger.Warnf("Error while setting the env %s: %v", dynamicReadReqInitialTimeoutEnv, err)
		}
		clientOpts = append(clientOpts, experimental.WithReadStallTimeout(&experimental.ReadStallTimeoutConfig{
			Min:              clientConfig.ReadStallRetryConfig.MinReqTimeout,
			TargetPercentile: clientConfig.ReadStallRetryConfig.ReqTargetPercentile,
		}))
	}
	return storage.NewClient(ctx, clientOpts...)
}

func (sh *storageClient) lookupBucketType(bucketName string) (*gcs.BucketType, error) {
	var nilControlClient *control.StorageControlClient = nil
	if sh.storageControlClient == nilControlClient {
		return &gcs.BucketType{}, nil // Assume defaults
	}

	storageLayout, err := sh.getStorageLayout(bucketName)

	if err != nil {
		return nil, err
	}

	return &gcs.BucketType{
		Hierarchical: storageLayout.GetHierarchicalNamespace().GetEnabled(),
		Zonal:        storageLayout.GetLocationType() == zonalLocationType,
	}, nil
}

func (sh *storageClient) getStorageLayout(bucketName string) (*controlpb.StorageLayout, error) {
	var callOptions []gax.CallOption
	stoargeLayout, err := sh.storageControlClient.GetStorageLayout(context.Background(), &controlpb.GetStorageLayoutRequest{
		Name:      fmt.Sprintf("projects/_/buckets/%s/storageLayout", bucketName),
		Prefix:    "",
		RequestId: "",
	}, callOptions...)

	return stoargeLayout, err
}

// NewStorageHandle returns the handle of http or grpc Go storage client based on the
// provided StorageClientConfig.ClientProtocol.
// Please check out the StorageClientConfig to know about the parameters used in
// http and gRPC client.
func NewStorageHandle(ctx context.Context, clientConfig storageutil.StorageClientConfig) (sh StorageHandle, err error) {
	var hc, gc *storage.Client
	// The default protocol for the Go Storage control client's folders API is gRPC.
	// gcsfuse will initially mirror this behavior due to the client's lack of HTTP support.
	var controlClient *control.StorageControlClient
	var clientOpts []option.ClientOption
	var directPathDetector *gRPCDirectPathDetector

	gc, err = createGRPCClientHandle(ctx, &clientConfig)
	if err == nil {
		clientOpts, err = createClientOptionForGRPCClient(&clientConfig)
		directPathDetector = &gRPCDirectPathDetector{clientOptions: clientOpts}
	}
	if err != nil {
		err = fmt.Errorf("go grpc storage client creation failed: %w", err)
		return
	}

	hc, err = createHTTPClientHandle(ctx, &clientConfig)
	if err != nil {
		err = fmt.Errorf("go http storage client creation failed: %w", err)
		return
	}

	// TODO: We will implement an additional check for the HTTP control client protocol once the Go SDK supports HTTP.
	// TODO: Custom endpoints do not currently support gRPC. Remove this additional check once TPC(custom-endpoint) supports gRPC.
	// Create storageControlClient irrespective of whether hns needs to be enabled or not.
	// Because we will use storageControlClient to check layout of given bucket.
	//  TODO: For now, disabling control-client for all cases where custom-endpoint is not-nil.
	//  These include TPC, storage-testbench and preprod environment. It should be
	//  re-enabled for preprod later by explicitly allowing the preprod endpoint.
	if clientConfig.EnableHNS && clientConfig.CustomEndpoint == "" {
		clientOpts, err = createClientOptionForGRPCClient(&clientConfig)
		if err != nil {
			return nil, fmt.Errorf("error in getting clientOpts for gRPC client: %w", err)
		}
		controlClient, err = storageutil.CreateGRPCControlClient(ctx, clientOpts, &clientConfig)
		if err != nil {
			return nil, fmt.Errorf("could not create StorageControl Client: %w", err)
		}
	}

	// ShouldRetry function checks if an operation should be retried based on the
	// response of operation (error.Code).
	// RetryAlways causes all operations to be checked for retries using
	// ShouldRetry function.
	// Without RetryAlways, only those operations are checked for retries which
	// are idempotent.
	// https://github.com/googleapis/google-cloud-go/blob/main/storage/storage.go#L1953
	retryOpts := []storage.RetryOption{storage.WithBackoff(gax.Backoff{
		Max:        clientConfig.MaxRetrySleep,
		Multiplier: clientConfig.RetryMultiplier,
	}),
		storage.WithPolicy(storage.RetryAlways),
		storage.WithErrorFunc(storageutil.ShouldRetry)}

	hc.SetRetry(retryOpts...)
	gc.SetRetry(retryOpts...)

	// The default MaxRetryAttempts value is 0 indicates no limit.
	if clientConfig.MaxRetryAttempts != 0 {
		hc.SetRetry(storage.WithMaxAttempts(clientConfig.MaxRetryAttempts))
		gc.SetRetry(storage.WithMaxAttempts(clientConfig.MaxRetryAttempts))
	}

	sh = &storageClient{
		httpClient:           hc,
		grpcClient:           gc,
		storageControlClient: controlClient,
		clientProtocol:       clientConfig.ClientProtocol,
		directPathDetector:   directPathDetector,
	}
	return
}

func (sh *storageClient) BucketHandle(ctx context.Context, bucketName string, billingProject string) (bh *bucketHandle, err error) {
	var client *storage.Client
	bucketType, err := sh.lookupBucketType(bucketName)
	if err != nil {
		return nil, fmt.Errorf("storageLayout call failed: %s", err)
	}

	if bucketType.Zonal || sh.clientProtocol == cfg.GRPC {
		client = sh.grpcClient
		if sh.directPathDetector != nil {
			if err := sh.directPathDetector.isDirectPathPossible(ctx, bucketName); err != nil {
				logger.Warnf("Direct path connectivity unavailable for %s, reason: %v", bucketName, err)
			}
		}
	} else if sh.clientProtocol == cfg.HTTP1 || sh.clientProtocol == cfg.HTTP2 {
		client = sh.httpClient
	} else {
		return nil, fmt.Errorf("invalid client-protocol requested: %s", sh.clientProtocol)
	}
	storageBucketHandle := client.Bucket(bucketName)

	if billingProject != "" {
		storageBucketHandle = storageBucketHandle.UserProject(billingProject)
	}

	bh = &bucketHandle{
		bucket:        storageBucketHandle,
		bucketName:    bucketName,
		controlClient: sh.storageControlClient,
		bucketType:    bucketType,
	}

	return
}

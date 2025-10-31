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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"cloud.google.com/go/storage/experimental"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/oauth2"
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
	BucketHandle(ctx context.Context, bucketName string, billingProject string, finalizeFileForRapid bool) (bh *bucketHandle, err error)
}

type storageClient struct {
	httpClient               *storage.Client
	grpcClient               *storage.Client
	grpcClientWithBidiConfig *storage.Client
	clientConfig             storageutil.StorageClientConfig
	// rawStorageControlClientWithoutGaxRetries is without any retries.
	rawStorageControlClientWithoutGaxRetries *control.StorageControlClient
	// rawStorageControlClientWithGaxRetries is with retry for Folder APIs.
	rawStorageControlClientWithGaxRetries *control.StorageControlClient
	// storageControlClient is with retry for GetStorageLayout and with handling for billing project.
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
func createClientOptionForGRPCClient(ctx context.Context, clientConfig *storageutil.StorageClientConfig, enableBidiConfig bool) (clientOpts []option.ClientOption, err error) {
	// Add custom endpoint if provided.
	if clientConfig.CustomEndpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(storageutil.StripScheme(clientConfig.CustomEndpoint)))

		// TODO(b/390799251): Check if this line can be merged with below anonymousAccess check.
		if clientConfig.AnonymousAccess {
			clientOpts = append(clientOpts, option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		}
	}

	// Configure authentication.
	if clientConfig.AnonymousAccess {
		clientOpts = append(clientOpts, option.WithoutAuthentication())
	} else if clientConfig.EnableGoogleLibAuth {
		var authOpts []option.ClientOption
		authOpts, _, err = storageutil.GetClientAuthOptionsAndToken(ctx, clientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get client auth options and token: %w", err)
		}
		clientOpts = append(clientOpts, authOpts...)
	} else {
		var tokenSrc oauth2.TokenSource
		tokenSrc, err = storageutil.CreateTokenSource(clientConfig)
		if err != nil {
			return nil, fmt.Errorf("while fetching token source: %w", err)
		}
		clientOpts = append(clientOpts, option.WithTokenSource(tokenSrc))
	}

	// Additional client options.
	if enableBidiConfig {
		clientOpts = append(clientOpts, experimental.WithGRPCBidiReads())
	}

	if clientConfig.SocketAddress != "" {
		dialer := &net.Dialer{}
		// The port can be 0, in which case the OS will choose a local port.
		// The format of SocketAddress is expected to be IP address.
		if err := storageutil.ConfigureDialerWithLocalAddr(dialer, clientConfig.SocketAddress); err != nil {
			return nil, fmt.Errorf("failed to configure dialer with socket address %q: %w", clientConfig.SocketAddress, err)
		}
		clientOpts = append(clientOpts, option.WithGRPCDialOption(grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", addr)
		})))
	}

	if clientConfig.TracingEnabled {
		clientOpts = append(clientOpts, option.WithGRPCDialOption(grpc.WithStatsHandler(otelgrpc.NewClientHandler())))
	}

	clientOpts = append(clientOpts, option.WithGRPCConnectionPool(clientConfig.GrpcConnPoolSize))
	clientOpts = append(clientOpts, option.WithUserAgent(clientConfig.UserAgent))
	// Turning off the go-sdk metrics exporter to prevent any problems.
	// TODO (kislaykishore) - to revisit here for monitoring support.
	clientOpts = append(clientOpts, storage.WithDisabledClientMetrics())

	return clientOpts, nil
}

func setRetryConfig(ctx context.Context, sc *storage.Client, clientConfig *storageutil.StorageClientConfig) {
	if sc == nil || clientConfig == nil {
		logger.Fatal("setRetryConfig: Empty storage client or clientConfig")
		return
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
		storage.WithErrorFunc(func(err error) bool {
			return storageutil.ShouldRetryWithMonitoring(ctx, err, clientConfig.MetricHandle)
		})}

	sc.SetRetry(retryOpts...)

	// The default MaxRetryAttempts value is 0 indicates no limit.
	if clientConfig.MaxRetryAttempts != 0 {
		sc.SetRetry(storage.WithMaxAttempts(clientConfig.MaxRetryAttempts))
	}
}

// Followed https://pkg.go.dev/cloud.google.com/go/storage#hdr-Experimental_gRPC_API to create the gRPC client.
func createGRPCClientHandle(ctx context.Context, clientConfig *storageutil.StorageClientConfig, enableBidiConfig bool) (sc *storage.Client, err error) {

	if err := os.Setenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS", "true"); err != nil {
		logger.Fatal("error setting direct path env var: %v", err)
	}

	var clientOpts []option.ClientOption
	clientOpts, err = createClientOptionForGRPCClient(ctx, clientConfig, enableBidiConfig)
	if err != nil {
		return nil, fmt.Errorf("error in getting clientOpts for gRPC client: %w", err)
	}

	sc, err = storage.NewGRPCClient(ctx, clientOpts...)
	if err != nil {
		err = fmt.Errorf("NewGRPCClient: %w", err)
	} else {
		setRetryConfig(ctx, sc, clientConfig)
	}

	// Unset the environment variable, since it's used only while creation of grpc client.
	if err := os.Unsetenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS"); err != nil {
		logger.Fatal("error while unsetting direct path env var: %v", err)
	}

	return
}

func createHTTPClientHandle(ctx context.Context, clientConfig *storageutil.StorageClientConfig) (sc *storage.Client, err error) {
	var clientOpts []option.ClientOption
	var tokenSrc oauth2.TokenSource = nil

	if clientConfig.EnableGoogleLibAuth {
		var authOpts []option.ClientOption
		authOpts, tokenSrc, err = storageutil.GetClientAuthOptionsAndToken(ctx, clientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get client auth options and token: %w", err)
		}
		clientOpts = append(clientOpts, authOpts...)
	}

	// Add WithHttpClient option.
	var httpClient *http.Client
	httpClient, err = storageutil.CreateHttpClient(clientConfig, tokenSrc)
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
	sc, err = storage.NewClient(ctx, clientOpts...)
	if err != nil {
		err = fmt.Errorf("go http storage client creation failed: %w", err)
		return
	}
	setRetryConfig(ctx, sc, clientConfig)
	return
}

func (sh *storageClient) lookupBucketType(bucketName string) (*gcs.BucketType, error) {
	if sh.storageControlClient == nil {
		return &gcs.BucketType{}, nil // Assume defaults
	}

	startTime := time.Now()
	logger.Infof("GetStorageLayout <- (%s)", bucketName)
	storageLayout, err := sh.getStorageLayout(bucketName)
	duration := time.Since(startTime)

	if err != nil {
		return nil, err
	}

	logger.Infof("GetStorageLayout -> (%s) %v msec", bucketName, duration.Milliseconds())

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

// NewStorageHandle creates control client and stores client config to allow dynamic
// creation of http or grpc client.
func NewStorageHandle(ctx context.Context, clientConfig storageutil.StorageClientConfig, billingProject string) (sh StorageHandle, err error) {
	// The default protocol for the Go Storage control client's folders API is gRPC.
	// gcsfuse will initially mirror this behavior due to the client's lack of HTTP support.
	var controlClient StorageControlClient
	var rawStorageControlClientWithoutGaxRetries *control.StorageControlClient
	var rawStorageControlClientWithGaxRetries *control.StorageControlClient
	var clientOpts []option.ClientOption

	// Control-client is needed for folder APIs and for getting storage-layout of the bucket.
	// GetStorageLayout API is not supported for storage-testbench, which are identified by custom-endpoint containing localhost.
	if clientConfig.EnableHNS && !strings.Contains(clientConfig.CustomEndpoint, "localhost") {
		clientOpts, err = createClientOptionForGRPCClient(ctx, &clientConfig, false)
		if err != nil {
			return nil, fmt.Errorf("error in getting clientOpts for gRPC client: %w", err)
		}
		rawStorageControlClientWithoutGaxRetries, err = storageutil.CreateGRPCControlClient(ctx, clientOpts, true)
		if err != nil {
			return nil, fmt.Errorf("could not create StorageControl Client without default gax retries: %w", err)
		}
		// rawStorageControlClientWithGaxRetries cannot be just a wrapper over rawStorageControlClientWithoutGaxRetries,
		// as it has its own dedicated array of CallOptions, and we need to keep those independent.
		rawStorageControlClientWithGaxRetries, err = storageutil.CreateGRPCControlClient(ctx, clientOpts, false)
		if err != nil {
			return nil, fmt.Errorf("could not create StorageControl Client with default gax retries: %w", err)
		}
		err = addGaxRetriesForFolderAPIs(rawStorageControlClientWithGaxRetries, &clientConfig)
		if err != nil {
			return nil, fmt.Errorf("could not add custom gax retries to StorageControl Client: %w", err)
		}
		// special handling for mounts created with custom billing projects.
		controlClientWithBillingProject := withBillingProject(rawStorageControlClientWithoutGaxRetries, billingProject)
		// Wrap the control client with retry-on-stall logic.
		// This will retry on only on GetStorageLayout call for all buckets.
		controlClient = withRetryOnStorageLayout(controlClientWithBillingProject, &clientConfig)
	} else {
		logger.Infof("Skipping storage control client creation because custom-endpoint %q was passed, which is assumed to be a storage testbench server because of 'localhost' in it.", clientConfig.CustomEndpoint)
	}

	sh = &storageClient{
		rawStorageControlClientWithoutGaxRetries: rawStorageControlClientWithoutGaxRetries,
		rawStorageControlClientWithGaxRetries:    rawStorageControlClientWithGaxRetries,
		storageControlClient:                     controlClient,
		clientConfig:                             clientConfig,
		directPathDetector:                       &gRPCDirectPathDetector{clientOptions: clientOpts},
	}
	return
}

func (sh *storageClient) getClient(ctx context.Context, isbucketZonal bool) (*storage.Client, error) {
	var err error
	if isbucketZonal {
		if sh.grpcClientWithBidiConfig == nil {
			sh.grpcClientWithBidiConfig, err = createGRPCClientHandle(ctx, &sh.clientConfig, true)
		}
		return sh.grpcClientWithBidiConfig, err
	}

	if sh.clientConfig.ClientProtocol == cfg.GRPC {
		if sh.grpcClient == nil {
			sh.grpcClient, err = createGRPCClientHandle(ctx, &sh.clientConfig, false)
		}
		return sh.grpcClient, err
	}

	if sh.clientConfig.ClientProtocol == cfg.HTTP1 || sh.clientConfig.ClientProtocol == cfg.HTTP2 {
		if sh.httpClient == nil {
			sh.httpClient, err = createHTTPClientHandle(ctx, &sh.clientConfig)
		}
		return sh.httpClient, err
	}

	return nil, fmt.Errorf("invalid client-protocol requested: %s", sh.clientConfig.ClientProtocol)
}

// controlClientForBucketHandle returns a storage control client for the given bucket handle,
// which takes care of properly adding support for retries and for billing project.
func (sh *storageClient) controlClientForBucketHandle(bucketType *gcs.BucketType, billingProject string) StorageControlClient {
	if sh.rawStorageControlClientWithGaxRetries == nil || sh.rawStorageControlClientWithoutGaxRetries == nil {
		return nil
	}

	var controlClientWithoutBillingProject StorageControlClient
	if bucketType.Zonal {
		// sh.storageControlClient already contains handling for billing project,
		// and enhanced retries for GetStorageLayout API call. Extending it here for
		// retries for folder APIs.
		// For zonal buckets, wrap the control client with retry-on-all-APIs.
		controlClientWithoutBillingProject = withRetryOnAllAPIs(sh.rawStorageControlClientWithoutGaxRetries, &sh.clientConfig)
	} else {
		// Apply GAX retries to the raw storage control client and returns a copy of it,
		// as it is important to avoid overwriting it,
		// as it is used with enhanced retries used by zonal buckets.
		controlClientWithoutBillingProject = withRetryOnStorageLayout(sh.rawStorageControlClientWithGaxRetries, &sh.clientConfig)
	}

	// Special handling for mounts created with custom billing projects.
	// Wrap it with billing-project, if there is any.
	return withBillingProject(controlClientWithoutBillingProject, billingProject)
}

func (sh *storageClient) BucketHandle(ctx context.Context, bucketName string, billingProject string, finalizeFileForRapid bool) (bh *bucketHandle, err error) {
	var client *storage.Client
	bucketType, err := sh.lookupBucketType(bucketName)
	if err != nil {
		return nil, fmt.Errorf("storageLayout call failed: %s", err)
	}

	client, err = sh.getClient(ctx, bucketType.Zonal)
	if err != nil {
		return nil, err
	}

	if bucketType.Zonal || sh.clientConfig.ClientProtocol == cfg.GRPC {
		if sh.directPathDetector != nil {
			if err := sh.directPathDetector.isDirectPathPossible(ctx, bucketName); err != nil {
				logger.Warnf("Direct path connectivity unavailable for %s, reason: %v", bucketName, err)
			} else {
				logger.Infof("Successfully connected over gRPC DirectPath for %s", bucketName)
			}
		}
	}

	storageBucketHandle := client.Bucket(bucketName)
	if billingProject != "" {
		storageBucketHandle = storageBucketHandle.UserProject(billingProject)
	}
	controlClient := sh.controlClientForBucketHandle(bucketType, billingProject)

	bh = &bucketHandle{
		bucket:               storageBucketHandle,
		bucketName:           bucketName,
		controlClient:        controlClient,
		bucketType:           bucketType,
		finalizeFileForRapid: finalizeFileForRapid,
	}

	return
}

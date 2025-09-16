// Copyright 2024 Google LLC
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
	"time"

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type StorageControlClient interface {
	GetStorageLayout(ctx context.Context,
		req *controlpb.GetStorageLayoutRequest,
		opts ...gax.CallOption) (*controlpb.StorageLayout, error)

	DeleteFolder(ctx context.Context,
		req *controlpb.DeleteFolderRequest,
		opts ...gax.CallOption) error

	GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error)

	RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error)

	CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error)
}

// storageControlClientWithBillingProject is a wrapper for an existing
// StorageControlClient object in that it passes
// the billing project in every call made through the base StorageControlClient.
type storageControlClientWithBillingProject struct {
	raw            StorageControlClient
	billingProject string
}

func (sccwbp *storageControlClientWithBillingProject) contextWithBillingProject(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-goog-user-project", sccwbp.billingProject)
}

func (sccwbp *storageControlClientWithBillingProject) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	return sccwbp.raw.GetStorageLayout(sccwbp.contextWithBillingProject(ctx), req, opts...)
}

func (sccwbp *storageControlClientWithBillingProject) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) error {
	return sccwbp.raw.DeleteFolder(sccwbp.contextWithBillingProject(ctx), req, opts...)
}

func (sccwbp *storageControlClientWithBillingProject) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	return sccwbp.raw.GetFolder(sccwbp.contextWithBillingProject(ctx), req, opts...)
}

func (sccwbp *storageControlClientWithBillingProject) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	// Don't pass billing-project for LROs as it's not supported.
	return sccwbp.raw.RenameFolder(ctx, req, opts...)
}

func (sccwbp *storageControlClientWithBillingProject) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	return sccwbp.raw.CreateFolder(sccwbp.contextWithBillingProject(ctx), req, opts...)
}

func withBillingProject(controlClient StorageControlClient, billingProject string) StorageControlClient {
	if billingProject != "" {
		controlClient = &storageControlClientWithBillingProject{raw: controlClient, billingProject: billingProject}
	}
	return controlClient
}

// storageControlClientWithRetry is a wrapper for an existing StorageControlClient object
// which implements gcsfuse-level retry logic if any of the control-client call gets stalled or returns a retryable error.
// It makes time-bound attempts to call the underlying StorageControlClient methods,
// retrying on errors that should be retried according to gcsfuse's retry logic.
type storageControlClientWithRetry struct {
	raw         StorageControlClient
	retryConfig *storageutil.RetryConfig

	// Whether or not to enable retries for GetStorageLayout call.
	enableRetriesOnStorageLayoutAPI bool
	// Whether or not to enable retries for folder APIs.
	enableRetriesOnFolderAPIs bool
}

func (sccwros *storageControlClientWithRetry) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	if !sccwros.enableRetriesOnStorageLayoutAPI {
		return sccwros.raw.GetStorageLayout(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.StorageLayout, error) {
		return sccwros.raw.GetStorageLayout(attemptCtx, req, opts...)
	}

	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "GetStorageLayout", req.Name, apiCall)
}

func (sccwros *storageControlClientWithRetry) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) error {
	if !sccwros.enableRetriesOnFolderAPIs {
		return sccwros.raw.DeleteFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (any, error) {
		err := sccwros.raw.DeleteFolder(attemptCtx, req, opts...)
		return struct{}{}, err
	}

	_, err := storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "DeleteFolder", req.Name, apiCall)
	return err
}

func (sccwros *storageControlClientWithRetry) GetFolder(ctx context.Context,
	req *controlpb.GetFolderRequest,
	opts ...gax.CallOption) (*controlpb.Folder, error) {
	if !sccwros.enableRetriesOnFolderAPIs {
		return sccwros.raw.GetFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.Folder, error) {
		return sccwros.raw.GetFolder(attemptCtx, req, opts...)
	}

	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "GetFolder", req.Name, apiCall)
}

func (sccwros *storageControlClientWithRetry) RenameFolder(ctx context.Context,
	req *controlpb.RenameFolderRequest,
	opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	if !sccwros.enableRetriesOnFolderAPIs {
		return sccwros.raw.RenameFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*control.RenameFolderOperation, error) {
		return sccwros.raw.RenameFolder(attemptCtx, req, opts...)
	}

	reqDescription := fmt.Sprintf("%q to %q", req.Name, req.DestinationFolderId)
	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "RenameFolder", reqDescription, apiCall)
}

func (sccwros *storageControlClientWithRetry) CreateFolder(ctx context.Context,
	req *controlpb.CreateFolderRequest,
	opts ...gax.CallOption) (*controlpb.Folder, error) {
	if !sccwros.enableRetriesOnFolderAPIs {
		return sccwros.raw.CreateFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.Folder, error) {
		return sccwros.raw.CreateFolder(attemptCtx, req, opts...)
	}

	reqDescription := fmt.Sprintf("%q in %q", req.FolderId, req.Parent)
	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "CreateFolder", reqDescription, apiCall)
}

// newRetryWrapper creates a new StorageControlClient with retry capabilities.
// It accepts various parameters to configure the retry behavior.
// The returned control client retries storage-layout.
// It also retries folder-related APIs if `retryFolderAPIs` is true.
func newRetryWrapper(controlClient StorageControlClient, clientConfig *storageutil.StorageClientConfig, retryDeadline, totalRetryBudget, initialBackoff time.Duration, retryFolderAPIs bool) StorageControlClient {
	// Avoid creating a nested wrapper.
	raw := controlClient
	if sccwros, ok := controlClient.(*storageControlClientWithRetry); ok {
		raw = sccwros.raw
	}

	retryConfig := storageutil.NewRetryConfig(clientConfig, retryDeadline, totalRetryBudget, initialBackoff)
	return &storageControlClientWithRetry{
		raw:                             raw,
		retryConfig:                     retryConfig,
		enableRetriesOnStorageLayoutAPI: true,
		enableRetriesOnFolderAPIs:       retryFolderAPIs,
	}
}

// withRetryOnAllAPIs wraps a StorageControlClient to do a time-bound retry approach for retryable errors for all API calls through it.
func withRetryOnAllAPIs(controlClient StorageControlClient,
	clientConfig *storageutil.StorageClientConfig) StorageControlClient {
	return newRetryWrapper(controlClient, clientConfig, storageutil.DefaultRetryDeadline, storageutil.DefaultTotalRetryBudget, storageutil.DefaultInitialBackoff, true)
}

// withRetryOnStorageLayout wraps a StorageControlClient to do a time-bound retry approach for retryable errors for the GetStorageLayout call through it.
func withRetryOnStorageLayout(controlClient StorageControlClient,
	clientConfig *storageutil.StorageClientConfig) StorageControlClient {
	return newRetryWrapper(controlClient, clientConfig, storageutil.DefaultRetryDeadline, storageutil.DefaultTotalRetryBudget, storageutil.DefaultInitialBackoff, false)

}

func storageControlClientGaxRetryOptions(clientConfig *storageutil.StorageClientConfig) []gax.CallOption {
	return []gax.CallOption{
		gax.WithTimeout(storageutil.DefaultTotalRetryBudget),
		gax.WithRetry(func() gax.Retryer {
			return gax.OnCodes([]codes.Code{
				codes.ResourceExhausted,
				codes.Unavailable,
				codes.DeadlineExceeded,
				codes.Internal,
				codes.Unknown,
			}, gax.Backoff{
				Max:        clientConfig.MaxRetrySleep,
				Multiplier: clientConfig.RetryMultiplier,
			})
		}),
	}
}

// addGaxRetriesForFolderAPIs updates the passed raw control client
// to add gax retries according to the given config in-place.
func addGaxRetriesForFolderAPIs(rawControlClient *control.StorageControlClient,
	clientConfig *storageutil.StorageClientConfig) error {
	if rawControlClient == nil || clientConfig == nil {
		return fmt.Errorf("invalid input: %v, %v", rawControlClient, clientConfig)
	}
	if rawControlClient.CallOptions == nil {
		return fmt.Errorf("cannot apply gax retries for folder APIs to raw control client: CallOptions is nil")
	}

	*rawControlClient.CallOptions = control.StorageControlCallOptions{}
	gaxRetryOptions := storageControlClientGaxRetryOptions(clientConfig)
	rawControlClient.CallOptions.RenameFolder = gaxRetryOptions
	rawControlClient.CallOptions.GetFolder = gaxRetryOptions
	rawControlClient.CallOptions.CreateFolder = gaxRetryOptions
	rawControlClient.CallOptions.DeleteFolder = gaxRetryOptions
	return nil
}

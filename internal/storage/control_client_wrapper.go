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

	control "cloud.google.com/go/storage/control/apiv2"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
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

// storageControlClientWithRetry is a wrapper for an existing StorageControlClient object
// that implements time-bound, exponential backoff retries for Storage Control API calls.
// By default, only GetStorageLayout requests are retried. Retry logic for folder-related
// API calls can be selectively enabled using the WithFolderAPIRetries builder.
// It also provides a WithBillingProject builder to optionally inject billing project headers.
type storageControlClientWithRetry struct {
	raw         StorageControlClient
	retryConfig *storageutil.RetryConfig

	// Whether or not to enable retries for folder APIs.
	enableRetriesOnFolderAPIs bool
}

func (sccwros *storageControlClientWithRetry) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	apiCall := func(attemptCtx context.Context) (*controlpb.StorageLayout, error) {
		return sccwros.raw.GetStorageLayout(attemptCtx, req, opts...)
	}

	return storageutil.ExecuteWithRetryAtLogLevel(ctx, sccwros.retryConfig, "GetStorageLayout", req.Name, req.RequestId, apiCall, logger.LevelInfo)
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

	_, err := storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "DeleteFolder", req.Name, req.RequestId, apiCall)
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

	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "GetFolder", req.Name, req.RequestId, apiCall)
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
	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "RenameFolder", reqDescription, req.RequestId, apiCall)
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
	return storageutil.ExecuteWithRetry(ctx, sccwros.retryConfig, "CreateFolder", reqDescription, req.RequestId, apiCall)
}

// ControlClientOption defines a functional option configuration.
type ControlClientOption func(*clientConfigState)

type clientConfigState struct {
	folderRetries  bool
	billingProject string
}

// WithRetriesOnFolderAPI returns an option that enables folder-level API stall retries.
func WithRetriesOnFolderAPI() ControlClientOption {
	return func(s *clientConfigState) {
		s.folderRetries = true
	}
}

// WithBillingProject returns an option that configures the billing project ID header to be injected into outgoing requests.
func WithBillingProject(project string) ControlClientOption {
	return func(s *clientConfigState) {
		s.billingProject = project
	}
}

// NewStorageControlClient creates a new StorageControlClient wrapping the provided raw client
// with stall retry capabilities and optional billing project header injection.
// It returns an error if clientConfig is nil, or if the raw client is already wrapped.
// This design keeps the call sites hygienic by failing early on invalid configurations
// or nested wrapping instead of silently handling them.
func NewStorageControlClient(raw StorageControlClient, clientConfig *storageutil.StorageClientConfig, opts ...ControlClientOption) (StorageControlClient, error) {
	if clientConfig == nil {
		return nil, fmt.Errorf("clientConfig cannot be nil")
	}
	if raw == nil {
		return nil, fmt.Errorf("raw client cannot be nil")
	}
	if _, ok := raw.(*storageControlClientWithBillingProject); ok {
		return nil, fmt.Errorf("raw client is already wrapped with billing project")
	}
	if _, ok := raw.(*storageControlClientWithRetry); ok {
		return nil, fmt.Errorf("raw client is already wrapped with retry capability")
	}

	var state clientConfigState
	for _, opt := range opts {
		opt(&state)
	}

	// 1. Wrap with retry capability
	retryConfig := storageutil.NewRetryConfig(clientConfig)
	wrapped := &storageControlClientWithRetry{
		raw:         raw,
		retryConfig: retryConfig,
	}
	if state.folderRetries {
		wrapped.enableRetriesOnFolderAPIs = true
	}

	// 2. Wrap with billing project if configured
	if state.billingProject != "" {
		return &storageControlClientWithBillingProject{raw: wrapped, billingProject: state.billingProject}, nil
	}
	return wrapped, nil
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
				// TODO(b/518674297): Please incorporate the correct fix post resolution of oauth2 issue.
				codes.Unauthenticated,
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

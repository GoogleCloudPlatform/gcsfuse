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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
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

// storageControlClientWithRetryOnStall is a wrapper for an existing StorageControlClient object
// which implements gcsfuse-level retry logic if any of the control-client call gets stalled.
// It makes time-bound attempts to call the underlying StorageControlClient methods,
// retrying on errors that should be retried according to gcsfuse's retry logic
type storageControlClientWithRetryOnStall struct {
	raw StorageControlClient

	minRetryDeadline time.Duration
	maxRetryDeadline time.Duration
	retryMultiplier  float64
	totalRetryBudget time.Duration

	// Whether or not to enable retries for folder APIs and other calls (except for the GetStorageLayout call).
	// If this object is used, then GetStorageLayout call is always retried.
	enableStallRetriesOnStorageLayoutCall bool
	enableStallRetriesOnAllCalls          bool
}

// executeWithStallRetry encapsulates the retry logic for control client operations.
// It performs time-bound, exponential backoff retries for a given API call.
func executeWithStallRetry[T any](
	sccwros *storageControlClientWithRetryOnStall,
	ctx context.Context,
	operationName string,
	reqDescription string,
	apiCall func(attemptCtx context.Context) (T, error),
) (T, error) {
	var zero T

	parentCtx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		attemptCtx, attemptCancel := context.WithTimeout(parentCtx, delay)

		logger.Tracef("Calling %s for %q with deadline=%v ...", operationName, reqDescription, delay)
		result, err := apiCall(attemptCtx)
		attemptCancel()

		if err == nil {
			return result, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if parentCtx.Err() != nil {
			return zero, fmt.Errorf("%s for %q timed out after multiple retries over %v: %w", operationName, reqDescription, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return zero, fmt.Errorf("%s for %q failed with a non-retryable error: %w", operationName, reqDescription, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying %s for %q with deadline=%v ...", operationName, reqDescription, delay)
	}
}

func (sccwros *storageControlClientWithRetryOnStall) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	if !sccwros.enableStallRetriesOnStorageLayoutCall {
		return sccwros.raw.GetStorageLayout(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.StorageLayout, error) {
		return sccwros.raw.GetStorageLayout(attemptCtx, req, opts...)
	}

	return executeWithStallRetry(sccwros, ctx, "GetStorageLayout", req.Name, apiCall)
}

func (sccwros *storageControlClientWithRetryOnStall) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) error {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.DeleteFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (any, error) {
		err := sccwros.raw.DeleteFolder(attemptCtx, req, opts...)
		return nil, err
	}

	_, err := executeWithStallRetry(sccwros, ctx, "DeleteFolder", req.Name, apiCall)
	return err
}

func (sccwros *storageControlClientWithRetryOnStall) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.GetFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.Folder, error) {
		return sccwros.raw.GetFolder(attemptCtx, req, opts...)
	}

	return executeWithStallRetry(sccwros, ctx, "GetFolder", req.Name, apiCall)
}

func (sccwros *storageControlClientWithRetryOnStall) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.RenameFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*control.RenameFolderOperation, error) {
		return sccwros.raw.RenameFolder(attemptCtx, req, opts...)
	}

	reqDescription := fmt.Sprintf("%q to %q", req.Name, req.DestinationFolderId)
	return executeWithStallRetry(sccwros, ctx, "RenameFolder", reqDescription, apiCall)
}

func (sccwros *storageControlClientWithRetryOnStall) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.CreateFolder(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.Folder, error) {
		return sccwros.raw.CreateFolder(attemptCtx, req, opts...)
	}

	reqDescription := fmt.Sprintf("%q in %q", req.FolderId, req.Parent)
	return executeWithStallRetry(sccwros, ctx, "CreateFolder", reqDescription, apiCall)
}

func newRetryWrapper(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration, retryAllCalls bool) StorageControlClient {
	raw := controlClient
	if sccwros, ok := controlClient.(*storageControlClientWithRetryOnStall); ok {
		raw = sccwros.raw
	}

	return &storageControlClientWithRetryOnStall{
		raw:                                   raw,
		minRetryDeadline:                      minRetryDeadline,
		maxRetryDeadline:                      maxRetryDeadline,
		retryMultiplier:                       retryMultiplier,
		totalRetryBudget:                      totalRetryBudget,
		enableStallRetriesOnStorageLayoutCall: true,
		enableStallRetriesOnAllCalls:          retryAllCalls,
	}
}

// withRetryOnStall wraps a StorageControlClient to implement gcsfuse-level retry logic.
// It retries operations with a time-bound approach, retrying on errors that should be retried
// according to gcsfuse's retry logic.
// The retry logic is based on a minimum and maximum delay, a retry multiplier, and a total attempts deadline.
func withRetryOnStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
	return newRetryWrapper(controlClient, minRetryDeadline, maxRetryDeadline, retryMultiplier, totalRetryBudget, true)
}

// withRetryOnStorageLayoutStall wraps a StorageControlClient which retries GetStorageLayout call with a time-bound approach, but the rest of the control-client calls pass through it as it is.
func withRetryOnStorageLayoutStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
	return newRetryWrapper(controlClient, minRetryDeadline, maxRetryDeadline, retryMultiplier, totalRetryBudget, false)
}

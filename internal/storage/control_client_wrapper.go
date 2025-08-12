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

const (
	// Default retry parameters for control client calls.
	defaultControlClientMinRetryDeadline = 10 * time.Second
	defaultControlClientMaxRetryDeadline = time.Minute
	defaultControlClientRetryMultiplier  = 2.0
	defaultControlClientTotalRetryBudget = 5 * time.Minute
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

	// Whether or not to enable retries for GetStorageLayout call.
	enableStallRetriesOnStorageLayoutCall bool
	// Whether or not to enable retries for folder APIs.
	enableStallRetriesOnAllCalls bool
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

		// If the attempt didn't time out but failed with another retryable error,
		// we must explicitly wait some time before the next attempt.
		// This is to avoid hammering the server with retries in a tight loop, and to let it recover.
		if attemptCtx.Err() == nil {
			select {
			case <-time.After(delay):
			case <-parentCtx.Done():
				return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtx.Err())
			}
		}

		// If the error is not retryable, return it immediately.
		if !storageutil.ShouldRetry(err) {
			return zero, fmt.Errorf("%s for %q failed with a non-retryable error: %w", operationName, reqDescription, err)
		}

		// If the parent context is cancelled, we should stop retrying.
		if parentCtx.Err() != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtx.Err())
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
		return struct{}{}, err
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
	// reasonable defaults for retry parameters
	if minRetryDeadline <= 0 {
		minRetryDeadline = defaultControlClientMinRetryDeadline
		logger.Warnf("minRetryDeadline was <= 0, defaulting to %v", defaultControlClientMinRetryDeadline)
	}
	if maxRetryDeadline < minRetryDeadline {
		maxRetryDeadline = max(defaultControlClientMaxRetryDeadline, minRetryDeadline)
		logger.Warnf("maxRetryDeadline was < minRetryDeadline, defaulting to %v", maxRetryDeadline)
	}
	if totalRetryBudget < maxRetryDeadline {
		totalRetryBudget = max(maxRetryDeadline, defaultControlClientTotalRetryBudget)
		logger.Warnf("totalRetryBudget was < maxRetryDeadline, defaulting to %v", totalRetryBudget)
	}
	if retryMultiplier <= 1.0 {
		retryMultiplier = defaultControlClientRetryMultiplier
		logger.Warnf("retryMultiplier was <= 1.0, defaulting to %v", retryMultiplier)
	}

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

// withRetryOnStall wraps a StorageControlClient to do a time-bound retry approach for retryable errors for all API calls through it.
func withRetryOnStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
	return newRetryWrapper(controlClient, minRetryDeadline, maxRetryDeadline, retryMultiplier, totalRetryBudget, true)
}

// withRetryOnStorageLayoutStall wraps a StorageControlClient to do a time-bound retry approach for retryable errors for the GetStorageLayout call through it.
func withRetryOnStorageLayoutStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
	return newRetryWrapper(controlClient, minRetryDeadline, maxRetryDeadline, retryMultiplier, totalRetryBudget, false)
}

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

	// Whether or not to enable reties for folder APIs and other calls (except for the GetStorageLayou call).
	// If this object is used, then GetStorageLayout call is always retried.
	enableStallRetriesOnStorageLayoutCall bool
	enableStallRetriesOnAllCalls          bool
}

func (sccwros *storageControlClientWithRetryOnStall) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (storageLayout *controlpb.StorageLayout, err error) {
	if !sccwros.enableStallRetriesOnStorageLayoutCall {
		return sccwros.raw.GetStorageLayout(ctx, req, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		var subCtx context.Context
		storageLayout, err = func() (*controlpb.StorageLayout, error) {
			subCtx, cancel = context.WithTimeout(ctx, delay)
			defer cancel()

			logger.Tracef("Calling GetStorageLayout for %q with deadline=%v ...", req.Name, delay)
			return sccwros.raw.GetStorageLayout(subCtx, req, opts...)
		}()

		if err == nil {
			return storageLayout, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("GetStorageLayout for %q timed out after multiple retries over %v: %w", req.Name, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("GetStorageLayout for %q failed with a non-retryable error: %w", req.Name, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying GetStorageLayout for %q with deadline=%v ...", req.Name, delay)
	}
}

func (sccwros *storageControlClientWithRetryOnStall) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) (err error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.DeleteFolder(ctx, req, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		var subCtx context.Context
		err = func() error {
			subCtx, cancel = context.WithTimeout(ctx, delay)
			defer cancel()

			logger.Tracef("Calling DeleteFolder for %q with deadline=%v ...", req.Name, delay)
			return sccwros.raw.DeleteFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if ctx.Err() != nil {
			return fmt.Errorf("DeleteFolder for %q timed out after multiple retries over %v: %w", req.Name, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return fmt.Errorf("DeleteFolder for %q failed with a non-retryable error: %w", req.Name, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying DeleteFolder for %q with deadline=%v ...", req.Name, delay)
	}
}

func (sccwros *storageControlClientWithRetryOnStall) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (folder *controlpb.Folder, err error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.GetFolder(ctx, req, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		var subCtx context.Context
		folder, err = func() (*controlpb.Folder, error) {
			subCtx, cancel = context.WithTimeout(ctx, delay)
			defer cancel()

			logger.Tracef("Calling GetFolder for %q with deadline=%v ...", req.Name, delay)
			return sccwros.raw.GetFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return folder, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("GetFolder for %q timed out after multiple retries over %v: %w", req.Name, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("GetFolder for %q failed with a non-retryable error: %w", req.Name, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying GetFolder for %q with deadline=%v ...", req.Name, delay)
	}
}

func (sccwros *storageControlClientWithRetryOnStall) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (op *control.RenameFolderOperation, err error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.RenameFolder(ctx, req, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		var subCtx context.Context
		op, err = func() (*control.RenameFolderOperation, error) {
			subCtx, cancel = context.WithTimeout(ctx, delay)
			defer cancel()

			logger.Tracef("Calling RenameFolder for %q -> %q with deadline=%v ...", req.Name, req.DestinationFolderId, delay)
			return sccwros.raw.RenameFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return op, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("RenameFolder from %q to %q timed out after multiple retries over %v: %w", req.Name, req.DestinationFolderId, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("RenameFolder from %q to %q failed with a non-retryable error: %w", req.Name, req.DestinationFolderId, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying RenameFolder from %q to %q with deadline=%v ...", req.Name, req.DestinationFolderId, delay)
	}
}

func (sccwros *storageControlClientWithRetryOnStall) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (folder *controlpb.Folder, err error) {
	if !sccwros.enableStallRetriesOnAllCalls {
		return sccwros.raw.CreateFolder(ctx, req, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	delay := sccwros.minRetryDeadline
	for {
		var subCtx context.Context
		folder, err = func() (*controlpb.Folder, error) {
			subCtx, cancel = context.WithTimeout(ctx, delay)
			defer cancel()

			logger.Tracef("Calling CreateFolder for %q in %q with deadline=%v ...", req.FolderId, req.Parent, delay)
			return sccwros.raw.CreateFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return folder, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("CreateFolder for %q in %q timed out after multiple retries over %v: %w", req.FolderId, req.Parent, sccwros.totalRetryBudget, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("CreateFolder for %q in %q failed with a non-retryable error: %w", req.FolderId, req.Parent, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Tracef("Retrying CreateFolder for %q in %q with deadline=%v ...", req.FolderId, req.Parent, delay)
	}
}

// withRetryOnStall wraps a StorageControlClient to implement gcsfuse-level retry logic.
// It retries operations with a time-bound approach, retrying on errors that should be retried
// according to gcsfuse's retry logic.
// The retry logic is based on a minimum and maximum delay, a retry multiplier, and a total attempts deadline.
func withRetryOnStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
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
		enableStallRetriesOnAllCalls:          true,
	}
}

// withRetryOnStorageLayoutStall wraps a StorageControlClient which retries GetStorageLayout call with a time-bound approach, but the rest of the control-client calls pass through it as it is.
func withRetryOnStorageLayoutStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
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
		enableStallRetriesOnAllCalls:          false,
	}
}

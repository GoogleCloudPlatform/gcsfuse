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

			return sccwros.raw.GetStorageLayout(subCtx, req, opts...)
		}()

		if err == nil {
			return storageLayout, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if subCtx.Err() != nil {
			return nil, fmt.Errorf("GetStorageLayout on %+v timed out after multiple retries: %w", req, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("GetStorageLayout on %+v failed with a non-retryable error: %w", req, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Warnf("Retrying GetStorageLayout on %+v with deadline %v ...", req, delay)
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

			return sccwros.raw.DeleteFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if subCtx.Err() != nil {
			return fmt.Errorf("DeleteFolder on %+v timed out after multiple retries: %w", req, err)
		}

		if !storageutil.ShouldRetry(err) {
			return fmt.Errorf("DeleteFolder on %+v failed with a non-retryable error: %w", req, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Warnf("Retrying DeleteFolder for %+v with deadline %v ...", req, delay)
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

			return sccwros.raw.GetFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return folder, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if subCtx.Err() != nil {
			return nil, fmt.Errorf("GetFolder on %+v timed out after multiple retries: %w", req, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("GetFolder on %+v failed with a non-retryable error: %w", req, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Warnf("Retrying GetFolder for %+v with deadline %v ...", req, delay)
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

			return sccwros.raw.RenameFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return op, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if subCtx.Err() != nil {
			return nil, fmt.Errorf("RenameFolder on %+v timed out after multiple retries: %w", req, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("RenameFolder on %+v failed with a non-retryable error: %w", req, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Warnf("Retrying RenameFolder for %+v with deadline %v ...", req, delay)
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

			return sccwros.raw.CreateFolder(subCtx, req, opts...)
		}()

		if err == nil {
			return folder, nil
		}

		// If the parent context is cancelled, we should stop retrying.
		if subCtx.Err() != nil {
			return nil, fmt.Errorf("CreateFolder on %+v timed out after multiple retries: %w", req, err)
		}

		if !storageutil.ShouldRetry(err) {
			return nil, fmt.Errorf("CreateFolder on %+v failed with a non-retryable error: %w", req, err)
		}

		// Increase delay for the next attempt.
		delay = min(sccwros.maxRetryDeadline, time.Duration(float64(delay)*sccwros.retryMultiplier))
		logger.Warnf("Retrying CreateFolder for %+v with deadline %v ...", req, delay)
	}
}

// withRetryOnStall wraps a StorageControlClient to implement gcsfuse-level retry logic.
// It retries operations with a time-bound approach, retrying on errors that should be retried
// according to gcsfuse's retry logic.
// The retry logic is based on a minimum and maximum delay, a retry multiplier, and a total attempts deadline.
func withRetryOnStall(controlClient StorageControlClient, minRetryDeadline time.Duration, maxRetryDeadline time.Duration, retryMultiplier float64, totalRetryBudget time.Duration) StorageControlClient {
	return &storageControlClientWithRetryOnStall{
		raw:                                   controlClient,
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
	return &storageControlClientWithRetryOnStall{
		raw: controlClient,
		// TODO: declare constants for these default values.
		minRetryDeadline:                      minRetryDeadline,
		maxRetryDeadline:                      maxRetryDeadline,
		retryMultiplier:                       retryMultiplier,
		totalRetryBudget:                      totalRetryBudget,
		enableStallRetriesOnStorageLayoutCall: true,
		enableStallRetriesOnAllCalls:          false,
	}
}

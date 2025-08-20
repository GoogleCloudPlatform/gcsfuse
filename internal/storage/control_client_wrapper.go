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
	"math/rand"
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
	defaultControlClientRetryDeadline    = 30 * time.Second
	defaultControlClientTotalRetryBudget = 5 * time.Minute
	defaultInitialBackoff                = 1 * time.Second
	defaultMaxBackoff                    = 1 * time.Minute
	defaultBackoffMultiplier             = 2.0
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

// exponentialBackoff holds the duration parameters for exponential backoff.
type exponentialBackoff struct {
	// Min duration for next backoff, which is same as the initial backoff.
	min time.Duration
	// Max duration returned for next back backoff i.e. Next().
	max time.Duration
	// The rate at which the backoff duration should grow
	// over subsequent calls to next().
	multiplier float64
	// Duration for next backoff. Capped at max. Returned by next().
	next time.Duration
}

func newBackoff(initialDuration, maxDuration time.Duration, multiplier float64) *exponentialBackoff {
	return &exponentialBackoff{
		min:        initialDuration,
		max:        maxDuration,
		multiplier: multiplier,
		next:       initialDuration,
	}
}

// nextDuration returns the next backoff duration.
func (b *exponentialBackoff) nextDuration() time.Duration {
	next := b.next
	b.next = min(b.max, time.Duration(float64(b.next)*b.multiplier))
	b.next = max(b.min, b.next)
	return next
}

// waitWithJitter waits for the next backoff duration with added jitter.
// The jitter adds randomness to the backoff duration to prevent the thundering herd problem.
// This is similar to how gax-retries backoff after each failed retry.
func (b *exponentialBackoff) waitWithJitter(ctx context.Context) error {
	nextDuration := b.nextDuration()
	jitteryBackoffDuration := time.Duration(max(int64(b.min), rand.Int63n(int64(nextDuration))))
	select {
	case <-time.After(jitteryBackoffDuration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// storageControlClientWithRetry is a wrapper for an existing StorageControlClient object
// which implements gcsfuse-level retry logic if any of the control-client call gets stalled or returns a retryable error.
// It makes time-bound attempts to call the underlying StorageControlClient methods,
// retrying on errors that should be retried according to gcsfuse's retry logic
type storageControlClientWithRetry struct {
	raw StorageControlClient

	// Time-limit to attempt the first call or a retried call.
	retryDeadline time.Duration
	// Total duration allowed across all the attempts.
	totalRetryBudget time.Duration

	// Backoff duration in-between retries.
	backoff *exponentialBackoff

	// Whether or not to enable retries for GetStorageLayout call.
	enableRetriesOnStorageLayoutCall bool
}

// executeWithRetry encapsulates the retry logic for control client operations.
// It performs time-bound, exponential backoff retries for a given API call.
// It is expected that the given apiCall returns a structure, and not an HTTP response,
// so that it does not leave behind any trace of a pending operation on server.
func executeWithRetry[T any](
	sccwros *storageControlClientWithRetry,
	ctx context.Context,
	operationName string,
	reqDescription string,
	apiCall func(attemptCtx context.Context) (T, error),
) (T, error) {
	var zero T

	parentCtx, cancel := context.WithTimeout(ctx, sccwros.totalRetryBudget)
	defer cancel()

	for {
		attemptCtx, attemptCancel := context.WithTimeout(parentCtx, sccwros.retryDeadline)

		logger.Tracef("Calling %s for %q with deadline=%v ...", operationName, reqDescription, sccwros.retryDeadline)
		result, err := apiCall(attemptCtx)
		// Cancel attemptCtx after it is no longer needed to free up its resources.
		attemptCancel()

		if err == nil {
			return result, nil
		}

		// If the error is not retryable, return it immediately.
		if !storageutil.ShouldRetry(err) {
			return zero, fmt.Errorf("%s for %q failed with a non-retryable error: %w", operationName, reqDescription, err)
		}

		// If the parent context is cancelled/timed-out, we should stop retrying.
		if parentCtx.Err() != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtx.Err())
		}

		// Do a jittery backoff after each retry.
		parentCtxErr := sccwros.backoff.waitWithJitter(parentCtx)
		if parentCtxErr != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtxErr)
		}
	}
}

func (sccwros *storageControlClientWithRetry) GetStorageLayout(ctx context.Context,
	req *controlpb.GetStorageLayoutRequest,
	opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	if !sccwros.enableRetriesOnStorageLayoutCall {
		return sccwros.raw.GetStorageLayout(ctx, req, opts...)
	}

	apiCall := func(attemptCtx context.Context) (*controlpb.StorageLayout, error) {
		return sccwros.raw.GetStorageLayout(attemptCtx, req, opts...)
	}

	return executeWithRetry(sccwros, ctx, "GetStorageLayout", req.Name, apiCall)
}

func (sccwros *storageControlClientWithRetry) DeleteFolder(ctx context.Context,
	req *controlpb.DeleteFolderRequest,
	opts ...gax.CallOption) error {
	return sccwros.raw.DeleteFolder(ctx, req, opts...)
}

func (sccwros *storageControlClientWithRetry) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	return sccwros.raw.GetFolder(ctx, req, opts...)
}

func (sccwros *storageControlClientWithRetry) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	return sccwros.raw.RenameFolder(ctx, req, opts...)
}

func (sccwros *storageControlClientWithRetry) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	return sccwros.raw.CreateFolder(ctx, req, opts...)
}

func newRetryWrapper(controlClient StorageControlClient, retryDeadline, totalRetryBudget, initialBackoff, maxBackoff time.Duration, backoffMultiplier float64) StorageControlClient {
	// Avoid creating a nested wrapper.
	raw := controlClient
	if sccwros, ok := controlClient.(*storageControlClientWithRetry); ok {
		raw = sccwros.raw
	}

	return &storageControlClientWithRetry{
		raw:                              raw,
		retryDeadline:                    retryDeadline,
		totalRetryBudget:                 totalRetryBudget,
		backoff:                          newBackoff(initialBackoff, maxBackoff, backoffMultiplier),
		enableRetriesOnStorageLayoutCall: true,
	}
}

// withRetryOnStorageLayout wraps a StorageControlClient to do a time-bound retry approach for retryable errors for the GetStorageLayout call through it.
func withRetryOnStorageLayout(controlClient StorageControlClient, retryDeadline time.Duration, totalRetryBudget time.Duration) StorageControlClient {
	return newRetryWrapper(controlClient, retryDeadline, totalRetryBudget, defaultInitialBackoff, defaultMaxBackoff, defaultBackoffMultiplier)
}

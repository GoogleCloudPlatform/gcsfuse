// Copyright 2023 Google LLC
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

package storageutil

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// retryAction defines the classification of a retry decision.
type retryAction int

const (
	// noRetry indicates the error is not retryable.
	noRetry retryAction = iota
	// retryTransient indicates the error is transient and retryable as per Go-SDK retry policy.
	retryTransient
	// retry401 indicates a 401 Unauthorized error which requires a retry due to credentials refresh.
	retry401
	// retryUnauthenticated indicates a gRPC Unauthenticated error which requires a retry due to credentials refresh.
	retryUnauthenticated
	// retry403 indicates a 403 PermissionDenied error retryable during mount operations.
	retry403
	// retry404BucketDoesNotExist indicates a 404 Bucket Not Found error retryable during mount operations.
	retry404BucketDoesNotExist
	// retryPermissionDenied indicates a gRPC PermissionDenied error retryable during mount operations.
	retryPermissionDenied
	// retryNotFoundBucketDoesNotExist indicates a gRPC NotFound bucket does not exist error retryable during mount operations.
	retryNotFoundBucketDoesNotExist
)

func determineRetryAction(err error) retryAction {
	if storage.ShouldRetry(err) {
		return retryTransient
	}

	// HTTP API errors (googleapi.Error)
	if typed, ok := err.(*googleapi.Error); ok {
		// HTTP 401 errors - Invalid Credentials
		// This is a work-around to fix the corner case where GCSFuse checks the token
		// as valid but GCS says invalid. This might be due to client-server timer
		// issues. Actual fix will be to refresh the token earlier than 1 hr.
		// Changes will be done post resolution of the below issue:
		// https://github.com/golang/oauth2/issues/623
		// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
		if typed.Code == 401 {
			return retry401
		}
		if typed.Code == 403 {
			return retry403
		}
		if typed.Code == 404 && isBucketNotFoundError(err) {
			return retry404BucketDoesNotExist
		}
	}

	// gRPC API errors (status.Status)
	if st, ok := status.FromError(err); ok {
		// gRPC UNAUTHENTICATED errors. See https://github.com/golang/oauth2/issues/623
		// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
		if st.Code() == codes.Unauthenticated {
			return retryUnauthenticated
		}
		if st.Code() == codes.PermissionDenied {
			return retryPermissionDenied
		}
		if st.Code() == codes.NotFound && isBucketNotFoundError(err) {
			return retryNotFoundBucketDoesNotExist
		}
	}
	return noRetry
}

// ShouldRetryWithoutLogging checks if the error is transient and should be retried.
// This method is same as ShouldRetry except it doesn't add warning logs.
func ShouldRetryWithoutLogging(err error) bool {
	action := determineRetryAction(err)
	return action == retryTransient || action == retry401 || action == retryUnauthenticated
}

// ShouldRetryWithRetryContext checks if the given error is transient and should be retried,
// logging the retry warning with RetryContext (operation, object, attempt, invocation ID).
// Returns true if the error is retryable, false otherwise.
func ShouldRetryWithRetryContext(err error, retryCtx *storage.RetryContext) bool {
	if !ShouldRetryWithoutLogging(err) {
		return false
	}
	if retryCtx == nil {
		logger.Warnf("Retrying for error: %v", err)
		return true
	}
	logger.Warnf("Retrying %s for %q: InvocationID: %s, Attempt: %d, due to error: %v",
		retryCtx.Operation, retryCtx.Object, retryCtx.InvocationID, retryCtx.Attempt+1, err)
	return true
}

func isBucketNotFoundError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "bucket does not exist")
}

// ShouldRetryOnMountWithRetryContext checks if the given error is retryable during mount operations.
// It first delegates to ShouldRetryWithRetryContext for standard transient error classification (logged at Warning level).
// If not a standard transient error, it checks for mount-specific retryable error conditions (such as metadata server
// connection refused, HTTP 403, HTTP 404 bucket does not exist, gRPC PermissionDenied, and gRPC NotFound bucket does not exist),
// logging them at Error level with RetryContext details.
// Returns true if the error is retryable, false otherwise.
func ShouldRetryOnMountWithRetryContext(err error, retryCtx *storage.RetryContext) bool {
	if ShouldRetryWithRetryContext(err, retryCtx) {
		return true
	}
	action := determineRetryAction(err)
	shouldRetry := false

	switch action {
	case retry404BucketDoesNotExist, retryNotFoundBucketDoesNotExist:
		shouldRetry = true
		logger.LogToStderr("GCSFuse Mounting: bucket does not exist")
	case retry403, retryPermissionDenied:
		shouldRetry = true
		logger.LogToStderr("GCSFuse Mounting: permission denied")
	}
	if !shouldRetry {
		return false
	}

	if retryCtx == nil {
		logger.Errorf("Retrying for error: %v", err)
	} else {
		logger.Errorf("Retrying %s for %q: InvocationID: %s, Attempt: %d, due to error: %v",
			retryCtx.Operation, retryCtx.Object, retryCtx.InvocationID, retryCtx.Attempt+1, err)
	}
	return true
}

func ShouldRetryWithMonitoringAndRetryContext(
	ctx context.Context,
	err error,
	retryCtx *storage.RetryContext,
	metricHandle metrics.MetricHandle,
) bool {
	if err == nil {
		return false
	}

	retry := ShouldRetryWithRetryContext(err, retryCtx)
	if !retry {
		return false
	}
	// Record metrics
	val := metrics.RetryErrorCategoryOTHERERRORSAttr
	if errors.Is(err, context.DeadlineExceeded) {
		val = metrics.RetryErrorCategorySTALLEDREADREQUESTAttr
	}

	metricHandle.GcsRetryCount(1, val)
	return retry
}

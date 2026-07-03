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
	// retry404BucketDoesNotExist indicates an HTTP 404 error where the bucket was not found during mount.
	retry404BucketDoesNotExist
	// retryNotFoundBucketDoesNotExist indicates a gRPC NotFound error where the bucket was not found during mount.
	retryNotFoundBucketDoesNotExist
	// retry403 indicates an HTTP 403 Permission Denied error during mount.
	retry403
	// retryPermissionDenied indicates a gRPC PermissionDenied error during mount.
	retryPermissionDenied
)

const errStrBucketNotExist = "bucket does not exist"

func determineRetryAction(err error) retryAction {
	if storage.ShouldRetry(err) {
		return retryTransient
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		// HTTP 401 errors - Invalid Credentials
		// This is a work-around to fix the corner case where GCSFuse checks the token
		// as valid but GCS says invalid. This might be due to client-server timer
		// issues. Actual fix will be refresh the token earlier than 1 hr.
		// Changes will be done post resolution of the below issue:
		// https://github.com/golang/oauth2/issues/623
		// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
		if apiErr.Code == 401 {
			return retry401
		}
		if apiErr.Code == 403 {
			return retry403
		}
		if apiErr.Code == 404 && strings.Contains(strings.ToLower(apiErr.Message), errStrBucketNotExist) {
			return retry404BucketDoesNotExist
		}
	}

	if status, ok := status.FromError(err); ok {
		// This is the same case as above, but for gRPC UNAUTHENTICATED errors. See
		// https://github.com/golang/oauth2/issues/623
		// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
		if status.Code() == codes.Unauthenticated {
			return retryUnauthenticated
		}
		if status.Code() == codes.PermissionDenied {
			return retryPermissionDenied
		}
		if status.Code() == codes.NotFound && strings.Contains(strings.ToLower(status.Message()), errStrBucketNotExist) {
			return retryNotFoundBucketDoesNotExist
		}
	}
	return noRetry
}

// ShouldRetryWithoutLogging checks if the error is transient and should be retried.
// This method is same as ShouldRetry except it doesn't add warning logs.
func ShouldRetryWithoutLogging(err error) bool {
	switch determineRetryAction(err) {
	case retryTransient, retry401, retryUnauthenticated:
		return true
	default:
		return false
	}
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

// ShouldRetryOnMount checks if the error is retryable during mount initialization.
// In addition to standard transient errors, it retries HTTP 403/404 and gRPC PermissionDenied/NotFound errors.
func ShouldRetryOnMount(err error) bool {
	return determineRetryAction(err) != noRetry
}

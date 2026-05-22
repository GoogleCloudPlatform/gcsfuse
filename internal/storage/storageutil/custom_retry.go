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
)

func determineRetryAction(err error) retryAction {
	if storage.ShouldRetry(err) {
		return retryTransient
	}

	// HTTP 401 errors - Invalid Credentials
	// This is a work-around to fix the corner case where GCSFuse checks the token
	// as valid but GCS says invalid. This might be due to client-server timer
	// issues. Actual fix will be refresh the token earlier than 1 hr.
	// Changes will be done post resolution of the below issue:
	// https://github.com/golang/oauth2/issues/623
	// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == 401 {
			return retry401
		}
	}

	// This is the same case as above, but for gRPC UNAUTHENTICATED errors. See
	// https://github.com/golang/oauth2/issues/623
	// TODO(b/518674297): Please incorporate the correct fix post resolution of the above issue.
	if status, ok := status.FromError(err); ok {
		if status.Code() == codes.Unauthenticated {
			return retryUnauthenticated
		}
	}
	return noRetry
}

// ShouldRetryWithoutLogging checks if the error is transient and should be retried.
// This method is same as ShouldRetry except it doesn't add warning logs.
func ShouldRetryWithoutLogging(err error) bool {
	return determineRetryAction(err) != noRetry
}

// ShouldRetryWithRetryContext checks if the given error is transient and should be retried,
// logging the retry warning with RetryContext (operation, object, attempt, invocation ID).
// Returns true if the error is retryable, false otherwise.
func ShouldRetryWithRetryContext(err error, retryCtx *storage.RetryContext) bool {
	if ShouldRetryWithoutLogging(err) {
		if retryCtx != nil {
			logger.Warnf("Retrying %s for %q: InvocationID: %s, Attempt: %d, due to error: %v",
				retryCtx.Operation, retryCtx.Object, retryCtx.InvocationID, retryCtx.Attempt+1, err)
		} else {
			logger.Warnf("Retrying for error: %v", err)
		}
		return true
	}
	if retryCtx != nil {
		logger.Errorf("%s for %q failed: InvocationID: %s, Attempt: %d, with error: %v",
			retryCtx.Operation, retryCtx.Object, retryCtx.InvocationID, retryCtx.Attempt, err)
	}
	return false
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

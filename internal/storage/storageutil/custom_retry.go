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
	"fmt"

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

// ShouldRetry checks if the given error is transient and should be retried.
// It logs a warning message with the error details and execution context (TraceID, remaining budget) for any retryable error.
// Returns true if the error is retryable, false otherwise.
func ShouldRetry(ctx context.Context, err error) bool {
	return ShouldRetryWithContext(ctx, err, nil)
}

// ShouldRetryWithContext checks if the given error is transient and should be retried,
// logging the retry warning with rich context from both the execution context (TraceID, remaining budget)
// and the Go Storage SDK's experimental RetryContext (operation, bucket, object, attempt, invocation ID).
func ShouldRetryWithContext(ctx context.Context, err error, retryCtx *storage.RetryContext) bool {
	switch determineRetryAction(err) {
	case retryTransient:
		logRetryWithContext("Retrying for transient error", err, retryCtx)
		return true
	case retry401:
		logRetryWithContext("Retrying for error-code 401", err, retryCtx)
		return true
	case retryUnauthenticated:
		logRetryWithContext("Retrying for UNAUTHENTICATED error", err, retryCtx)
		return true
	default:
		return false
	}
}

func ShouldRetryWithMonitoring(ctx context.Context, err error, metricHandle metrics.MetricHandle) bool {
	return ShouldRetryWithMonitoringAndContext(ctx, err, nil, metricHandle)
}

func ShouldRetryWithMonitoringAndContext(
	ctx context.Context,
	err error,
	retryCtx *storage.RetryContext,
	metricHandle metrics.MetricHandle,
) bool {
	if err == nil {
		return false
	}

	retry := ShouldRetryWithContext(ctx, err, retryCtx)
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

func logRetryWithContext(prefix string, err error, retryCtx *storage.RetryContext) {
	var errorDetails string
	if typed, ok := err.(*googleapi.Error); ok {
		errorDetails = fmt.Sprintf(" [HTTP Code: %d, Message: %q]", typed.Code, typed.Message)
	} else if st, ok := status.FromError(err); ok {
		errorDetails = fmt.Sprintf(" [gRPC Code: %s, Message: %q]", st.Code().String(), st.Message())
	}

	var sdkRetryInfo string
	if retryCtx != nil {
		sdkRetryInfo = fmt.Sprintf(" [Op: %s, Bucket: %q, Object: %q, Attempt: %d, InvocationID: %s]",
			retryCtx.Operation, retryCtx.Bucket, retryCtx.Object, retryCtx.Attempt, retryCtx.InvocationID)
	}

	logger.Warnf("%s: %v%s%s", prefix, err, errorDetails, sdkRetryInfo)
}

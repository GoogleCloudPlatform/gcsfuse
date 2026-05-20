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
	// retryUnauthenticated indicates a gRPC Unauthenticated error which requires a retry.
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
	// TODO: Please incorporate the correct fix post resolution of the above issue.
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == 401 {
			return retry401
		}
	}

	// This is the same case as above, but for gRPC UNAUTHENTICATED errors. See
	// https://github.com/golang/oauth2/issues/623
	// TODO: Please incorporate the correct fix post resolution of the above issue.
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
// It logs a warning message with the error details for any retryable error.
// Returns true if the error is retryable, false otherwise.
func ShouldRetry(err error) bool {
	switch determineRetryAction(err) {
	case retryTransient:
		logger.Warnf("Retrying for the error: %v", err)
		return true
	case retry401:
		logger.Warnf("Retrying for error-code 401: %v", err)
		return true
	case retryUnauthenticated:
		logger.Warnf("Retrying for UNAUTHENTICATED error: %v", err)
		return true
	default:
		return false
	}
}

func ShouldRetryWithMonitoring(ctx context.Context, err error, metricHandle metrics.MetricHandle) bool {
	if err == nil {
		return false
	}

	retry := ShouldRetry(err)
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

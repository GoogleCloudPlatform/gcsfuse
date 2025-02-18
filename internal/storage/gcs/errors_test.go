// Copyright 2025 Google LLC
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

package gcs

import (
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"errors"
)

func TestGetGCSError(t *testing.T) {
	preconditionAPIErr, ok := apierror.FromError(status.Error(codes.FailedPrecondition, codes.FailedPrecondition.String()))
	assert.True(t, ok)

	notFoundAPIErr, ok := apierror.FromError(status.Error(codes.NotFound, codes.NotFound.String()))
	assert.True(t, ok)

	otherAPIErr, ok := apierror.FromError(status.Error(codes.Internal, codes.Internal.String()))
	assert.True(t, ok)

	// TODO: to check - why does wrapped_grpc_status_NotFound test fails without same error in input and output?
	grpcNotFoundErr := status.Error(codes.NotFound, "not found")
	assert.True(t, ok)

	testCases := []struct {
		name        string
		inputErr    error
		expectedErr error
	}{
		{
			name:        "nil_error",
			inputErr:    nil,
			expectedErr: nil,
		},
		{
			name:        "googleapi.Error_NotFound",
			inputErr:    &googleapi.Error{Code: http.StatusNotFound},
			expectedErr: &NotFoundError{Err: &googleapi.Error{Code: http.StatusNotFound}},
		},
		{
			name:        "googleapi.Error_PreconditionFailed",
			inputErr:    &googleapi.Error{Code: http.StatusPreconditionFailed},
			expectedErr: &PreconditionError{Err: &googleapi.Error{Code: http.StatusPreconditionFailed}},
		},
		{
			name:        "googleapi.Error_other_code",
			inputErr:    &googleapi.Error{Code: http.StatusBadRequest},
			expectedErr: &googleapi.Error{Code: http.StatusBadRequest},
		},
		{
			name:        "wrapped_googleapi.Error_NotFound",
			inputErr:    fmt.Errorf("wrapped: %w", &googleapi.Error{Code: http.StatusNotFound}),
			expectedErr: &NotFoundError{Err: fmt.Errorf("wrapped: %w", &googleapi.Error{Code: http.StatusNotFound})},
		},
		{
			name:        "grpc_status_NotFound",
			inputErr:    status.Error(codes.NotFound, "not found"),
			expectedErr: &NotFoundError{Err: status.Error(codes.NotFound, "not found")},
		},
		{
			name:        "grpc_status_FailedPrecondition",
			inputErr:    status.Error(codes.FailedPrecondition, "failed precondition"),
			expectedErr: &PreconditionError{Err: status.Error(codes.FailedPrecondition, "failed precondition")},
		},
		{
			name:        "grpc_status_other_code",
			inputErr:    status.Error(codes.Internal, "internal error"),
			expectedErr: status.Error(codes.Internal, "internal error"),
		},
		{
			name:        "other_error",
			inputErr:    errors.New("some error"),
			expectedErr: errors.New("some error"),
		},
		{
			name:        "wrapped_grpc_status_NotFound",
			inputErr:    fmt.Errorf("wrapped: %w", grpcNotFoundErr),
			expectedErr: &NotFoundError{Err: fmt.Errorf("wrapped: %w", grpcNotFoundErr)},
		},
		{
			name:        "GCS_Precondition_error",
			inputErr:    &PreconditionError{Err: errors.New("precondition error")},
			expectedErr: &PreconditionError{Err: errors.New("precondition error")},
		},
		{
			name:        "GCS_NotFound_error",
			inputErr:    &NotFoundError{Err: errors.New("not found error")},
			expectedErr: &NotFoundError{Err: errors.New("not found error")},
		},
		{
			name:        "wrapped_GCS_Precondition_error",
			inputErr:    fmt.Errorf("wrapped: %w", &PreconditionError{Err: errors.New("precondition error")}),
			expectedErr: fmt.Errorf("wrapped: %w", &PreconditionError{Err: errors.New("precondition error")}),
		},
		{
			name:        "storage_object_not_exist",
			inputErr:    storage.ErrObjectNotExist,
			expectedErr: &NotFoundError{Err: storage.ErrObjectNotExist},
		},
		{
			name:        "precondition_apierror",
			inputErr:    preconditionAPIErr,
			expectedErr: &PreconditionError{Err: preconditionAPIErr},
		},
		{
			name:        "notfound_apierror",
			inputErr:    notFoundAPIErr,
			expectedErr: &NotFoundError{Err: notFoundAPIErr},
		},
		{
			name:        "other_apierror",
			inputErr:    otherAPIErr,
			expectedErr: otherAPIErr,
		},
		{
			name:        "wrapped_precondition_apierror",
			inputErr:    fmt.Errorf("wrapped: %w", preconditionAPIErr),
			expectedErr: &PreconditionError{Err: fmt.Errorf("wrapped: %w", preconditionAPIErr)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetGCSError(tc.inputErr)
			assert.Equal(t, tc.expectedErr, got)
		})
	}
}

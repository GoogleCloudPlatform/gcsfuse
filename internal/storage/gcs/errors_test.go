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

func TestWrapUnderCommonGCSError(t *testing.T) {
	preconditionAPIErr, ok := apierror.FromError(status.Error(codes.FailedPrecondition, codes.FailedPrecondition.String()))
	assert.True(t, ok)

	notFoundAPIErr, ok := apierror.FromError(status.Error(codes.NotFound, codes.NotFound.String()))
	assert.True(t, ok)

	otherAPIErr, ok := apierror.FromError(status.Error(codes.Internal, codes.Internal.String()))
	assert.True(t, ok)

	testCases := []struct {
		name            string
		inputErr        error
		expectedErr     error
		expectedWrapped bool
	}{
		{
			name:            "nil_error",
			inputErr:        nil,
			expectedErr:     nil,
			expectedWrapped: false,
		},
		{
			name:            "googleapi.Error_NotFound",
			inputErr:        &googleapi.Error{Code: http.StatusNotFound},
			expectedErr:     &NotFoundError{Err: &googleapi.Error{Code: http.StatusNotFound}},
			expectedWrapped: true,
		},
		{
			name:            "googleapi.Error_PreconditionFailed",
			inputErr:        &googleapi.Error{Code: http.StatusPreconditionFailed},
			expectedErr:     &PreconditionError{Err: &googleapi.Error{Code: http.StatusPreconditionFailed}},
			expectedWrapped: true,
		},
		{
			name:            "googleapi.Error_other_code",
			inputErr:        &googleapi.Error{Code: http.StatusBadRequest},
			expectedErr:     &googleapi.Error{Code: http.StatusBadRequest},
			expectedWrapped: false,
		},
		{
			name:            "grpc_status_NotFound",
			inputErr:        status.Error(codes.NotFound, "not found"),
			expectedErr:     &NotFoundError{Err: status.Error(codes.NotFound, "not found")},
			expectedWrapped: true,
		},
		{
			name:            "grpc_status_FailedPrecondition",
			inputErr:        status.Error(codes.FailedPrecondition, "failed precondition"),
			expectedErr:     &PreconditionError{Err: status.Error(codes.FailedPrecondition, "failed precondition")},
			expectedWrapped: true,
		},
		{
			name:            "grpc_status_other_code",
			inputErr:        status.Error(codes.Internal, "internal error"),
			expectedErr:     status.Error(codes.Internal, "internal error"),
			expectedWrapped: false,
		},
		{
			name:            "other_error",
			inputErr:        errors.New("some error"),
			expectedErr:     errors.New("some error"),
			expectedWrapped: false,
		},
		{
			name:            "GCS_Precondition_error",
			inputErr:        &PreconditionError{Err: errors.New("precondition error")},
			expectedErr:     &PreconditionError{Err: errors.New("precondition error")},
			expectedWrapped: false,
		},
		{
			name:            "GCS_NotFound_error",
			inputErr:        &NotFoundError{Err: errors.New("not found error")},
			expectedErr:     &NotFoundError{Err: errors.New("not found error")},
			expectedWrapped: false,
		},
		{
			name:            "Storage_object_not_exist",
			inputErr:        storage.ErrObjectNotExist,
			expectedErr:     &NotFoundError{Err: storage.ErrObjectNotExist},
			expectedWrapped: true,
		},
		{
			name:            "precondition_apierror",
			inputErr:        preconditionAPIErr,
			expectedErr:     &PreconditionError{Err: preconditionAPIErr},
			expectedWrapped: true,
		},
		{
			name:            "notfound_apierror",
			inputErr:        notFoundAPIErr,
			expectedErr:     &NotFoundError{Err: notFoundAPIErr},
			expectedWrapped: true,
		},
		{
			name:            "other_apierror",
			inputErr:        otherAPIErr,
			expectedErr:     otherAPIErr,
			expectedWrapped: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, wrapped := WrapUnderCommonGCSError(tc.inputErr)
			assert.Equal(t, tc.expectedWrapped, wrapped)
			assert.Equal(t, tc.expectedErr, got)

		})
	}
}

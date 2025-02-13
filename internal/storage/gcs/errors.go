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

package gcs

import (
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A *NotFoundError value is an error that indicates an object name or a
// particular generation for that name were not found.
type NotFoundError struct {
	Err error
}

func (nfe *NotFoundError) Error() string {
	return fmt.Sprintf("gcs.NotFoundError: %v", nfe.Err)
}

// A *PreconditionError value is an error that indicates a precondition failed.
type PreconditionError struct {
	Err error
}

// Returns pe.Err.Error().
func (pe *PreconditionError) Error() string {
	return fmt.Sprintf("gcs.PreconditionError: %v", pe.Err)
}

// WrapGCSFuseError converts an error returned by go-sdk into gcsfuse specific gcs error.
// It returns converted error and a boolean indicating whether a conversion occured or not.
func WrapGCSFuseError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}

	// Http client error.
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case http.StatusNotFound:
			return &NotFoundError{Err: err}, true
		case http.StatusPreconditionFailed:
			return &PreconditionError{Err: err}, true
		}

	}

	// Control client error.
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.GRPCStatus().Code() {
		case codes.NotFound:
			return &NotFoundError{Err: err}, true
		case codes.FailedPrecondition:
			return &PreconditionError{Err: err}, true
		}
	}

	// RPC error.
	if rpcErr, ok := status.FromError(err); ok {
		switch rpcErr.Code() {
		case codes.NotFound:
			return &NotFoundError{Err: err}, true
		case codes.FailedPrecondition:
			return &PreconditionError{Err: err}, true
		}
	}

	if err == storage.ErrObjectNotExist {
		return &NotFoundError{Err: err}, true
	}

	return err, false
}

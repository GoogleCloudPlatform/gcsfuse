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

func (nfe *NotFoundError) Unwrap() error {
	return nfe.Err
}

// A *PreconditionError value is an error that indicates a precondition failed.
type PreconditionError struct {
	Err error
}

// Returns pe.Err.Error().
func (pe *PreconditionError) Error() string {
	return fmt.Sprintf("gcs.PreconditionError: %v", pe.Err)
}

// ObjectTooLargeError indicates that an object exceeds the maximum size supported by the upload strategy.
type ObjectTooLargeError struct {
	Err error
}

func (e *ObjectTooLargeError) Error() string {
	return fmt.Sprintf("gcs.ObjectTooLargeError: %v", e.Err)
}

func (e *ObjectTooLargeError) Unwrap() error {
	return e.Err
}

// ErrorInfo domain and reasons that GCS returns when an upload exceeds the
// maximum object size.
const (
	gcsErrorDomain                    = "storage.googleapis.com"
	reasonSegmentNumberOutOfRange     = "GRPC_SEGMENT_NUMBER_OUT_OF_RANGE"
	reasonObjectSizeExceedsGCSMaximum = "GRPC_OBJECT_SIZE_EXCEEDS_GCS_MAXIMUM"
)

// GetParallelUploadError converts an error into an *ObjectTooLargeError when
// GCS reports that the upload exceeds the maximum object size. Other errors are
// returned unchanged.
func GetParallelUploadError(err error) error {
	var objectTooLargeErr *ObjectTooLargeError
	if err == nil || errors.As(err, &objectTooLargeErr) {
		return err
	}
	if apiErr, ok := apierror.FromError(err); ok && apiErr.Domain() == gcsErrorDomain {
		switch apiErr.Reason() {
		case reasonSegmentNumberOutOfRange, reasonObjectSizeExceedsGCSMaximum:
			return &ObjectTooLargeError{Err: err}
		}
	}
	return err
}

// GetGCSError converts an error returned by go-sdk into gcsfuse specific common gcs error.
func GetGCSError(err error) error {
	if err == nil {
		return nil
	}

	// Http client error.
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case http.StatusNotFound:
			return &NotFoundError{Err: err}
		case http.StatusPreconditionFailed:
			return &PreconditionError{Err: err}
		}
	}

	// RPC error (all gRPC client including control client).
	if rpcErr, ok := status.FromError(err); ok {
		switch rpcErr.Code() {
		case codes.NotFound:
			return &NotFoundError{Err: err}
		case codes.FailedPrecondition:
			return &PreconditionError{Err: err}
		}
	}

	// If storage object doesn't exist, go-sdk returns as ErrObjectNotExist.
	// Important to note: currently go-sdk doesn't format/convert error coming from the control-client.
	// Ref: http://shortn/_CY9Jyqf2wF
	if errors.Is(err, storage.ErrObjectNotExist) {
		return &NotFoundError{Err: err}
	}

	return err
}

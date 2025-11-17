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

package wrappers

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"

	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorMapping struct {
	suite.Suite
	preconditionErrCfg bool
}

func TestWithErrorMapping(testSuite *testing.T) {
	suite.Run(testSuite, new(ErrorMapping))
}

func (testSuite *ErrorMapping) TestPermissionDeniedGrpcApiError() {
	statusErr := status.New(codes.PermissionDenied, "Permission denied")
	apiError, _ := apierror.FromError(statusErr.Err())

	fsErr := errno(apiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.EACCES, fsErr)
}

func (testSuite *ErrorMapping) TestAlreadyExistGrpcApiError() {
	statusErr := status.New(codes.AlreadyExists, "already exist")
	apiError, _ := apierror.FromError(statusErr.Err())

	fsErr := errno(apiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.EEXIST, fsErr)
}

func (testSuite *ErrorMapping) TestNotFoundGrpcApiError() {
	statusErr := status.New(codes.NotFound, "Not found")
	apiError, _ := apierror.FromError(statusErr.Err())

	fsErr := errno(apiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.ENOENT, fsErr)
}

func (testSuite *ErrorMapping) TestCanceledGrpcApiError() {
	statusErr := status.New(codes.Canceled, "Canceled error")
	apiError, _ := apierror.FromError(statusErr.Err())

	fsErr := errno(apiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.EINTR, fsErr)
}

func (testSuite *ErrorMapping) TestUnAuthenticatedGrpcApiError() {
	statusErr := status.New(codes.Unauthenticated, "UnAuthenticated error")
	apiError, _ := apierror.FromError(statusErr.Err())

	fsErr := errno(apiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.EACCES, fsErr)
}

func (testSuite *ErrorMapping) TestUnAuthenticatedHttpGoogleApiError() {
	googleApiError := &googleapi.Error{Code: http.StatusUnauthorized}
	googleApiError.Wrap(fmt.Errorf("UnAuthenticated error"))

	fsErr := errno(googleApiError, testSuite.preconditionErrCfg)

	assert.Equal(testSuite.T(), syscall.EACCES, fsErr)
}

func (testSuite *ErrorMapping) TestFileClobberedErrorWithPreconditionErrCfg() {
	clobberedErr := &gcsfuse_errors.FileClobberedError{
		Err:        fmt.Errorf("some error"),
		ObjectName: "foo.txt",
	}

	gotErrno := errno(clobberedErr, true)

	assert.Equal(testSuite.T(), syscall.ESTALE, gotErrno)
}

func (testSuite *ErrorMapping) TestFileClobberedErrorWithoutPreconditionErrCfg() {
	clobberedErr := &gcsfuse_errors.FileClobberedError{
		Err:        fmt.Errorf("some error"),
		ObjectName: "foo.txt",
	}

	gotErrno := errno(clobberedErr, false)

	assert.Equal(testSuite.T(), nil, gotErrno)
}

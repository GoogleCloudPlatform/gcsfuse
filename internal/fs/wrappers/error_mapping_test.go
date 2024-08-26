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
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ErrorMapping struct {
	suite.Suite
}

func TestWithErrorMapping(testSuite *testing.T) {
	suite.Run(testSuite, new(ErrorMapping))
}

func (testSuite *ErrorMapping) TestPermissionDeniedError() {
	err := fmt.Errorf("rpc error: code = PermissionDenied desc = xxx does not have storage.folders.create access to the Google Cloud Storage bucket. Permission denied on resource (or it may not exist).")

	fsErr := errno(err)

	assert.Equal(testSuite.T(), syscall.EACCES, fsErr)
}

func (testSuite *ErrorMapping) TestNotFoundError() {
	err := fmt.Errorf("rpc error: code = NotFound desc = The folder does not exist.")

	fsErr := errno(err)

	assert.Equal(testSuite.T(), syscall.ENOENT, fsErr)
}

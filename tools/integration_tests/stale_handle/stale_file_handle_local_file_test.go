// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stale_handle

import (
	"os"
	"path"
	"syscall"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

type staleFileHandleLocalFileStreamingWrites struct {
	staleFileHandleLocalFile
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFileStreamingWrites) validate(err error) {
	require.NoError(s.T(), err)
}

func (s *staleFileHandleLocalFile) SetupTest() {
	s.testDirPath = setup.SetupTestDirectory(s.T().Name())
	// Create a local file.
	var err error
	s.f1, err = os.OpenFile(path.Join(s.testDirPath, FileName1), os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, operations.FilePermission_0600)
	assert.NoError(s.T(), err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, s.T().Name(), FileName1, s.T())
	s.data = setup.GenerateRandomString(operations.MiB * 5)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	for _, flags := range flagsSet {
		ts := new(staleFileHandleLocalFile)
		ts.validator = ts
		ts.flags = flags
		suite.Run(t, ts)
	}
}

func TestStaleFileHandleLocalFileStreamingWritesTest(t *testing.T) {
	for _, flags := range flagsSetStreamingWrites {
		ts := new(staleFileHandleLocalFileStreamingWrites)
		ts.validator = ts
		ts.flags = flags
		suite.Run(t, ts)
	}
}

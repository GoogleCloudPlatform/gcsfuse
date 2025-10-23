// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stale_handle

import (
	"path"
	"slices"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFileTest struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFileTest) SetupTest() {
	// Create a local file.
	s.fileName = path.Base(s.T().Name()) + setup.GenerateRandomString(5)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(testEnv.testDirPath, s.fileName))
	s.isLocal = true
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	// Run tests for mounted directory if the flag is set and return.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, new(staleFileHandleLocalFileTest))
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		s := new(staleFileHandleLocalFileTest)
		s.flags = flags
		s.isStreamingWritesEnabled = !slices.Contains(s.flags, "--enable-streaming-writes=false")
		suite.Run(t, s)
	}
}

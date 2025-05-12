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
	"path"
	"slices"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleLocalFile struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (s *staleFileHandleLocalFile) SetupTest() {
	s.testDirPath = setup.SetupTestDirectory(s.T().Name())
	// Create a local file.
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(s.testDirPath, FileName1))
	s.isLocal = true
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestStaleFileHandleLocalFileTest(t *testing.T) {
	for _, flags := range flagsSet {
		s := new(staleFileHandleLocalFile)
		s.flags = flags
		s.isStreamingWritesEnabled = slices.Contains(s.flags, "--enable-streaming-writes=true")
		suite.Run(t, s)
	}
}

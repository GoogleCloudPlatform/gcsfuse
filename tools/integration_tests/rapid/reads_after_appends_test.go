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

package rapid

import (
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Tests for the SingleMountReadsTestSuite
////////////////////////////////////////////////////////////////////////

// runAppendAndReadTest contains the core test logic for the SingleMountReadsTestSuite.
func (t *SingleMountReadsTestSuite) runAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.primaryMount.testDirPath, t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	for i := range numAppends {
		// Wait for a minute for stat to return the correct file size, which is needed by appendToFile.
		if i > 0 {
			time.Sleep(operations.WaitDurationAfterFlushZB)
		}

		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		sizeAfterAppend := len(t.fileContent)

		// For same-mount appends/reads, file size is always current.
		verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
	}
}

func (t *SingleMountReadsTestSuite) TestSequentialRead() {
	t.runAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *SingleMountReadsTestSuite) TestRandomRead() {
	t.runAppendAndReadTest(readRandomlyAndVerify)
}

////////////////////////////////////////////////////////////////////////
// Tests for the DualMountReadsTestSuite
////////////////////////////////////////////////////////////////////////

// runAppendAndReadTest contains the core test logic for the DualMountReadsTestSuite.
func (t *DualMountReadsTestSuite) runAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.getAppendPath(), t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	for i := range numAppends {
		sizeBeforeAppend := len(t.fileContent)
		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		sizeAfterAppend := len(t.fileContent)

		// If metadata cache is enabled, gcsfuse reads up to the cached file size.
		// The initial read (i=0) bypasses cache, seeing the latest file size.
		if !t.isMetadataCacheEnabled() || (i == 0) {
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
		} else {
			// Read only up to the cached file size (before append).
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeBeforeAppend]))

			// Wait for metadata cache to expire to fetch the latest size for the next read.
			// Metadata update for appends in current iteration itself takes a minute, so the
			// cached size will expire in ttl-60 secs from now, so wait accordingly.
			time.Sleep(time.Duration(metadataCacheTTLSecs*time.Second - operations.WaitDurationAfterFlushZB))
			// Expect read up to the latest file size which is the size after the append.
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
		}
	}
}

func (t *DualMountReadsTestSuite) TestSequentialRead() {
	t.runAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *DualMountReadsTestSuite) TestRandomRead() {
	t.runAppendAndReadTest(readRandomlyAndVerify)
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

func TestSingleMountReadsTestSuite(t *testing.T) {
	RunTests(t, "TestSingleMountReadsTestSuite", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &SingleMountReadsTestSuite{BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags}}
	})
}

func TestDualMountReadsTestSuiteWithMetadataCache(t *testing.T) {
	RunTests(t, "TestDualMountReadsTestSuiteWithMetadataCache", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &DualMountReadsTestSuite{BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags, metadataCacheEnabled: true}}
	})
}

func TestDualMountReadsTestSuiteWithoutMetadataCache(t *testing.T) {
	RunTests(t, "TestDualMountReadsTestSuiteWithoutMetadataCache", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &DualMountReadsTestSuite{BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags, metadataCacheEnabled: false}}
	})
}

// Copyright 2026 Google LLC
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

package rapid_operations

import (
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func (t *FinalizeRapidWritesTestSuite) TestFileClosedInFinalizedState() {
	if !t.isFinalizeEnabled {
		t.T().Skip("Skipping test since finalize-file-for-rapid is false")
	}
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
	defer t.deleteUnfinalizedObject()
	initialContent := "test content"
	bucket := testEnv.storageClient.Bucket(testEnv.cfg.TestBucket)
	objHandle := bucket.Object(path.Join(testDirName, t.fileName))

	operations.CreateFileWithContent(filePath, operations.FilePermission_0600, initialContent, t.T())

	attrs, err := objHandle.Attrs(testEnv.ctx)
	require.NoError(t.T(), err)
	assert.False(t.T(), attrs.Finalized.IsZero(), "Finalized field should not be zero for a finalized object")
}

func (t *FinalizeRapidWritesTestSuite) TestFileClosedInUnfinalizedState() {
	if t.isFinalizeEnabled {
		t.T().Skip("Skipping test since finalize-file-for-rapid is true")
	}
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
	defer t.deleteUnfinalizedObject()
	initialContent := "test content"
	bucket := testEnv.storageClient.Bucket(testEnv.cfg.TestBucket)
	objHandle := bucket.Object(path.Join(testDirName, t.fileName))

	operations.CreateFileWithContent(filePath, operations.FilePermission_0600, initialContent, t.T())

	attrs, err := objHandle.Attrs(testEnv.ctx)
	require.NoError(t.T(), err)
	assert.True(t.T(), attrs.Finalized.IsZero(), "Finalized field should be zero for an unfinalized object")
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

func TestFinalizeRapidWritesTestSuite(t *testing.T) {
	RunTests(t, "TestFinalizeRapidWritesTestSuite", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &FinalizeRapidWritesTestSuite{
			BaseSuite:         BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags},
			isFinalizeEnabled: strings.Contains(strings.Join(primaryFlags, " "), "finalize-file-for-rapid=true"),
		}
	})
}

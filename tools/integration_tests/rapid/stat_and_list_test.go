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

package rapid

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func (t *StatAndListTestSuite) TestStatOfNewFile() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
	defer t.deleteUnfinalizedObject()
	expectedSize := int64(len("test content"))
	client.CreateFinalizedObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, testDirName, t.fileName, "test content", t.T())

	size := operations.RetryUntil(testEnv.ctx, t.T(), 2*time.Second, defaultMetadataCacheTTL, func() (int64, error) {
		fi, err := os.Stat(filePath)
		if err != nil {
			return 0, err
		}
		if fi.Size() != expectedSize {
			return 0, fmt.Errorf("expected size %d, got %d", expectedSize, fi.Size())
		}
		return fi.Size(), nil
	})

	require.Equal(t.T(), expectedSize, size)
}

func (t *StatAndListTestSuite) TestListOfNewFile() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	defer t.deleteUnfinalizedObject()
	expectedSize := int64(len("test content"))
	client.CreateFinalizedObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, testDirName, t.fileName, "test content", t.T())

	size := operations.RetryUntil(testEnv.ctx, t.T(), 2*time.Second, defaultMetadataCacheTTL, func() (int64, error) {
		entries, err := os.ReadDir(t.primaryMount.testDirPath)
		if err != nil {
			return 0, err
		}
		for _, entry := range entries {
			if entry.Name() == t.fileName {
				info, err := entry.Info()
				if err != nil {
					return 0, err
				}
				if info.Size() != expectedSize {
					return 0, fmt.Errorf("listing expected size %d, got %d", expectedSize, info.Size())
				}
				return info.Size(), nil
			}
		}
		return 0, fmt.Errorf("file not found in directory listing")
	})

	require.Equal(t.T(), expectedSize, size)
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

func TestStatAndListTestSuite(t *testing.T) {
	RunTests(t, "TestStatAndListTestSuite", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		return &StatAndListTestSuite{BaseSuite{primaryFlags: primaryFlags, secondaryFlags: secondaryFlags}}
	})
}

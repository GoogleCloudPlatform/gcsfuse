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

package dummy_test_package

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type dummyFeatureSuite struct {
	flags   []string
	testDir string
	suite.Suite
}

func (s *dummyFeatureSuite) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
}

func (s *dummyFeatureSuite) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *dummyFeatureSuite) SetupTest() {
	// Generate random safe target path.
	s.testDir = testDirName + setup.GenerateRandomString(5)
	testEnv.testDirPath = setup.SetupTestDirectory(s.testDir)
}

func (s *dummyFeatureSuite) TearDownTest() {
	// Auto-save logs in Kokoro or local directories upon execution failure.
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

// TestScenarioExample demonstrates Arrange-Act-Assert integration tests behavior.
func (s *dummyFeatureSuite) TestScenarioExample() {
	// --- Arrange ---
	targetDir := path.Join(testEnv.testDirPath, "arrange_folder")
	operations.CreateDirectory(targetDir, s.T())
	targetFile := path.Join(targetDir, "test_item")
	testContent := "hello integration test"

	// --- Act ---
	operations.CreateFileWithContent(targetFile, setup.FilePermission_0600, testContent, s.T())
	content, err := operations.ReadFile(targetFile)

	// --- Assert ---
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), testContent, string(content))

	// Safe OS Check assertions
	_, err = os.Stat(targetFile)
	assert.NoError(s.T(), err)
}

func TestDummyFeatureSuite(t *testing.T) {
	suiteTarget := &dummyFeatureSuite{}

	// If GKE environment is active, execute a single direct pass.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, suiteTarget)
		return
	}

	// For local test execution, fetch the dynamic flag configurations.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())

	// Loop and execute suite scenarios per flag configuration item.
	for _, suiteTarget.flags = range flagsSet {
		log.Printf("Running suite scenario with flags: %s", suiteTarget.flags)
		suite.Run(t, suiteTarget)
	}
}

// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package refac_rapid_appends

import (
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Suite Definitions
// //////////////////////////////////////////////////////////////////////

type mountPoint struct {
	rootDir     string
	mntDir      string
	testDirPath string
	logFilePath string
}

// BaseSuite provides the common structure and configuration-driven setup logic.
type BaseSuite struct {
	suite.Suite
	cfg            *testConfig
	primaryMount   mountPoint
	secondaryMount mountPoint
	fileName       string
	fileContent    string
}

// ReadsTestSuite groups all tests related to reading after appends.
type ReadsTestSuite struct{ BaseSuite }

// AppendsTestSuite groups general tests for append behavior.
type AppendsTestSuite struct{ BaseSuite }

// //////////////////////////////////////////////////////////////////////
// Common Suite Logic
// //////////////////////////////////////////////////////////////////////

// SetupTest is run before each test, configuring and mounting gcsfuse based on the suite's config.
func (s *BaseSuite) SetupTest() {
	s.primaryMount.setupTestDir()
	s.mountPrimaryMount(s.cfg.primaryMountFlags)

	if s.cfg.isDualMount {
		s.secondaryMount.setupTestDir()
		s.mountSecondaryMount(s.cfg.secondaryMountFlags)
	}
}

// TearDownTest unmounts everything after each test.
func (s *BaseSuite) TearDownTest() {
	s.unmountPrimaryMount()
	if s.cfg.isDualMount {
		s.unmountSecondaryMount()
	}
}

// //////////////////////////////////////////////////////////////////////
// General Helpers
// //////////////////////////////////////////////////////////////////////

func (mnt *mountPoint) setupTestDir() {
	setup.SetUpTestDirForTestBucketFlag()
	mnt.rootDir = setup.TestDir()
	mnt.mntDir = setup.MntDir()
	mnt.logFilePath = setup.LogFile()
	mnt.testDirPath = path.Join(setup.MntDir(), testDirName)
}

func (s *BaseSuite) mountPrimaryMount(flags []string) {
	setup.SetMntDir(s.primaryMount.mntDir)
	setup.SetLogFile(s.primaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(s.T(), err, "Unable to mount primary: %v", err)
	setup.SetupTestDirectory(testDirName)
}

func (s *BaseSuite) unmountPrimaryMount() { setup.UnmountGCSFuse(s.primaryMount.mntDir) }

func (s *BaseSuite) mountSecondaryMount(flags []string) {
	setup.SetMntDir(s.secondaryMount.mntDir)
	setup.SetLogFile(s.secondaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(s.T(), err, "Unable to mount secondary: %v", err)
	s.secondaryMount.testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *BaseSuite) unmountSecondaryMount() { setup.UnmountGCSFuse(s.secondaryMount.mntDir) }

func (s *BaseSuite) createUnfinalizedObject() {
	s.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	s.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, s.T(), storageClient, path.Join(testDirName, s.fileName), s.fileContent)
}

func (s *BaseSuite) deleteUnfinalizedObject() {
	if s.fileName != "" {
		err := os.Remove(path.Join(s.primaryMount.testDirPath, s.fileName))
		require.NoError(s.T(), err)
		s.fileName = ""
	}
}

func (s *BaseSuite) getAppendPath() string {
	if s.cfg.isDualMount {
		return s.secondaryMount.testDirPath
	}
	return s.primaryMount.testDirPath
}

func (s *BaseSuite) appendToFile(file *os.File, appendContent string) {
	s.T().Helper()
	n, err := file.WriteString(appendContent)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), len(appendContent), n)
	s.fileContent += appendContent
	if s.cfg.isDualMount {
		operations.SyncFile(file, s.T())
	}
}

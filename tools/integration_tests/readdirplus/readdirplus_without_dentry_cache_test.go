// Copyright 2024 Google LLC
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

package readdirplus

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type ReaddirplusWithoutDentryCacheTest struct {
	suite.Suite
	flags [][]string
}

func (s *ReaddirplusWithoutDentryCacheTest) SetupSuite() {
	s.flags = setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, s.T().Name())
}

func (s *ReaddirplusWithoutDentryCacheTest) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.flags[0])
}

func (s *ReaddirplusWithoutDentryCacheTest) TearDownTest() {
	setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())
}

func (s *ReaddirplusWithoutDentryCacheTest) TestReaddirplusWithoutDentryCache() {
	// Create a directory with a few files.
	dirPath := path.Join(testEnv.testDirPath, targetDirName)
	err := os.Mkdir(dirPath, 0755)
	s.Require().NoError(err)

	file1Path := path.Join(dirPath, "file1")
	operations.CreateFile(file1Path, 0644, s.T())
	err = operations.WriteFile(file1Path, "content1")
	s.Require().NoError(err)

	file2Path := path.Join(dirPath, "file2")
	operations.CreateFile(file2Path, 0644, s.T())
	err = operations.WriteFile(file2Path, "content2")
	s.Require().NoError(err)

	startTime := time.Now()
	// ls the directory. This should call ReadDirPlus.
	_, err = os.ReadDir(dirPath)
	s.Require().NoError(err)
	endTime := time.Now()

	// Validate that ReadDirPlus and LookUpInode are called.
	validateLogsForReaddirplus(s.T(), setup.LogFile(), false, startTime, endTime)
}

func TestReaddirplusWithoutDentryCacheTest(t *testing.T) {
	suite.Run(t, new(ReaddirplusWithoutDentryCacheTest))
}

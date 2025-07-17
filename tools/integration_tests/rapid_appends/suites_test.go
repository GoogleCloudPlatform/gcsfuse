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

package rapid_appends

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Struct to store the details of a mount point
type mountPoint struct {
	rootDir     string // Root directory of the test folder, which contains mnt and gcsfuse.log.
	mntDir      string // Directory where the GCS bucket is mounted. This is 'mnt' inside rootDir.
	testDirPath string // Path to the 'RapidAppendsTest' directory inside mntDir.
	logFilePath string // Path to the GCSFuse log file. This is gcsfuse.log inside rootDir.
}

const (
	// TODO(b/432179045): `--write-global-max-blocks=-1` is needed right now because of a bug in global semaphore release.
	// Remove this flag once bug is fixed.
	infiniteWriteGlobalMaxBlocks = "--write-global-max-blocks=-1"
)

var (
	secondaryMountFlags = []string{writeRapidAppendsEnableFlag, infiniteWriteGlobalMaxBlocks}
)

// //////////////////////////////////////////////////////////////////////
// Suite Setup and Teardown
// //////////////////////////////////////////////////////////////////////

// RapidAppendsSuite is the base suite for rapid appends tests.
type RapidAppendsSuite struct {
	suite.Suite
	primaryMount            mountPoint
	fileName                string
	fileContent             string
	isSyncNeededAfterAppend bool
	appendMountPath         string
}

func (t *RapidAppendsSuite) SetupSuite() {
	// Set up test directory for primary mount.
	setup.SetUpTestDirForTestBucketFlag()
	t.primaryMount.rootDir = setup.TestDir()
	t.primaryMount.mntDir = setup.MntDir()
	t.primaryMount.logFilePath = setup.LogFile()
	t.primaryMount.testDirPath = path.Join(setup.MntDir(), testDirName)
}

func (t *RapidAppendsSuite) TearDownSuite() {
	if t.T().Failed() {
		setup.SetLogFile(t.primaryMount.logFilePath)
		log.Printf("Saving primary mount log file %q ...", t.primaryMount.logFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *RapidAppendsSuite) deleteUnfinalizedObject() {
	if t.fileName != "" {
		err := os.Remove(path.Join(t.primaryMount.testDirPath, t.fileName))
		require.NoError(t.T(), err)
		t.fileName = ""
	}
}

func (t *RapidAppendsSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *RapidAppendsSuite) mountPrimaryMount(flags []string) {
	// Create primary mountpoint.
	setup.SetMntDir(t.primaryMount.mntDir)
	setup.SetLogFile(t.primaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount gcsfuse with flags %v: %v", flags, err)
	setup.SetupTestDirectory(testDirName)
}

func (t *RapidAppendsSuite) unmountPrimaryMount() {
	setup.UnmountGCSFuse(t.primaryMount.mntDir)
}

// SingleMountRapidAppendsSuite is the suite for rapid appends tests with a single mount,
// i.e. primary mount for both reading and writing/appending.
type SingleMountRapidAppendsSuite struct {
	RapidAppendsSuite
}

func (t *SingleMountRapidAppendsSuite) SetupSuite() {
	t.RapidAppendsSuite.SetupSuite()

	t.appendMountPath = t.primaryMount.testDirPath
	//  fsync is not needed after append if append is from the same mount point as the read mount point.
	t.isSyncNeededAfterAppend = false
}

func (t *SingleMountRapidAppendsSuite) TearDownSuite() {
	t.RapidAppendsSuite.TearDownSuite()
}

// DualMountRapidAppendsSuite is the suite for rapid appends tests with two mounts,
// primary for reading and secondary for writing/appending.
type DualMountRapidAppendsSuite struct {
	RapidAppendsSuite
	secondaryMount mountPoint
}

func (t *DualMountRapidAppendsSuite) SetupSuite() {
	t.RapidAppendsSuite.SetupSuite()

	// Create secondary mount.
	setup.SetUpTestDirForTestBucketFlag()
	t.secondaryMount.rootDir = setup.TestDir()
	t.secondaryMount.mntDir = setup.MntDir()
	t.secondaryMount.logFilePath = setup.LogFile()
	setup.MountGCSFuseWithGivenMountFunc(secondaryMountFlags, mountFunc)
	t.secondaryMount.testDirPath = setup.SetupTestDirectory(testDirName)

	t.appendMountPath = t.secondaryMount.testDirPath
	// fsync is needed after append if append is from a different mount point (secondary mount) from the read mount point (primary mount).
	t.isSyncNeededAfterAppend = true
}

func (t *DualMountRapidAppendsSuite) TearDownSuite() {
	// Clean up of secondary mount.
	setup.UnmountGCSFuse(t.secondaryMount.mntDir)
	if t.T().Failed() {
		setup.SetLogFile(t.secondaryMount.logFilePath)
		log.Printf("Saving secondary mount log file %q ...", t.secondaryMount.logFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
	t.RapidAppendsSuite.TearDownSuite()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// appendToFile appends the given "appendContent" to the given file.
func (t *RapidAppendsSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileContent += appendContent
	if t.isSyncNeededAfterAppend {
		operations.SyncFile(file, t.T())
	}
}

func getNewEmptyCacheDir(rootDir string) string {
	cacheDirPath, err := os.MkdirTemp(rootDir, "cache_dir_*")
	if err != nil {
		log.Fatalf("Failed to create temporary directory for cache dir for tests: %v", err)
	}
	return cacheDirPath
}

////////////////////////////////////////////////////////////////////////
// Test Functions (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSingleMountSuite(t *testing.T) {
	suite.Run(t, new(SingleMountRapidAppendsSuite))
}

func TestRapidAppendsDualMountSuite(t *testing.T) {
	suite.Run(t, new(DualMountRapidAppendsSuite))
}

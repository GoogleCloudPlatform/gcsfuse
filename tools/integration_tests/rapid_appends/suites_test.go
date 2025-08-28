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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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

var (
	// TODO(b/432179045): `--write-global-max-blocks=-1` is needed right now because of a bug in global semaphore release.
	secondaryMountFlags = []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"}
)

// //////////////////////////////////////////////////////////////////////
// Suite Setup and Teardown
// //////////////////////////////////////////////////////////////////////

// CommonAppendsSuite is the base suite for rapid appends tests.
// Adding any tests (.e. Test*() member functions) to this struct will add it
// for all structs that encapsulate it, e.g. SingleMountAppendsSuite.
type CommonAppendsSuite struct {
	suite.Suite
	primaryMount            mountPoint
	fileName                string
	fileContent             string
	isSyncNeededAfterAppend bool
	appendMountPath         string
}

func (t *CommonAppendsSuite) SetupSuite() {
	t.primaryMount.setupTestDir()
}

func (t *CommonAppendsSuite) TearDownSuite() {
	t.tearDownMount(&t.primaryMount)
}

// SingleMountAppendsSuite is the suite for rapid appends tests with a single mount,
// i.e. primary mount.
// Adding any tests (.e. Test*() member functions) to this struct will run
// only for single-mount.
type SingleMountAppendsSuite struct {
	CommonAppendsSuite
}

func (t *SingleMountAppendsSuite) SetupSuite() {
	t.CommonAppendsSuite.SetupSuite()
	// Set up appends to work through secondary mount.
	t.appendMountPath = t.primaryMount.testDirPath
	//  fsync is not needed after append if append is from the same mount point as the read mount point.
	t.isSyncNeededAfterAppend = false
}

func (t *SingleMountAppendsSuite) TearDownSuite() {
	t.CommonAppendsSuite.TearDownSuite()
}

// DualMountAppendsSuite is the suite for rapid appends tests with two mounts,
// primary for reading and secondary for writing/appending.
// Adding any tests (.e. Test*() member functions) to this struct will run
// only for dual-mount.
type DualMountAppendsSuite struct {
	CommonAppendsSuite
	secondaryMount mountPoint
}

func (t *DualMountAppendsSuite) SetupSuite() {
	t.CommonAppendsSuite.SetupSuite()
	t.secondaryMount.setupTestDir()
	t.mountSecondaryMount(secondaryMountFlags)

	t.appendMountPath = t.secondaryMount.testDirPath
	// fsync is needed after append if append is from a different mount point (secondary mount) from the read mount point (primary mount).
	t.isSyncNeededAfterAppend = true
}

func (t *DualMountAppendsSuite) TearDownSuite() {
	t.unmountSecondaryMount()
	t.tearDownMount(&t.secondaryMount)
	t.CommonAppendsSuite.TearDownSuite()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (mnt *mountPoint) setupTestDir() {
	setup.SetUpTestDirForTestBucketFlag()
	mnt.rootDir = setup.TestDir()
	mnt.mntDir = setup.MntDir()
	mnt.logFilePath = setup.LogFile()
	mnt.testDirPath = path.Join(setup.MntDir(), testDirName)
}

func (t *CommonAppendsSuite) tearDownMount(mnt *mountPoint) {
	if t.T().Failed() {
		setup.SetLogFile(mnt.logFilePath)
		log.Printf("Saving mnt at %q mount log file %q ...", t.primaryMount.mntDir, t.primaryMount.logFilePath)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *CommonAppendsSuite) deleteUnfinalizedObject() {
	if t.fileName != "" {
		err := os.Remove(path.Join(t.primaryMount.testDirPath, t.fileName))
		require.NoError(t.T(), err)
		t.fileName = ""
	}
}

func (t *CommonAppendsSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
	time.Sleep(time.Minute) // Wait for a minute so that further stat() calls return correct size.
}

func (t *CommonAppendsSuite) mountPrimaryMount(flags []string) {
	// Create primary mountpoint.
	setup.SetMntDir(t.primaryMount.mntDir)
	setup.SetLogFile(t.primaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount gcsfuse with flags %v: %v", flags, err)
	setup.SetupTestDirectory(testDirName)
}

func (t *CommonAppendsSuite) unmountPrimaryMount() {
	setup.UnmountGCSFuse(t.primaryMount.mntDir)
}

func (t *DualMountAppendsSuite) mountSecondaryMount(flags []string) {
	// Create secondary mountpoint.
	setup.SetMntDir(t.secondaryMount.mntDir)
	setup.SetLogFile(t.secondaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount gcsfuse with flags %v: %v", flags, err)
	//setup.SetupTestDirectory(testDirName)
	t.secondaryMount.testDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *DualMountAppendsSuite) unmountSecondaryMount() {
	setup.UnmountGCSFuse(t.secondaryMount.mntDir)
}

// appendToFile appends the given "appendContent" to the given file.
func (t *CommonAppendsSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(appendContent), n)
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

func TestSingleMountAppendsSuite(t *testing.T) {
	suite.Run(t, new(SingleMountAppendsSuite))
}

func TestDualMountAppendsSuite(t *testing.T) {
	suite.Run(t, new(DualMountAppendsSuite))
}

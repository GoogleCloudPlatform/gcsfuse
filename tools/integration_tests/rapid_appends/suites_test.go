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
	"fmt"
	"log"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Suite Definitions
// //////////////////////////////////////////////////////////////////////

// Struct to store the details of a mount point
type mountPoint struct {
	rootDir     string // Root directory of the test folder, which contains mnt and gcsfuse.log.
	mntDir      string // Directory where the GCS bucket is mounted. This is 'mnt' inside rootDir.
	testDirPath string // Path to the 'RapidAppendsTest' directory inside mntDir.
	logFilePath string // Path to the GCSFuse log file. This is gcsfuse.log inside rootDir.
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

func (t *BaseSuite) SetupTest() {
	t.primaryMount.setupTestDir()

	// Create a mutable copy of flags and add file cache flags if configured.
	primaryFlags := make([]string, len(t.cfg.primaryMountFlags))
	copy(primaryFlags, t.cfg.primaryMountFlags)
	if t.cfg.fileCache {
		cacheDir := getNewEmptyCacheDir(t.primaryMount.rootDir)
		primaryFlags = append(primaryFlags, "--file-cache-max-size-mb=-1", "--cache-dir="+cacheDir)
	}
	if t.cfg.metadataCacheOnRead {
		primaryFlags = append(primaryFlags, fmt.Sprintf("--metadata-cache-ttl-secs=%v", metadataCacheTTLSecs))
	} else {
		primaryFlags = append(primaryFlags, "--metadata-cache-ttl-secs=0")
	}
	t.mountPrimaryMount(primaryFlags)

	if t.cfg.isDualMount {
		t.secondaryMount.setupTestDir()
		t.mountSecondaryMount(t.cfg.secondaryMountFlags)
	}
}

func (t *BaseSuite) TearDownTest() {
	t.unmountPrimaryMount()
	if t.cfg.isDualMount {
		t.unmountSecondaryMount()
	}
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

func (t *BaseSuite) mountPrimaryMount(flags []string) {
	setup.SetMntDir(t.primaryMount.mntDir)
	setup.SetLogFile(t.primaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount primary: %v", err)
	setup.SetupTestDirectory(testDirName)
	log.Printf("Running tests with primary mount flags %v", flags)
}

func (t *BaseSuite) unmountPrimaryMount() { setup.UnmountGCSFuse(t.primaryMount.mntDir) }

func (t *BaseSuite) mountSecondaryMount(flags []string) {
	setup.SetMntDir(t.secondaryMount.mntDir)
	setup.SetLogFile(t.secondaryMount.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount secondary: %v", err)
	t.secondaryMount.testDirPath = setup.SetupTestDirectory(testDirName)
	log.Printf("Running tests with secondary mount flags %v", flags)
}

func (t *BaseSuite) unmountSecondaryMount() { setup.UnmountGCSFuse(t.secondaryMount.mntDir) }

func (t *BaseSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *BaseSuite) deleteUnfinalizedObject() {
	if t.fileName != "" {
		err := os.Remove(path.Join(t.primaryMount.testDirPath, t.fileName))
		require.NoError(t.T(), err)
		t.fileName = ""
	}
}

func (t *BaseSuite) getAppendPath() string {
	if t.cfg.isDualMount {
		return t.secondaryMount.testDirPath
	}
	return t.primaryMount.testDirPath
}

func (t *BaseSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(appendContent), n)
	t.fileContent += appendContent
	if t.cfg.isDualMount {
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

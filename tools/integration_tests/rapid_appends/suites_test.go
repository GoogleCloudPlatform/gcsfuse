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

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Suite Definitions
////////////////////////////////////////////////////////////////////////

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

// SingleMountReadsTestSuite groups all single-mount tests related to reading after appends.
type SingleMountReadsTestSuite struct{ BaseSuite }

// DualMountReadsTestSuite groups all dual-mount tests related to reading after appends.
type DualMountReadsTestSuite struct{ BaseSuite }

// SingleMountAppendsTestSuite groups general single-mount tests for append behavior.
type SingleMountAppendsTestSuite struct{ BaseSuite }

// DualMountAppendsTestSuite groups general dual-mount tests for append behavior.
type DualMountAppendsTestSuite struct{ BaseSuite }

////////////////////////////////////////////////////////////////////////
// Common Suite Logic
////////////////////////////////////////////////////////////////////////

func (t *BaseSuite) SetupTest() {
	t.primaryMount.setupTestDir()

	// Create a mutable copy of flags.
	primaryFlags := make([]string, len(t.cfg.primaryMountFlags))
	copy(primaryFlags, t.cfg.primaryMountFlags)
	// Add file cache flags if configured.
	if t.cfg.fileCacheEnabled {
		cacheDir := getNewEmptyCacheDir(t.primaryMount.rootDir)
		primaryFlags = append(primaryFlags, "--file-cache-max-size-mb=-1", "--cache-dir="+cacheDir)
	}
	if t.cfg.metadataCacheEnabled {
		primaryFlags = append(primaryFlags, fmt.Sprintf("--metadata-cache-ttl-secs=%v", metadataCacheTTLSecs))
	} else {
		primaryFlags = append(primaryFlags, "--metadata-cache-ttl-secs=0")
	}
	t.mountGcsfuse(t.primaryMount, "primary", primaryFlags)

	if t.cfg.isDualMount {
		t.secondaryMount.setupTestDir()
		t.mountGcsfuse(t.secondaryMount, "secondary", t.cfg.secondaryMountFlags)
	}
}

func (t *BaseSuite) TearDownTest() {
	if t.T().Failed() {
		// Save logs for both mounts on failure to aid debugging.
		setup.SaveLogFileAsArtifact(t.primaryMount.logFilePath, "gcsfuse-primary-log-"+t.T().Name())
		if t.cfg.isDualMount {
			setup.SaveLogFileAsArtifact(t.secondaryMount.logFilePath, "gcsfuse-secondary-log-"+t.T().Name())
		}
	}

	// Unmount and clean up the root directories to remove all test artifacts.
	t.unmountAndCleanupMount(t.primaryMount, "primary")
	if t.cfg.isDualMount {
		t.unmountAndCleanupMount(t.secondaryMount, "secondary")
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

func (t *BaseSuite) mountGcsfuse(mnt mountPoint, mountType string, flags []string) {
	setup.SetMntDir(mnt.mntDir)
	setup.SetLogFile(mnt.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount %s: %v", mountType, err)
	mnt.testDirPath = setup.SetupTestDirectory(testDirName)
	log.Printf("Running tests with %s mount flags %v", mountType, flags)
}

func (t *BaseSuite) unmountAndCleanupMount(m mountPoint, name string) {
	setup.UnmountGCSFuse(m.mntDir)
	err := os.RemoveAll(m.rootDir)
	require.NoError(t.T(), err, "Failed to clean up %v mount root directory", name)
	// Cleaning up the intermediate generated test files.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, m.testDirPath)
}

func (t *BaseSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
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

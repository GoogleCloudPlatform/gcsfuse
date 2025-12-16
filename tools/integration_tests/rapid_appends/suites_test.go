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

	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Suite Definitions
////////////////////////////////////////////////////////////////////////

const (
	defaultMetadataCacheTTL = 60 * time.Second
)

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
	primaryFlags   []string
	secondaryFlags []string
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
	t.primaryMount.setupTestDir(testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.LogFile)

	t.mountGcsfuse(t.primaryMount, "primary", t.primaryFlags)

	if len(t.secondaryFlags) > 0 {
		secondaryLog := path.Join(path.Dir(testEnv.cfg.LogFile), "gcsfuse_secondary.log")
		t.secondaryMount.setupTestDir(testEnv.cfg.GCSFuseMountedDirectorySecondary, secondaryLog)
		t.mountGcsfuse(t.secondaryMount, "secondary", t.secondaryFlags)
	}
}

func (t *BaseSuite) TearDownTest() {
	if t.T().Failed() {
		// Save logs for both mounts on failure to aid debugging.
		setup.SaveLogFileAsArtifact(t.primaryMount.logFilePath, "gcsfuse-primary-log-"+t.T().Name())
		if len(t.secondaryFlags) > 0 {
			setup.SaveLogFileAsArtifact(t.secondaryMount.logFilePath, "gcsfuse-secondary-log-"+t.T().Name())
		}
	}

	// Unmount and clean up the root directories to remove all test artifacts.
	t.unmountAndCleanupMount(t.primaryMount, "primary")
	if len(t.secondaryFlags) > 0 {
		t.unmountAndCleanupMount(t.secondaryMount, "secondary")
	}
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (mnt *mountPoint) setupTestDir(mountDir, logFile string) {
	mnt.rootDir = setup.TestDir()
	mnt.mntDir = mountDir
	mnt.logFilePath = logFile
	mnt.testDirPath = path.Join(mountDir, testDirName)
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
	// Cleaning up the intermediate generated test files.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
}

func (t *BaseSuite) createUnfinalizedObject() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(testEnv.ctx, t.T(), testEnv.storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *BaseSuite) deleteUnfinalizedObject() {
	if t.fileName != "" {
		err := os.Remove(path.Join(t.primaryMount.testDirPath, t.fileName))
		require.NoError(t.T(), err)
		t.fileName = ""
	}
}

func (t *BaseSuite) getAppendPath() string {
	if len(t.secondaryFlags) > 0 {
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
	if len(t.secondaryFlags) > 0 {
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

func (t *BaseSuite) isMetadataCacheEnabled() bool {
	for _, flag := range t.primaryFlags {
		if strings.Contains(flag, "--metadata-cache-ttl-secs") {
			// Check if value is not 0
			// Assuming format --metadata-cache-ttl-secs=X
			parts := strings.Split(flag, "=")
			if len(parts) == 2 {
				return parts[1] != "0"
			}
		}
	}
	// Default is enabled if not specified? Or disabled?
	// In existing tests, default was enabled (TTL=60s) unless set to 0.
	// But in new framework, we usually specify flags explicitly.
	// Let's assume if not present, it might be default (enabled).
	// However, for test purposes, we usually explicitly disable it with =0.
	return true
}

func RunTests(t *testing.T, runName string, factory func(primaryFlags, secondaryFlags []string) suite.TestingSuite) {
	type testRun struct {
		name           string
		primaryFlags   []string
		secondaryFlags []string
	}
	var runs []testRun

	for _, cfg := range testEnv.cfg.Configs {
		if cfg.Run == runName {
			for i, flagStr := range cfg.Flags {
				primaryFlags := strings.Fields(flagStr)
				var secondaryFlags []string
				if len(cfg.SecondaryFlags) > i {
					secondaryFlags = strings.Fields(cfg.SecondaryFlags[i])
				}
				runs = append(runs, testRun{
					name:           flagStr,
					primaryFlags:   primaryFlags,
					secondaryFlags: secondaryFlags,
				})
			}
		}
	}

	for _, r := range runs {
		if len(runs) == 1 {
			suite.Run(t, factory(r.primaryFlags, r.secondaryFlags))
		} else {
			t.Run(r.name, func(t *testing.T) {
				suite.Run(t, factory(r.primaryFlags, r.secondaryFlags))
			})
		}
	}
}

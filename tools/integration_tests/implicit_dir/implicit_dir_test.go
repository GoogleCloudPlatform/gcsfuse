// Copyright 2023 Google LLC
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

// Provide tests when implicit directory present and mounted bucket with --implicit-dir flag.
package implicit_dir_test

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const ExplicitDirInImplicitDir = "explicitDirInImplicitDir"
const ExplicitDirInImplicitSubDir = "explicitDirInImplicitSubDir"
const PrefixFileInExplicitDirInImplicitDir = "fileInExplicitDirInImplicitDir"
const PrefixFileInExplicitDirInImplicitSubDir = "fileInExplicitDirInImplicitSubDir"
const NumberOfFilesInExplicitDirInImplicitSubDir = 1
const NumberOfFilesInExplicitDirInImplicitDir = 1
const DirForImplicitDirTests = "dirForImplicitDirTests"

// IMPORTANT: To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

var testEnv env

type implicitDirTestSuite struct {
	suite.Suite
}

func runImplicitDirSuite(t *testing.T, runSuiteFunc func()) {
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		runSuiteFunc()
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		log.Printf("Running %s with flags: %s", t.Name(), flags)
		err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(testEnv.cfg, flags)
		require.NoError(t, err, "Mount failed")

		runSuiteFunc()

		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.UnmountGCSFuseWithConfig(testEnv.cfg)
	}
}

func TestImplicitDirBase(t *testing.T) {
	runImplicitDirSuite(t, func() {
		suite.Run(t, new(implicitDirTestSuite))
		suite.Run(t, &implicitDirLocalFileTest{isRapidWritesEnabled: false})
	})
}

func setupTestDir(dirName string) string {
	dir := setup.SetupTestDirectory(DirForImplicitDirTests)
	dirPath := path.Join(dir, dirName)
	err := os.Mkdir(dirPath, setup.DirPermission_0755)
	if err != nil {
		log.Fatalf("Error while setting up directory %s for testing: %v", dirPath, err)
	}

	return dirPath
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ImplicitDir) == 0 {
		log.Fatal("No configuration found for ImplicitDir in config file.")
	}

	// 2. Create storage client before running tests.
	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.ImplicitDir[0])
	testEnv.cfg = &cfg.ImplicitDir[0]
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	successCode := m.Run()

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

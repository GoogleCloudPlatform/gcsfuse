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

// Provides integration tests for write large files sequentially and randomly.
package write_large_files

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

const (
	testDirName    = "WriteLargeFilesTests"
	onlyDirMounted = "OnlyDirMountWriteLargeFiles"
	TmpDir         = "/tmp"
	OneMiB         = 1024 * 1024
)

type env struct {
	testDirPath   string
	mountFunc     func(*test_suite.TestConfig, []string) error
	mountDir      string
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
	bucketType    string
	cfg           test_suite.TestConfig
}

var (
	testEnv env
)

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) error {
	err := setup.MayMountGCSFuseWithGivenMountWithConfigFunc(&testEnv.cfg, flags, testEnv.mountFunc)
	if err != nil {
		return err
	}
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.mountDir)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
	return nil
}

func mustMountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	if err := mountGCSFuseAndSetupTestDir(flags, ctx, storageClient); err != nil {
		panic(err)
	}
}

type WriteLargeFilesTestSuite struct {
	suite.Suite
	flags []string
}

func (s *WriteLargeFilesTestSuite) SetupSuite() {
	mustMountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *WriteLargeFilesTestSuite) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *WriteLargeFilesTestSuite) TearDownSuite() {
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.UnmountGCSFuse(testEnv.rootDir)
	}
}

func TestWriteLargeFiles(t *testing.T) {
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, new(WriteLargeFilesTestSuite))
		return
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		s := &WriteLargeFilesTestSuite{flags: flags}
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			suite.Run(t, s)
		})
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.WriteLargeFiles) == 0 {
		log.Fatal("No configuration found for WriteLargeFiles in config file.")
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.WriteLargeFiles[0])
	testEnv.cfg = cfg.WriteLargeFiles[0]
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucket(&testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(&testEnv.cfg, setup.TestDir())

	testEnv.mountDir, testEnv.rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	testEnv.mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		testEnv.mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		testEnv.mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		if err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, path.Join(setup.OnlyDirMounted(), testDirName)); err != nil {
			log.Printf("Error deleting object on GCS: %v", err)
		}
	}

	setup.SaveLogFileInCaseOfFailure(successCode)
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

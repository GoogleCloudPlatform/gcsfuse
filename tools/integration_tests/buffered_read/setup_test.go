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

package buffered_read

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName                         = "BufferedReadTest"
	testFileName                        = "foo"
	logFileNameForMountedDirectoryTests = "/tmp/gcsfuse_buffered_read_test_logs/log.json"
	// Global block size constant for tests
	blockSizeInBytes = int64(8 * util.MiB)
)

var (
	mountFunc func(*test_suite.TestConfig, []string) error
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
	bucketType    string
}

var testEnv env

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupForMountedDirectoryTests() {
	if setup.MountedDirectory() != "" {
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
	}
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.BufferedRead) == 0 {
		log.Println("No configuration found for buffered_read tests in config. Using flags instead.")
		cfg.BufferedRead = make([]test_suite.TestConfig, 1)
		cfg.BufferedRead[0].TestBucket = setup.TestBucket()
		cfg.BufferedRead[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.BufferedRead[0].LogFile = setup.LogFile()
		cfg.BufferedRead[0].Configs = make([]test_suite.ConfigItem, 3)

		cfg.BufferedRead[0].Configs[0].Flags = []string{
			"--enable-buffered-read --read-block-size-mb=8 --read-max-blocks-per-handle=20 --read-start-blocks-per-handle=1 --read-min-blocks-per-handle=2 --enable-kernel-reader=false",
			"--client-protocol=grpc --enable-buffered-read --read-block-size-mb=8 --read-max-blocks-per-handle=20 --read-start-blocks-per-handle=1 --read-min-blocks-per-handle=2 --enable-kernel-reader=false",
		}
		cfg.BufferedRead[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.BufferedRead[0].Configs[0].Run = "TestSequentialReadSuite"

		cfg.BufferedRead[0].Configs[1].Flags = []string{
			"--enable-buffered-read --read-block-size-mb=8 --read-min-blocks-per-handle=2 --read-global-max-blocks=1 --read-max-blocks-per-handle=10 --read-start-blocks-per-handle=2 --enable-kernel-reader=false",
			"--client-protocol=grpc --enable-buffered-read --read-block-size-mb=8 --read-min-blocks-per-handle=2 --read-global-max-blocks=1 --read-max-blocks-per-handle=10 --read-start-blocks-per-handle=2 --enable-kernel-reader=false",
		}
		cfg.BufferedRead[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.BufferedRead[0].Configs[1].Run = "TestInsufficientPoolCreationSuite"

		cfg.BufferedRead[0].Configs[2].Flags = []string{
			"--enable-buffered-read --read-block-size-mb=8 --read-max-blocks-per-handle=20 --read-start-blocks-per-handle=2 --read-min-blocks-per-handle=2 --enable-kernel-reader=false",
			"--client-protocol=grpc --enable-buffered-read --read-block-size-mb=8 --read-max-blocks-per-handle=20 --read-start-blocks-per-handle=2 --read-min-blocks-per-handle=2 --enable-kernel-reader=false",
		}
		cfg.BufferedRead[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.BufferedRead[0].Configs[2].Run = "TestRandomReadFallbackSuite"
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.BufferedRead[0])
	testEnv.cfg = &cfg.BufferedRead[0]

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile

	// Run the tests.
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
	os.Exit(successCode)
}

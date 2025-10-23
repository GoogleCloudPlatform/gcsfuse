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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName                         = "BufferedReadTest"
	testFileName                        = "foo"
	clientProtocolHTTP1                 = "http1"
	clientProtocolGRPC                  = "grpc"
	logFileNameForMountedDirectoryTests = "/tmp/gcsfuse_buffered_read_test_logs/log.json"
)

var (
	mountFunc     func([]string) error
	storageClient *storage.Client
	ctx           context.Context
	rootDir       string
)

type gcsfuseTestFlags struct {
	clientProtocol       string
	enableBufferedRead   bool
	blockSizeMB          int64
	maxBlocksPerHandle   int64
	startBlocksPerHandle int64
	minBlocksPerHandle   int64
	globalMaxBlocks      int64
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupForMountedDirectoryTests() {
	if setup.MountedDirectory() != "" {
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
	}
}

func createConfigFile(flags *gcsfuseTestFlags) string {
	mountConfig := make(map[string]any)
	readConfig := make(map[string]any)

	// Only add flags to the config if they have a non-zero/non-default value.
	// This prevents writing zero values that would override GCSFuse's built-in defaults.
	if flags.enableBufferedRead {
		readConfig["enable-buffered-read"] = flags.enableBufferedRead
	}
	if flags.blockSizeMB != 0 {
		readConfig["block-size-mb"] = flags.blockSizeMB
	}
	if flags.maxBlocksPerHandle != 0 {
		readConfig["max-blocks-per-handle"] = flags.maxBlocksPerHandle
	}
	if flags.startBlocksPerHandle != 0 {
		readConfig["start-blocks-per-handle"] = flags.startBlocksPerHandle
	}
	if flags.minBlocksPerHandle != 0 {
		readConfig["min-blocks-per-handle"] = flags.minBlocksPerHandle
	}
	if flags.globalMaxBlocks != 0 {
		readConfig["global-max-blocks"] = flags.globalMaxBlocks
	}
	if len(readConfig) > 0 {
		mountConfig["read"] = readConfig
	}
	if flags.clientProtocol != "" {
		mountConfig["gcs-connection"] = map[string]any{"client-protocol": flags.clientProtocol}
	}
	filePath := setup.YAMLConfigFile(mountConfig, "config.yaml")
	return filePath
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// A test bucket must be provided for all tests.
	if setup.TestBucket() == "" {
		log.Fatal("The --testbucket flag is required for this test.")
	}

	if setup.MountedDirectory() != "" {
		rootDir = setup.MountedDirectory()
		os.Exit(setup.RunTestsForMountedDirectory(setup.MountedDirectory(), m))
	}

	// Else run tests for testBucket.
	setup.SetUpTestDirForTestBucketFlag()
	rootDir = setup.MntDir()

	// Set up the static mounting function.
	mountFunc = func(flags []string) error {
		config := &test_suite.TestConfig{
			TestBucket:              setup.TestBucket(),
			GKEMountedDirectory:     setup.MountedDirectory(),
			GCSFuseMountedDirectory: setup.MntDir(),
			LogFile:                 setup.LogFile(),
		}
		return static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
	}

	// Run the tests.
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName              = "BufferedReadTest"
	onlyDirMounted           = "OnlyDirMountBufferedRead"
	testFileName             = "foo"
	configFileName           = "config"
	logFileNameForMountedDir = "/tmp/gcsfuse_buffered_read_test_logs/log.json"
	http1ClientProtocol      = "http1"
	grpcClientProtocol       = "grpc"
)

var (
	testDirPath   string
	mountFunc     func([]string) error
	mountDir      string
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
)

type gcsfuseTestFlags struct {
	cliFlags             []string
	clientProtocol       string
	enableBufferedRead   bool
	blockSizeMB          int64
	maxBlocksPerHandle   int64
	startBlocksPerHandle int64
	fileName             string
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupForMountedDirectoryTests() {
	if setup.MountedDirectory() != "" {
		mountDir = setup.MountedDirectory()
		setup.SetLogFile(logFileNameForMountedDir)
	}
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func createConfigFile(flags *gcsfuseTestFlags) string {
	mountConfig := map[string]interface{}{
		"read": map[string]interface{}{
			"enable-buffered-read":    flags.enableBufferedRead,
			"block-size-mb":           flags.blockSizeMB,
			"max-blocks-per-handle":   flags.maxBlocksPerHandle,
			"start-blocks-per-handle": flags.startBlocksPerHandle,
		},
		"gcs-connection": map[string]interface{}{
			"client-protocol": flags.clientProtocol,
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, flags.fileName)
	return filePath
}

func appendClientProtocolConfigToFlagSet(testFlagSet []gcsfuseTestFlags) (testFlagsWithHttpAndGrpc []gcsfuseTestFlags) {
	for _, testFlags := range testFlagSet {
		testFlagsWithHttp := testFlags
		testFlagsWithHttp.clientProtocol = http1ClientProtocol
		testFlagsWithHttpAndGrpc = append(testFlagsWithHttpAndGrpc, testFlagsWithHttp)

		testFlagsWithGrpc := testFlags
		testFlagsWithGrpc.clientProtocol = grpcClientProtocol
		testFlagsWithHttpAndGrpc = append(testFlagsWithHttpAndGrpc, testFlagsWithGrpc)
	}
	return
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

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" {
		os.Exit(setup.RunTestsForMountedDirectory(setup.MountedDirectory(), m))
	}

	// Else run tests for testBucket.
	setup.SetUpTestDirForTestBucket(setup.TestBucket())

	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = func(flags []string) error {
		config := &test_suite.TestConfig{
			TestBucket:       setup.TestBucket(),
			MountedDirectory: setup.MountedDirectory(),
			LogFile:          setup.LogFile(),
		}
		return static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
	}
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

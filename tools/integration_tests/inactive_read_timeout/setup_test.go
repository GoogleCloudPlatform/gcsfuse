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

package inactive_read_timeout

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName                         = "inactiveReadTimeoutTest"
	onlyDirMounted                      = "onlyDirInactiveReadTimeout"
	fileSize                            = 128 * 1024 // 128 KiB
	chunkSizeToRead                     = 64 * 1024  // 64 KiB
	testFileName                        = "foo"
	configFileName                      = "config_inactive_read.yaml"
	defaultInactiveReadTimeoutInSeconds = 1 // A short timeout for testing
	logFileNameForMountedDirectoryTests = "/tmp/gcsfuse_inactive_read_timeout_test_logs/log.json"
	http1ClientProtocol                 = "http1"
	grpcClientProtocol                  = "grpc"
)

var (
	testDirPath   string // /tmp/**/mnt/inactiveReadTimeout
	mountFunc     func([]string) error
	mountDir      string
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
)

type gcsfuseTestFlags struct {
	cliFlags            []string
	inactiveReadTimeout time.Duration
	fileName            string
	clientProtocol      string
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client, testDirName string) {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func createConfigFile(flags *gcsfuseTestFlags) string {
	// Create a temporary directory for the cache if needed by other flags, though not directly used by inactive_read_timeout itself.
	// This makes the config structure similar to other tests.
	cacheDir := path.Join(setup.TestDir(), "temp_cache_inactive_read")
	operations.RemoveDir(cacheDir) // Clean up before use

	mountConfig := map[string]interface{}{
		"read": map[string]interface{}{
			"inactive-stream-timeout": flags.inactiveReadTimeout.String(),
		},
		"gcs-connection": map[string]interface{}{
			"client-protocol": flags.clientProtocol,
		},
		// Add other necessary cache/logging configs if your tests depend on them
		"logging": map[string]interface{}{
			"file-path": setup.LogFile(),
			"format":    "json", // Ensure JSON logs for easier parsing
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, flags.fileName)
	return filePath
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Create common storage client to be used in test.
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
		mountDir = setup.MountedDirectory()
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
		// Run tests for mounted directory if the flag is set.
		os.Exit(m.Run())
	}

	// Else run tests for testBucket.
	setup.SetUpTestDirForTestBucketFlag()

	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	disable := true
	if successCode == 0 && !disable {
		log.Println("Running dynamic mounting tests...")
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 && !disable {
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

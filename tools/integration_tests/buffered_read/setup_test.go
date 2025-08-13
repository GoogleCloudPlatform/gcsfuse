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
	testDirName         = "BufferedReadTest"
	testFileName        = "foo"
	clientProtocolHTTP1 = "http1"
	clientProtocolGRPC  = "grpc"
)

var (
	mountFunc     func([]string) error
	storageClient *storage.Client
	ctx           context.Context
)

type gcsfuseTestFlags struct {
	clientProtocol       string
	enableBufferedRead   bool
	blockSizeMB          int64
	maxBlocksPerHandle   int64
	startBlocksPerHandle int64
	minBlocksPerHandle   int64
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createConfigFile(flags *gcsfuseTestFlags) string {
	mountConfig := map[string]interface{}{
		"read": map[string]interface{}{
			"enable-buffered-read":    flags.enableBufferedRead,
			"block-size-mb":           flags.blockSizeMB,
			"max-blocks-per-handle":   flags.maxBlocksPerHandle,
			"start-blocks-per-handle": flags.startBlocksPerHandle,
			"min-blocks-per-handle":   flags.minBlocksPerHandle,
		},
		"gcs-connection": map[string]interface{}{
			"client-protocol": flags.clientProtocol,
		},
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

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" {
		os.Exit(setup.RunTestsForMountedDirectory(setup.MountedDirectory(), m))
	}

	// Else run tests for testBucket.
	setup.SetUpTestDirForTestBucket(setup.TestBucket())

	// Set up the static mounting function.
	mountFunc = func(flags []string) error {
		config := &test_suite.TestConfig{
			TestBucket:       setup.TestBucket(),
			MountedDirectory: setup.MountedDirectory(),
			LogFile:          setup.LogFile(),
		}
		return static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
	}

	// Run the tests.
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

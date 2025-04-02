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

package stale_handle

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

var (
	mountFunc     func([]string, string, string) error
	storageClient *storage.Client
	ctx           context.Context
	flagsSetMap   map[string][]string
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create common storage client to be used in test.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	// Define flag set to run the tests.
	flagsSetMap = map[string][]string{

		"Default":             {"--metadata-cache-ttl-secs=0", "--precondition-errors=true"},
		"DefaultGrpc":         {"--metadata-cache-ttl-secs=0", "--precondition-errors=true", "--client-protocol=grpc"},
		"StreamingWrites":     {"--metadata-cache-ttl-secs=0", "--precondition-errors=true", "--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=1"},
		"StreamingWritesGrpc": {"--metadata-cache-ttl-secs=0", "--precondition-errors=true", "--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=1", "--client-protocol=grpc"},
	}

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingMntDirAndLogFile
	successCode := m.Run()
	if err := setup.UnMountAllMountsWithPrefix(setup.TestDir()); err != nil {
		successCode = 1
	}
	os.Exit(successCode)
}

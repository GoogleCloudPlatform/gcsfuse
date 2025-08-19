// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package refac_rapid_appends

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName    = "RapidAppendsTest"
	fileNamePrefix = "rapid-append-file-"
	// Minimum content size to trigger block upload; calculated as (2*blocksize+1) MiB.
	contentSizeForBW = 3
	// Block size for buffered writes is 1MiB.
	blockSize            = operations.OneMiB
	metadataCacheTTLSecs = 10
)

var (
	storageClient *storage.Client
	ctx           context.Context
)

// //////////////////////////////////////////////////////////////////////
// TestMain
// //////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if !setup.IsZonalBucketRun() {
		log.Fatalf("This package must be run with a Zonal Bucket.")
	}

	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		log.Fatalf("This package does not support the --mountedDirectory flag.")
	}

	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		if err := closeStorageClient(); err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// The mount function is set implicitly by the test suites now.
	log.Println("Running static mounting tests for rapid appends...")
	successCode := m.Run()

	// Clean up the test directory on GCS after all tests have run.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

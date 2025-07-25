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
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName    = "RapidAppendsTest"
	fileNamePrefix = "rapid-append-file-"
	// Minimum content size to write in order to trigger block upload while writing ; calculated as (2*blocksize+1) mb.
	contentSizeForBW              = 3
	// Block size for buffered writes is set to 1MiB.
	blockSize                     = operations.OneMiB
)

var (
	// Mount function to be used for the mounting.
	mountFunc func([]string) error

	// Clients to create the object in GCS.
	storageClient *storage.Client
	ctx           context.Context
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	if !setup.IsZonalBucketRun() {
		log.Fatalf("This package is not supposed to be run with Regional Buckets.")
	}
	// TODO(b/431926259): Add support for mountedDir tests as this
	// package has multi-mount scenario tests and currently we only
	// pass single mountedDir to test package.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		log.Fatalf("This package doesn't support --mountedDirectory option currently.")
	}
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

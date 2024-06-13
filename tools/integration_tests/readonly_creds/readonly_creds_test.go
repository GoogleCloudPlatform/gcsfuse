// Copyright 2024 Google Inc. All Rights Reserved.
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

package readonly_creds

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName           = "ReadOnlyCredsTest"
	testFileName          = "fileName.txt"
	content               = "write content."
	permissionDeniedError = "permission denied"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" {
		log.Println("These tests will not run with mounted directory..")
		return
	}

	// Create test directory.
	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*15)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v", err)
		}
	}()
	client.SetupTestDirectory(ctx, storageClient, testDirName)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	// Test for viewer permission on test bucket.
	flags := [][]string{{"--implicit-dirs=true"}, {"--implicit-dirs=false"}}
	successCode := creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(flags, "objectViewer", m)

	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

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

// Provides integration tests for file and directory operations.

package local_file

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// set the test dir to local file test
	testDirName = testDirLocalFileTest

	// Create storage client before running tests.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()
	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as local_file tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Set up flags to run tests on local file test suite.
	// Not setting config file explicitly with 'create-empty-file: false' as it is default.
	flagsSet := [][]string{
		{"--implicit-dirs=true", "--rename-dir-limit=3"},
		{"--implicit-dirs=false", "--rename-dir-limit=3"}}

	if hnsFlagSet, err := setup.AddHNSFlagForHierarchicalBucket(ctx, storageClient); err == nil {
		flagsSet = append(flagsSet, hnsFlagSet)
	}

	if !testing.Short() {
		setup.AppendFlagsToAllFlagsInTheFlagsSet(&flagsSet, "--client-protocol=grpc")
	}

	successCode := static_mounting.RunTests(flagsSet, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flagsSet, onlyDirMounted, m)
	}

	// Dynamic mounting tests create a bucket and perform tests on that bucket,
	// which is not a hierarchical bucket. So we are not running those tests with
	// hierarchical bucket.
	if successCode == 0 && !setup.IsHierarchicalBucket(ctx, storageClient) {
		successCode = dynamic_mounting.RunTests(ctx, storageClient, flagsSet, m)
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

func TestLocalFileTestSuite(t *testing.T) {
	s := new(localFileTestSuite)
	s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
	suite.Run(t, s)
}

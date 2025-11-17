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

// Provides integration tests when --o=ro flag is set.
package readonly_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const TestDirForReadOnlyTest = "testDirForReadOnlyTest"
const DirectoryNameInTestBucket = "Test"         //  testBucket/testDirForReadOnlyTest/Test
const FileNameInTestBucket = "Test1.txt"         //  testBucket/testDirForReadOnlyTest/Test1.txt
const SubDirectoryNameInTestBucket = "b"         //  testBucket/testDirForReadOnlyTest/Test/b
const FileNameInDirectoryTestBucket = "a.txt"    //  testBucket/testDirForReadOnlyTest/Test/a.txt
const FileNameInSubDirectoryTestBucket = "b.txt" //  testBucket/testDirForReadOnlyTest/Test/b/b.txt
const NumberOfObjectsInTestBucket = 2
const NumberOfObjectsInDirectoryTestBucket = 2
const NumberOfObjectsInSubDirectoryTestBucket = 1
const FileNotExist = "notExist.txt"
const DirNotExist = "notExist"
const ContentInFileInTestBucket = "This is from file Test1\n"
const ContentInFileInDirectoryTestBucket = "This is from directory Test file a\n"
const ContentInFileInSubDirectoryTestBucket = "This is from directory Test/b file b\n"
const RenameFile = "rename.txt"
const RenameDir = "rename"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func createTestDataForReadOnlyTests(ctx context.Context, storageClient *storage.Client) error {
	// Define the text to write and the files to create
	files := []struct {
		fileContent string
		filePath    string
	}{
		{"This is from directory Test file a", TestDirForReadOnlyTest + "/Test/a.txt"},
		{"This is from file Test1", TestDirForReadOnlyTest + "/Test1.txt"},
		{"This is from directory Test/b file b", TestDirForReadOnlyTest + "/Test/b/b.txt"},
	}

	bucket, dirPath := setup.GetBucketAndObjectBasedOnTypeOfMount("")
	bucketHandle := storageClient.Bucket(bucket)

	// Loop through the file data and create/upload files
	for _, file := range files {
		filePath := path.Join(dirPath, file.filePath)
		// Create a storage writer for the destination object
		object := bucketHandle.Object(filePath)
		writer, err := client.NewWriter(ctx, object, storageClient)
		if err != nil {
			return fmt.Errorf("Error opening writer for object %s: %w\n", file.filePath, err)
		}

		// Write the text to the object
		if _, err = writer.Write([]byte(file.fileContent + "\n")); err != nil {
			log.Printf("Error writing to object %s: %v\n", file.filePath, err)
		}
		err = writer.Close()
		if err != nil {
			log.Printf("Error in closing writer: %v", err)
			return err
		}
	}

	return nil
}

func checkErrorForObjectNotExist(err error, t *testing.T) {
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Incorrect error for object not exist: %v", err.Error())
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	ctx = context.Background()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReadOnly) == 0 {
		log.Println("No configuration found for readonly tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ReadOnly = make([]test_suite.TestConfig, 1)
		cfg.ReadOnly[0].TestBucket = setup.TestBucket()
		cfg.ReadOnly[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ReadOnly[0].Configs = make([]test_suite.ConfigItem, 1)
		cacheDirPath := path.Join(os.TempDir(), "cache-dir-readonly-"+setup.GenerateRandomString(4))
		cfg.ReadOnly[0].Configs[0].Flags = []string{
			"--o=ro --implicit-dirs=true",
			"--file-mode=544 --dir-mode=544 --implicit-dirs=true",
			"--client-protocol=grpc --o=ro --implicit-dirs=true",
			fmt.Sprintf("--o=ro --implicit-dirs=true --cache-dir=%s --file-cache-max-size-mb=3", cacheDirPath),
		}
		cfg.ReadOnly[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	// 2. Create storage client before running tests.
	bucketType := setup.TestEnvironment(ctx, &cfg.ReadOnly[0])
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Create test data.
	if err := createTestDataForReadOnlyTests(ctx, storageClient); err != nil {
		log.Fatalf("Failed creating test data for readonly tests: %v", err)
	}

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set.
	if cfg.ReadOnly[0].GKEMountedDirectory != "" && cfg.ReadOnly[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.ReadOnly[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.ReadOnly[0], bucketType, "")
	setup.SetUpTestDirForTestBucket(&cfg.ReadOnly[0])

	// 5. Run tests.
	successCode := static_mounting.RunTestsWithConfigFile(&cfg.ReadOnly[0], flags, m)
	if successCode == 0 {
		successCode = persistent_mounting.RunTestsWithConfigFile(&cfg.ReadOnly[0], flags, m)
	}
	if successCode == 0 {
		// These tests don't apply to GCSFuse sidecar.
		// Validate that tests work with viewer permission on test bucket.
		successCode = creds_tests.RunTestsForDifferentAuthMethods(ctx, &cfg.ReadOnly[0], storageClient, flags, "objectViewer", m)
	}

	os.Exit(successCode)
}

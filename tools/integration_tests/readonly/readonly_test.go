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
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
	cacheDir      string
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

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	cacheDirPath := path.Join(os.TempDir(), cacheDir)

	// Set up config file for file cache.
	mountConfig := map[string]interface{}{
		"file-cache": map[string]interface{}{
			// Keeping the size as small because the operations are performed on small
			// files.
			"max-size-mb": 3,
		},
		"cache-dir": cacheDirPath,
	}
	filePath := setup.YAMLConfigFile(mountConfig, "config.yaml")
	flags = append(flags, []string{"--o=ro", "--implicit-dirs=true", "--config-file=" + filePath})

	return flags
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	var err error
	ctx = context.Background()
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	cacheDir = "cache-dir-readonly-hns-" + strconv.FormatBool(setup.IsHierarchicalBucket(ctx, storageClient))

	flags := [][]string{{"--o=ro", "--implicit-dirs=true"}, {"--file-mode=544", "--dir-mode=544", "--implicit-dirs=true"}}

	if !testing.Short() {
		flags = append(flags, []string{"--client-protocol=grpc", "--o=ro", "--implicit-dirs=true"})
	}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Create test data.
	if err := createTestDataForReadOnlyTests(ctx, storageClient); err != nil {
		log.Printf("Failed creating test data for readonly tests: %v", err)
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	// Setup config file for tests when --testbucket flag is enabled.
	mountConfigFlags := createMountConfigsAndEquivalentFlags()
	flags = append(flags, mountConfigFlags...)

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = persistent_mounting.RunTests(flags, m)
	}

	if successCode == 0 {
		// Test for viewer permission on test bucket.
		successCode = creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(ctx, storageClient, flags, "objectViewer", m)
	}

	os.Exit(successCode)
}

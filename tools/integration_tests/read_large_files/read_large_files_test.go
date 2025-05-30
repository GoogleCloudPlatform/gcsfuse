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

// Provides integration tests for read large files sequentially and randomly.
package read_large_files

import (
	"context"
	"log"
	"os"
	"path"
	"strconv"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const OneMB = 1024 * 1024
const FiveHundredMB = 500 * OneMB
const ChunkSize = 200 * OneMB
const NumberOfRandomReadCalls = 200
const MinReadableByteFromFile = 0
const MaxReadableByteFromFile = 500 * OneMB
const DirForReadLargeFilesTests = "dirForReadLargeFilesTests"

var (
	storageClient     *storage.Client
	ctx               context.Context
	FiveHundredMBFile = "fiveHundredMBFile" + setup.GenerateRandomString(5) + ".txt"
	cacheDir          string
)

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	cacheDirPath := path.Join(os.TempDir(), cacheDir)

	// Set up config file for file cache with cache-file-for-range-read: false
	mountConfig1 := map[string]interface{}{
		"file-cache": map[string]interface{}{
			// Keeping the size as high because the operations are performed on large
			// files.
			"max-size-mb":               700,
			"cache-file-for-range-read": true,
		},
		"cache-dir": cacheDirPath,
	}
	filePath1 := setup.YAMLConfigFile(mountConfig1, "config1.yaml")
	flags = append(flags, []string{"--implicit-dirs=true", "--config-file=" + filePath1})

	// Set up config file for file cache with unlimited capacity
	mountConfig2 := map[string]interface{}{
		"file-cache": map[string]interface{}{
			"max-size-mb":               -1,
			"cache-file-for-range-read": false,
		},
		"cache-dir": cacheDirPath,
	}
	filePath2 := setup.YAMLConfigFile(mountConfig2, "config2.yaml")
	flags = append(flags, []string{"--implicit-dirs=true", "--config-file=" + filePath2})

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
	cacheDir = "cache-dir-read-large-files-hns-" + strconv.FormatBool(setup.IsHierarchicalBucket(ctx, storageClient))

	flags := [][]string{{"--implicit-dirs"}}
	mountConfigFlags := createMountConfigsAndEquivalentFlags()
	flags = append(flags, mountConfigFlags...)
	setup.AppendFlagsToAllFlagsInTheFlagsSet(&flags, "", "--client-protocol=grpc")

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)
	os.Exit(successCode)
}

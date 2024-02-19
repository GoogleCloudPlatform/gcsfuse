// Copyright 2023 Google Inc. All Rights Reserved.
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
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirectoryNameInTestBucket = "Test"         //  testBucket/Test
const FileNameInTestBucket = "Test1.txt"         //  testBucket/Test1.txt
const SubDirectoryNameInTestBucket = "b"         //  testBucket/Test/b
const FileNameInDirectoryTestBucket = "a.txt"    //  testBucket/Test/a.txt
const FileNameInSubDirectoryTestBucket = "b.txt" //  testBucket/Test/b/b.txt
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

func checkErrorForReadOnlyFileSystem(err error, t *testing.T) {
	if !strings.Contains(err.Error(), "read-only file system") && !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Incorrect error for readonly filesystem: %v", err.Error())
	}
}

func checkErrorForObjectNotExist(err error, t *testing.T) {
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Incorrect error for object not exist: %v", err.Error())
	}
}

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	cacheDirPath := path.Join(os.Getenv("HOME"), "cache-dir")

	// Set up config file for file cache
	mountConfig := config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			// Keeping the size as small because the operations are performed on small
			// files
			MaxSizeMB: 3,
		},
		CacheDir: config.CacheDir(cacheDirPath),
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, "config.yaml")
	flags = append(flags, []string{"--o=ro", "--implicit-dirs=true", "--config-file=" + filePath})

	return flags
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--o=ro", "--implicit-dirs=true"}, {"--file-mode=544", "--dir-mode=544", "--implicit-dirs=true"}}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Clean the bucket for readonly testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())

	// Create objects in bucket for testing.
	setup.RunScriptForTestData("testdata/create_objects.sh", setup.TestBucket())

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
		successCode = creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(flags, "objectViewer", m)
	}

	// Delete objects from bucket after testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())

	setup.RemoveBinFileCopiedForTesting()

	os.Exit(successCode)
}

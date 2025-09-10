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

package streaming_writes

import (
	"os"
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/assert"
)

func TestWritesWithDifferentConfig(t *testing.T) {
	// Do not run this test with mounted directory flag.
	if setup.MountedDirectory() != "" {
		t.SkipNow()
	}
	// Create a separate mountDir for these tests so it doesn't interfere with the other tests.
	oldMntDir := setup.MntDir()
	newMountDir := path.Join(setup.TestDir(), "mntTestWritesWithDifferentConfig")
	err := os.MkdirAll(newMountDir, 0755)
	assert.True(t, err == nil || os.IsExist(err))
	mountFunc := static_mounting.MountGcsfuseWithStaticMounting
	setup.SetGlobalVars(&test_suite.TestConfig{
		TestBucket:          setup.TestBucket(),
		LogFile:             setup.LogFile(),
		GKEMountedDirectory: newMountDir,
	})
	defer setup.SetGlobalVars(&test_suite.TestConfig{
		TestBucket:          setup.TestBucket(),
		LogFile:             setup.LogFile(),
		GKEMountedDirectory: oldMntDir,
	})

	testCases := []struct {
		name     string
		flags    []string
		fileSize int64
	}{
		{
			name:     "BlockSizeGreaterThanFileSize",
			flags:    []string{"--write-block-size-mb=5", "--write-max-blocks-per-file=2"},
			fileSize: 2 * 1024 * 1024,
		},
		{
			name:     "BlockSizeLessThanFileSize",
			flags:    []string{"--write-block-size-mb=1", "--write-max-blocks-per-file=20"},
			fileSize: 5 * 1024 * 1024,
		},
		{
			// BlockSize*num_blocks < fileSize
			name:     "NumberOfBlocksLessThanFileSize",
			flags:    []string{"--write-block-size-mb=1", "--write-max-blocks-per-file=2"},
			fileSize: 10 * 1024 * 1024,
		},
		{
			name:     "BlockSizeEqualToFileSize",
			flags:    []string{"--write-block-size-mb=5", "--write-max-blocks-per-file=2"},
			fileSize: 5 * 1024 * 1024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setup.MountGCSFuseWithGivenMountFunc(tc.flags, mountFunc)
			defer setup.SaveGCSFuseLogFileInCaseOfFailure(t)
			defer setup.UnmountGCSFuse(newMountDir)
			testEnv.testDirPath = setup.SetupTestDirectory(testDirName)
			// Create a local file.
			fh := operations.CreateFile(path.Join(testEnv.testDirPath, FileName1), FilePerms, t)
			testDirName := GetDirName(testEnv.testDirPath)
			if setup.IsZonalBucketRun() {
				ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, testDirName, FileName1, "", t)
			} else {
				ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, testDirName, FileName1, t)
			}
			data, err := operations.GenerateRandomData(tc.fileSize)
			if err != nil {
				t.Fatalf("Error in generating data: %v", err)
			}

			// Write data to file.
			operations.WriteAt(string(data[:]), 0, fh, t)

			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, fh, testDirName, FileName1, string(data[:]), t)
		})
	}
}

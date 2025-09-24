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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritesWithDifferentConfig(t *testing.T) {
	// Do not run this test with mounted directory flag.
	if testEnv.cfg.GKEMountedDirectory != "" {
		t.SkipNow()
	}
	// Create a separate mountDir for these tests so it doesn't interfere with the other tests.
	oldMntDir := testEnv.cfg.GCSFuseMountedDirectory
	newMountDir := path.Join(setup.TestDir(), "mntTestWritesWithDifferentConfig")
	err := os.MkdirAll(newMountDir, 0755)
	assert.True(t, err == nil || os.IsExist(err))
	testEnv.cfg.GCSFuseMountedDirectory = newMountDir
	defer func() {
		testEnv.cfg.GCSFuseMountedDirectory = oldMntDir
	}()
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
			err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(&testEnv.cfg, tc.flags)
			require.NoError(t, err)
			defer setup.SaveGCSFuseLogFileInCaseOfFailure(t)
			defer setup.UnmountGCSFuseWithConfig(&testEnv.cfg)
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

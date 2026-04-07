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

package read_large_files

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

var FileOne = "fileOne" + setup.GenerateRandomString(5) + ".txt"
var FileTwo = "fileTwo" + setup.GenerateRandomString(5) + ".txt"
var FileThree = "fileThree" + setup.GenerateRandomString(5) + ".txt"

const NumberOfFilesInLocalDiskForConcurrentRead = 3

func TestReadFilesConcurrently(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForReadLargeFilesTests)

	fileNames := [NumberOfFilesInLocalDiskForConcurrentRead]string{FileOne, FileTwo, FileThree}
	localFilePaths := make([]string, NumberOfFilesInLocalDiskForConcurrentRead)
	mntFilePaths := make([]string, NumberOfFilesInLocalDiskForConcurrentRead)

	var creationGroup errgroup.Group
	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		fileIndex := i
		creationGroup.Go(func() error {
			localFilePath := path.Join(os.Getenv("HOME"), fileNames[fileIndex])
			localFilePaths[fileIndex] = localFilePath

			mntFilePath := path.Join(testDir, fileNames[fileIndex])
			mntFilePaths[fileIndex] = mntFilePath

			operations.CreateFileOnDiskAndCopyToMntDir(t, localFilePath, mntFilePath, FiveHundredMB)
			return nil
		})
	}
	if err := creationGroup.Wait(); err != nil {
		t.Fatalf("Error creating files: %v", err)
	}

	// Register cleanup for local files.
	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		filePath := localFilePaths[i]
		t.Cleanup(func() {
			operations.RemoveFile(filePath)
		})
	}

	var eG errgroup.Group

	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to read the current file.
		eG.Go(func() error {
			operations.ReadAndCompare(t, mntFilePaths[fileIndex], localFilePaths[fileIndex], 0, FiveHundredMB)
			return nil
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

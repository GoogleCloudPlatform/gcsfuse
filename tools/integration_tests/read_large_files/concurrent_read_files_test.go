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

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

var FileOne = "fileOne" + setup.GenerateRandomString(5) + ".txt"
var FileTwo = "fileTwo" + setup.GenerateRandomString(5) + ".txt"
var FileThree = "fileThree" + setup.GenerateRandomString(5) + ".txt"

const NumberOfFilesInLocalDiskForConcurrentRead = 3

func TestReadFilesConcurrently(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForReadLargeFilesTests)

	filesInLocalDisk := [NumberOfFilesInLocalDiskForConcurrentRead]string{FileOne, FileTwo, FileThree}
	var filesPathInLocalDisk []string
	var filesPathInMntDir []string

	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		fileInLocalDisk := path.Join(os.Getenv("HOME"), filesInLocalDisk[i])
		filesPathInLocalDisk = append(filesPathInLocalDisk, fileInLocalDisk)

		file := path.Join(testDir, filesInLocalDisk[i])
		filesPathInMntDir = append(filesPathInMntDir, file)

		operations.CreateFileOnDiskAndCopyToMntDir(t, fileInLocalDisk, file, FiveHundredMB)
	}

	var eG errgroup.Group

	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to read the current file.
		eG.Go(func() error {
			operations.ReadAndCompare(t, filesPathInMntDir[fileIndex], filesPathInLocalDisk[fileIndex], 0, FiveHundredMB)
			return nil
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	for i := range NumberOfFilesInLocalDiskForConcurrentRead {
		operations.RemoveFile(filesPathInLocalDisk[i])
	}
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

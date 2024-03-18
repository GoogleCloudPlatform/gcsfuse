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

package read_large_files

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

const FileOne = "fileOne.txt"
const FileTwo = "fileTwo.txt"
const FileThree = "fileThree.txt"
const NumberOfFilesInLocalDiskForConcurrentRead = 3

func readFile(fileInLocalDisk string, fileInMntDir string) error {
	dataInMntDirFile, err := operations.ReadFile(fileInMntDir)
	if err != nil {
		return err
	}

	dataInLocalDiskFile, err := operations.ReadFile(fileInLocalDisk)
	if err != nil {
		return err
	}

	// Compare actual content and expect content.
	if bytes.Equal(dataInLocalDiskFile, dataInMntDirFile) == false {
		return fmt.Errorf("Reading incorrect file.")
	}

	return nil
}

func TestReadFilesConcurrently(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	filesInLocalDisk := [NumberOfFilesInLocalDiskForConcurrentRead]string{FileOne, FileTwo, FileThree}
	var filesPathInLocalDisk []string
	var filesPathInMntDir []string

	for i := 0; i < NumberOfFilesInLocalDiskForConcurrentRead; i++ {
		fileInLocalDisk := path.Join(os.Getenv("HOME"), filesInLocalDisk[i])
		filesPathInLocalDisk = append(filesPathInLocalDisk, fileInLocalDisk)

		file := path.Join(setup.MntDir(), filesInLocalDisk[i])
		filesPathInMntDir = append(filesPathInMntDir, file)

		createFileOnDiskAndCopyToMntDir(fileInLocalDisk, file, FiveHundredMB, t)
	}

	var eG errgroup.Group

	for i := 0; i < NumberOfFilesInLocalDiskForConcurrentRead; i++ {
		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to read the current file.
		eG.Go(func() error {
			return readFile(filesPathInLocalDisk[fileIndex], filesPathInMntDir[fileIndex])
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

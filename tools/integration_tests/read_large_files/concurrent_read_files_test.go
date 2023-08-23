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
	"os"
	"path"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const FileOne = "fileOne.txt"
const FileTwo = "fileTwo.txt"
const FileThree = "fileThree.txt"
const NumberOfFilesInLocalDiskForConcurrentRead = 3

func readFile(fileInLocalDisk string, fileInMntDir string, wg *sync.WaitGroup, t *testing.T) {
	// Reduce thread count when it read the file.
	defer wg.Done()

	dataInMntDirFile, err := operations.ReadFile(fileInMntDir)
	if err != nil {
		return
	}

	dataInLocalDiskFile, err := operations.ReadFile(fileInLocalDisk)
	if err != nil {
		return
	}

	// Compare actual content and expect content.
	if bytes.Equal(dataInLocalDiskFile, dataInMntDirFile) == false {
		t.Errorf("Reading incorrect file.")
	}
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

	// For waiting on threads.
	var wg sync.WaitGroup

	for i := 0; i < NumberOfFilesInLocalDiskForConcurrentRead; i++ {
		// Increment the WaitGroup counter.
		wg.Add(1)
		// Thread to read file.
		go readFile(filesPathInLocalDisk[i], filesPathInMntDir[i], &wg, t)
	}

	// Wait on threads to end.
	wg.Wait()
}

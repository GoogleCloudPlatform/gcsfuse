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
	"log"
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

var tokens = make(chan struct{}, 10)

func ReadFileParellaly(fileInLocalDisk string, fileInMntDir string, wg *sync.WaitGroup, t *testing.T) {
	log.Print(fileInMntDir)
	// For wait group (wait until all threads done).
	defer wg.Done()

	// Acquire token.
	tokens <- struct{}{}

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

	// Release token.
	<-tokens
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Create file of 500 MB with random data in local disk.
	fileInLocalDisk1 := path.Join(os.Getenv("HOME"), FileOne)
	setup.RunScriptForTestData("testdata/write_content_of_fix_size_in_file.sh", fileInLocalDisk1, "200")

	fileInLocalDisk2 := path.Join(os.Getenv("HOME"), FileTwo)
	setup.RunScriptForTestData("testdata/write_content_of_fix_size_in_file.sh", fileInLocalDisk2, "200")

	fileInLocalDisk3 := path.Join(os.Getenv("HOME"), FileThree)
	setup.RunScriptForTestData("testdata/write_content_of_fix_size_in_file.sh", fileInLocalDisk3, "200")

	file1 := path.Join(setup.MntDir(), FileOne)
	CopyFileFromLocalDiskToMntDir(fileInLocalDisk1, file1, t)

	file2 := path.Join(setup.MntDir(), FileTwo)
	CopyFileFromLocalDiskToMntDir(fileInLocalDisk2, file2, t)

	file3 := path.Join(setup.MntDir(), FileThree)
	CopyFileFromLocalDiskToMntDir(fileInLocalDisk3, file3, t)

	// For waiting on threads.
	var wg sync.WaitGroup

	// Increment the WaitGroup counter.
	wg.Add(1)
	// Thread.
	go ReadFileParellaly(fileInLocalDisk1, file1, &wg, t)

	// Increment the WaitGroup counter.
	wg.Add(2)
	// Thread.
	go ReadFileParellaly(fileInLocalDisk2, file2, &wg, t)

	// Increment the WaitGroup counter.
	wg.Add(3)
	// Thread.
	go ReadFileParellaly(fileInLocalDisk3, file3, &wg, t)

	// Wait on threads to end.
	wg.Wait()
}

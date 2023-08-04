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

package write_large_files

import (
	"crypto/rand"
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
const NumberOfFilesInLocalDiskForConcurrentWrite = 3

func writeFile(fileName string, fileSize int64, wg *sync.WaitGroup, t *testing.T) {
	file := path.Join(setup.MntDir(), fileName)

	// Reduce thread count when it is done.
	defer wg.Done()

	data := make([]byte, fileSize)
	_, err := rand.Read(data)
	if err != nil {
		t.Errorf("error while generating random string: %s", err)
	}

	err = operations.WriteFile(file, string(data))
	if err != nil {
		return
	}

	// Download the file from a bucket in which we write the content.
	fileInBucket := path.Join(os.Getenv("HOME"), fileName)
	setup.RunScriptForTestData("../util/operations/download_file_from_bucket.sh", setup.TestBucket(), fileName, fileInBucket)

	_, err = operations.StatFile(fileInBucket)
	if err != nil {
		t.Errorf("Error in stating file:%v", err)
	}

	_, err = operations.DiffFiles(file, fileInBucket)

	if err != nil {
		t.Errorf("Error in writing files concurrently.")
	}
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// For waiting on threads.
	var wg sync.WaitGroup

	files := [NumberOfFilesInLocalDiskForConcurrentWrite]string{FileOne, FileTwo, FileThree}

	for i := 0; i < NumberOfFilesInLocalDiskForConcurrentWrite; i++ {
		// Increment the WaitGroup counter.
		wg.Add(1)
		// Thread to write first file.
		go writeFile(files[i], FiveHundredMB, &wg, t)
	}

	// Wait on threads to end.
	wg.Wait()
}

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

const (
	FileOne                                    = "fileOne.txt"
	FileTwo                                    = "fileTwo.txt"
	FileThree                                  = "fileThree.txt"
	NumberOfFilesInLocalDiskForConcurrentWrite = 3
	DirForConcurrentWrite                      = "dirForConcurrentWrite"
)

func writeFile(fileName string, fileSize int64, wg *sync.WaitGroup, t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirForConcurrentWrite, fileName)

	// Reduce thread count when it is done.
	defer wg.Done()

	data := make([]byte, fileSize)
	_, err := rand.Read(data)
	if err != nil {
		t.Errorf("error while generating random string: %s", err)
	}

	err = operations.WriteFile(filePath, string(data))
	if err != nil {
		return
	}

	// Download the file from a bucket in which we write the content.
	filePathInGcsBucket := path.Join(setup.TestBucket(), DirForConcurrentWrite, fileName)
	localFilePath := path.Join(TmpDir, fileName)
	err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
	if err != nil {
		t.Fatalf("Error:%v", err)
	}
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	concurrentWriteDir := path.Join(setup.MntDir(), DirForConcurrentWrite)
	err := os.Mkdir(concurrentWriteDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory:%v", err)
	}
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

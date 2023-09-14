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
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	FileOne               = "fileOne.txt"
	FileTwo               = "fileTwo.txt"
	FileThree             = "fileThree.txt"
	DirForConcurrentWrite = "dirForConcurrentWrite"
)

func writeFile(fileName string, fileSize int64) (err error) {
	filePath := path.Join(setup.MntDir(), DirForConcurrentWrite, fileName)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, WritePermission_0200)
	if err != nil {
		err = fmt.Errorf("Open file for write at start: %v", err)
		return err
	}

	// Closing file at the end.
	defer operations.CloseFile(f)

	err = writeChunkSizeInFile(f, int(fileSize), 0)
	if err != nil {
		err = fmt.Errorf("Error:%v", err)
		return err
	}

	filePathInGcsBucket := path.Join(setup.TestBucket(), DirForConcurrentWrite, fileName)
	localFilePath := path.Join(TmpDir, fileName)
	err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
	if err != nil {
		err = fmt.Errorf("Error:%v", err)
		return err
	}

	return
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	concurrentWriteDir := path.Join(setup.MntDir(), DirForConcurrentWrite)
	err := os.Mkdir(concurrentWriteDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory:%v", err)
	}

	// Clean up.
	defer operations.RemoveDir(concurrentWriteDir)

	// For waiting on threads.
	var wg sync.WaitGroup

	files := []string{FileOne, FileTwo, FileThree}

	// Concurrently write three files.
	for i := range files {
		// Increment the WaitGroup counter.
		wg.Add(1)

		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to write the current file.
		go func() {
			// Reduce thread count when it is done.
			defer wg.Done()
			err = writeFile(files[fileIndex], FiveHundredMB)
			if err != nil {
				log.Fatalf("Error:%v", err)
			}
		}()
	}

	// Wait on threads to end.
	wg.Wait()
}

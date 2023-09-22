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
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

const (
	FileOne               = "fileOne.txt"
	FileTwo               = "fileTwo.txt"
	FileThree             = "fileThree.txt"
	DirForConcurrentWrite = "dirForConcurrentWrite"
)

func writeFile(fileName string, fileSize int64) error {
	filePath := path.Join(setup.MntDir(), DirForConcurrentWrite, fileName)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, WritePermission_0200)
	if err != nil {
		return fmt.Errorf("Open file for write at start: %v", err)
	}

	// Closing file at the end.
	defer operations.CloseFile(f)

	err = operations.WriteChunkOfRandomBytesToFile(f, int(fileSize), 0)
	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	filePathInGcsBucket := path.Join(setup.TestBucket(), DirForConcurrentWrite, fileName)
	localFilePath := path.Join(TmpDir, fileName)
	err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	return nil
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	concurrentWriteDir := path.Join(setup.MntDir(), DirForConcurrentWrite)
	err := os.Mkdir(concurrentWriteDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory: %v", err)
	}

	// Clean up.
	defer operations.RemoveDir(concurrentWriteDir)

	files := []string{FileOne, FileTwo, FileThree}

	var eG errgroup.Group

	// Concurrently write three files.
	for i := range files {
		// Copy the current value of i into a local variable to avoid data races.
		fileIndex := i

		// Thread to write the current file.
		eG.Go(func() error {
			return writeFile(files[fileIndex], FiveHundredMB)
		})
	}

	// Wait on threads to end.
	err = eG.Wait()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

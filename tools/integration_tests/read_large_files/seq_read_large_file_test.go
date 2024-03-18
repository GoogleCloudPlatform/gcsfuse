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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func TestReadLargeFileSequentially(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Create file of 500 MB with random data in local disk and copy it in mntDir.
	fileInLocalDisk := path.Join(os.Getenv("HOME"), FiveHundredMBFile)
	file := path.Join(setup.MntDir(), FiveHundredMBFile)
	createFileOnDiskAndCopyToMntDir(fileInLocalDisk, file, FiveHundredMB, t)

	// Sequentially read the data from file.
	content, err := operations.ReadFileSequentially(file, ChunkSize)
	if err != nil {
		t.Errorf("Error in reading file: %v", err)
	}

	// Read actual content from file located in local disk.
	actualContent, err := operations.ReadFile(fileInLocalDisk)
	if err != nil {
		t.Errorf("Error in reading file: %v", err)
	}

	// Compare actual content and expect content.
	if bytes.Equal(actualContent, content) == false {
		t.Errorf("Error in reading file sequentially.")
	}

	// Removing file after testing.
	operations.RemoveFile(fileInLocalDisk)
}

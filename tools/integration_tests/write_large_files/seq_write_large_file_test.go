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
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const OneMB = 1024 * 1024
const FiveHundredMB = 500 * OneMB
const FiveHundredMBFile = "fiveHundredMBFile.txt"
const ChunkSize = 20 * OneMB

func TestWriteLargeFileSequentially(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	filePath := path.Join(setup.MntDir(), FiveHundredMBFile)

	// Sequentially read the data from file.
	err := operations.WriteFileSequentially(filePath, FiveHundredMB, ChunkSize)
	if err != nil {
		t.Errorf("Error in writing file: %v", err)
	}

	// Download the file from a bucket in which we write the content.
	fileInBucket := path.Join(os.Getenv("HOME"), FileDownloadedFromBucket)
	setup.RunScriptForTestData("../util/operations/download_file_from_bucket.sh", setup.TestBucket(), FiveHundredMBFile, fileInBucket)

	contentInFileDownloadedFromBucket, err := operations.ReadFile(fileInBucket)
	if err != nil {
		t.Errorf("Error in reading file.")
	}

	// Check if 500MB data written in the file.
	fStat, err := os.Stat(fileInBucket)
	if err != nil {
		t.Errorf("Error in stating file:%v", err)
	}

	if fStat.Size() != FiveHundredMB {
		t.Errorf("Expecred file size %v found %d", FiveHundredMB, fStat.Size())
	}

	contentInFileFromMntDir, err := operations.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error in reading file.")
	}

	// Compare actual content and expect content.
	if bytes.Equal(contentInFileFromMntDir, contentInFileDownloadedFromBucket) == false {
		t.Errorf("Incorrect content written in the file.")
	}

	// Remove file after testing.
	operations.RemoveFile(fileInBucket)
}

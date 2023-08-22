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
const DirForSeqWrite = "dirForSeqWrite"
const FiveHundredMBFileForSeqWriteInLocalSystem = "fiveHundredMBFileForSeqWriteInLocalSystem"

func TestWriteLargeFileSequentially(t *testing.T) {
	seqWriteDir := path.Join(setup.MntDir(), DirForSeqWrite)
	err := os.Mkdir(seqWriteDir, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory:%v", err)
	}
	filePath := path.Join(seqWriteDir, FiveHundredMBFile)

	// Sequentially read the data from file.
	err = operations.WriteFileSequentially(filePath, FiveHundredMB, ChunkSize)
	if err != nil {
		t.Errorf("Error in writing file: %v", err)
	}

	// Download the file from a bucket in which we write the content and compare with
	// the file content we wrote in mntDir.
	filePathInGcsBucket := path.Join(setup.TestBucket(), DirForSeqWrite, FiveHundredMBFile)
	localFilePath := path.Join(os.Getenv("HOME"), FiveHundredMBFileForSeqWriteInLocalSystem)
	err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
	if err != nil {
		t.Errorf("Error:%v", err)
	}

	// Clean up.
	operations.RemoveDir(seqWriteDir)
}

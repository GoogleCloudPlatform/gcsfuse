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

const FiveHundredMB = 500 * 1024 * 1024
const FiveHundredMBFile = "fiveHundredMBFile.txt"
const ChunkSize = 20 * 1024 * 1024

func TestWriteLargeFileSequentially(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	filePath := path.Join(setup.MntDir(), FiveHundredMBFile)

	// Sequentially read the data from file.
	err := operations.WriteFileSequentially(filePath, FiveHundredMB, ChunkSize)
	if err != nil {
		t.Errorf("Error in writing file: %v", err)
	}

	// Check if 500MB data written in the file.
	fStat, err := os.Stat(filePath)
	if err != nil {
		t.Errorf("Error in stating file:%v", err)
	}

	if fStat.Size() != FiveHundredMB {
		t.Errorf("Expecred file size %v found %d", FiveHundredMB, fStat.Size())
	}
}

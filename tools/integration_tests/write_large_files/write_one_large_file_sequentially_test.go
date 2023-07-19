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

const HundredMB = 1048576
const HundredMBFile = "hundredMBFile.txt"

func TestWriteLargeFileSequentially(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	filePath := path.Join(setup.MntDir(), HundredMBFile)

	// Sequentially read the data from file.
	err := operations.WriteFileSequentially(filePath, HundredMB)
	if err != nil {
		t.Errorf("Error in writing file: %v", err)
	}

	fStat, err := os.Stat(filePath)
	if fStat.Size() != HundredMB {
		t.Errorf("Expecred file size %v found %d", HundredMB, fStat.Size())
	}
}

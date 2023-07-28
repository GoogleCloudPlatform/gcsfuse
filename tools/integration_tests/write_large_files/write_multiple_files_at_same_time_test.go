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

func WriteFileParellaly(file string, fileSize int64, wg *sync.WaitGroup, t *testing.T) {
	// For wait group (wait until all threads done).
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

	f, err := os.Stat(file)
	if err != nil {
		t.Errorf("Error in stating file:%v", err)
	}
	if f.Size() != fileSize {
		t.Errorf("Error in writing multiple files at same time.")
	}
}

func TestMultipleFilesAtSameTime(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// For waiting on threads.
	var wg sync.WaitGroup

	file1 := path.Join(setup.MntDir(), FileOne)
	file2 := path.Join(setup.MntDir(), FileTwo)
	file3 := path.Join(setup.MntDir(), FileThree)

	// Increment the WaitGroup counter.
	wg.Add(1)
	// Thread to read first file.
	go WriteFileParellaly(file1, FiveHundredMB, &wg, t)

	// Increment the WaitGroup counter.
	wg.Add(1)
	// Thread to read second file.
	go WriteFileParellaly(file2, FiveHundredMB, &wg, t)

	// Increment the WaitGroup counter.
	wg.Add(1)
	// Thread to read third file.
	go WriteFileParellaly(file3, FiveHundredMB, &wg, t)

	// Wait on threads to end.
	wg.Wait()
}

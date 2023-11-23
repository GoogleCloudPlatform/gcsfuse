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

package log_rotation

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	MiB          = 1024 * 1024
	filePerms    = 0644
	testFileName = "foo"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func generateRandomData(t *testing.T, sizeInBytes int64) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]byte, sizeInBytes)
	_, err := r.Read(data)
	if err != nil {
		t.Errorf("error while generating random data to write to file.")
	}
	return data
}

func runOperationsOnFileTillLogRotation(t *testing.T, wg *sync.WaitGroup, fileName string) {
	defer wg.Done()

	testDirPath := path.Join(setup.MntDir(), testDirName)
	// Generate random data to write to file.
	randomData := generateRandomData(t, MiB)
	// Setup file to write to.
	filePath := path.Join(testDirPath, fileName)
	fh := operations.CreateFile(filePath, filePerms, t)

	// Keep performing operations in mounted directory until log file is rotated.
	var lastLogFileSize int64 = 0
	for {
		operations.WriteAt(string(randomData), 0, fh, t)

		fi, err := operations.StatFile(logFilePath)
		if err != nil {
			t.Errorf("stat operation on file %s: %v", logFilePath, err)
		}
		if (*fi).Size() < lastLogFileSize {
			// Log file got rotated as current log file size < last log file size.
			break
		}
		lastLogFileSize = (*fi).Size()
	}
	operations.CloseFileShouldNotThrowError(fh, t)
}

func runParallelOperationsInMountedDirectoryTillLogRotation(t *testing.T) {
	// Parallely perform operations on 10 files in-order to generate logs.
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go runOperationsOnFileTillLogRotation(t, &wg, fmt.Sprintf(testFileName+"-%d", i))
	}
	wg.Wait()
}

func validateLogFileSize(t *testing.T, dirEntry os.DirEntry) {
	fi, err := dirEntry.Info()
	if err != nil {
		t.Fatalf("log file size could not be fetched: %v", err)
	}
	if fi.Size() > MiB {
		t.Errorf("log file size: expected (max): %d, actual: %d", MiB, fi.Size())
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestLogRotation(t *testing.T) {
	setup.SetupTestDirectory(testDirName)

	// Perform log rotation once.
	runParallelOperationsInMountedDirectoryTillLogRotation(t)

	// Validate that in the end we have one compressed rotated log file and one
	// active log file.
	dirEntries := operations.ReadDirectory(logDirPath, t)
	if len(dirEntries) != 2 {
		t.Errorf("Expected log files in dirEntries folder: 2, got: %d", len(dirEntries))
	}
	rotatedCompressedFileCtr := 0
	logFileCtr := 0
	rotatedUncompressedFileCtr := 0
	for i := 0; i < 2; i++ {
		if dirEntries[i].Name() == logFileName {
			logFileCtr++
			validateLogFileSize(t, dirEntries[i])
		} else if strings.Contains(dirEntries[i].Name(), "txt.gz") {
			rotatedCompressedFileCtr++
		} else {
			rotatedUncompressedFileCtr++
			validateLogFileSize(t, dirEntries[i])
		}
	}

	// validate only 1 log file is present.
	if logFileCtr != 1 {
		t.Errorf("expected countOfLogFile: 1, got: %d", logFileCtr)
	}
	// validate only 1 rotated log file is present.
	rotatedLogFiles := rotatedCompressedFileCtr + rotatedUncompressedFileCtr
	if rotatedLogFiles != 1 {
		t.Errorf("expected rotated files: 1, got: %d", rotatedLogFiles)
	}
}

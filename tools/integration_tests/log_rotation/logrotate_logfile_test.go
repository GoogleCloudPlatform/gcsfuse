// Copyright 2023 Google LLC
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
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	MiB          = 1024 * 1024
	filePerms    = 0644
	testFileName = "foo"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func runOperationsOnFileTillLogRotation(t *testing.T, wg *sync.WaitGroup, fileName string) {
	defer wg.Done()

	// Generate random data to write to file.
	randomData, err := operations.GenerateRandomData(5 * MiB)
	if err != nil {
		t.Errorf("operations.GenerateRandomData: %v", err)
	}
	// Setup file with 5 MiB content in test directory.
	testDirPath := path.Join(setup.MntDir(), testDirName)
	filePath := path.Join(testDirPath, fileName)
	operations.CreateFileWithContent(filePath, filePerms, string(randomData), t)
	currentLogFile := cfg.LogFile

	// Keep performing operations in mounted directory until log file is rotated.
	var lastLogFileSize int64 = 0
	var retryStatLogFile = true
	for {
		// 1. Perform Read operation to generate logs
		_, err = operations.ReadFile(filePath)
		if err != nil {
			t.Errorf("ReadFile failed: %v", err)
		}

		// Break the loop when log file is rotated.
		fi, err := operations.StatFile(currentLogFile)
		if err != nil {
			// --- StatFile Error Handling with Retry Limit ---
			t.Logf("Stat operation on file %s failed: %v.",
				currentLogFile, err)
			if !retryStatLogFile {
				t.Errorf("Stat retry exhausted on log file")
			}
			retryStatLogFile = false
			continue
		}
		if (*fi).Size() < lastLogFileSize {
			// Log file got rotated as current log file size < last log file size.
			break
		}
		lastLogFileSize = (*fi).Size()
	}
}

func runParallelOperationsInMountedDirectoryTillLogRotation(t *testing.T) {
	// Parallelly performs operations on 5 files in-order to generate logs.
	var wg sync.WaitGroup
	wg.Add(5)
	for i := range 5 {
		go runOperationsOnFileTillLogRotation(t, &wg, fmt.Sprintf(testFileName+"-%d", i))
	}
	wg.Wait()
}

func validateLogFileSize(t *testing.T, dirEntry os.DirEntry) {
	fi, err := dirEntry.Info()
	if err != nil {
		t.Fatalf("log file size could not be fetched: %v", err)
	}
	if fi.Size() > maxFileSizeMB*MiB {
		t.Errorf("log file size: expected (max): %d, actual: %d", maxFileSizeMB*MiB, fi.Size())
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestLogRotation(t *testing.T) {
	setup.SetupTestDirectory(testDirName)
	// Perform log rotation 4 times.
	for range 4 {
		runParallelOperationsInMountedDirectoryTillLogRotation(t)
	}
	// Adding 1-second sleep here because there is slight delay in compression
	// of log files.
	time.Sleep(1 * time.Second)

	// Validate log files generated.
	logFilesDirectory := path.Dir(cfg.LogFile)
	dirEntries := operations.ReadDirectory(logFilesDirectory, t)

	if len(dirEntries) != logFileCount {
		t.Errorf("Expected log files in dirEntries folder: %d, got: %d",
			logFileCount, len(dirEntries))
	}

	// Get the base name of the log file from the setup.
	activeLogFileName := t.Name() + ".log"
	rotatedCompressedFileCtr := 0
	logFileCtr := 0
	rotatedUncompressedFileCtr := 0
	for _, entry := range dirEntries {
		// Skip directories created by the test setup.
		if entry.IsDir() {
			continue
		}
		if entry.Name() == activeLogFileName {
			logFileCtr++
			validateLogFileSize(t, entry)
		} else if strings.HasSuffix(entry.Name(), ".log.gz") {
			rotatedCompressedFileCtr++
		} else if !strings.HasSuffix(entry.Name(), ".stderr") {
			rotatedUncompressedFileCtr++
			validateLogFileSize(t, entry)
		}
	}

	if logFileCtr != activeLogFileCount {
		t.Errorf("expected countOfLogFile: %d, got: %d", activeLogFileCount, logFileCtr)
	}

	rotatedLogFiles := rotatedCompressedFileCtr + rotatedUncompressedFileCtr
	if rotatedLogFiles != backupLogFileCount {
		t.Errorf("expected rotated files: %d, got: %d", backupLogFileCount, rotatedLogFiles)
	}
}

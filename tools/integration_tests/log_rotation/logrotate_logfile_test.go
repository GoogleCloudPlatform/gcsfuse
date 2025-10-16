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

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
	t.Logf("testDirPath: %s", testDirPath) // Added for debugging
	filePath := path.Join(testDirPath, fileName)
	t.Logf("filePath: %s", filePath) // Added for debugging
	operations.CreateFileWithContent(filePath, filePerms, string(randomData), t)
	t.Logf("adadas %v", logFilePath)
	// Keep performing operations in mounted directory until log file is rotated.
var lastLogFileSize int64 = 0
const maxStatRetries = 2 // Max times to retry StatFile error before failing
var currentStatRetries = 0

// Set a limit for the main loop to prevent infinite execution
const maxIterations = 2
var currentIteration = 0

	for {
		currentIteration++

		// --- 1. Check Overall Loop Limit ---
		if currentIteration > maxIterations {
			t.Fatalf("Log file rotation loop exceeded maximum iterations (%d) without rotation.", maxIterations)
		}
		t.Logf("--- Iteration %d ---", currentIteration)

		// --- 2. Perform Read operation to generate logs ---
		t.Logf("Performing ReadFile operation on: %s", filePath)
		_, err = operations.ReadFile(filePath)
		if err != nil {
			t.Errorf("ReadFile failed: %v", err)
			// Note: You might want to break or continue here depending on whether
			// a ReadFile error should halt the rotation check. Assuming it continues.
		}

		// --- 3. Stat Log File and Check for Rotation ---
		t.Logf("Checking log file size for rotation: %s", logFilePath)
		fi, err := operations.StatFile(logFilePath)

		if err != nil {
			// --- StatFile Error Handling with Retry Limit ---
			t.Logf("Stat operation on file %s failed: %v. Retry attempt %d/%d.",
				logFilePath, err, currentStatRetries+1, maxStatRetries)

			if currentStatRetries >= maxStatRetries {
				t.Fatalf("Stat operation failed persistently on log file %s after %d retries.",
					logFilePath, maxStatRetries)
				// Note: Use t.Fatalf to stop the test entirely if a crucial stat operation fails
			}
			currentStatRetries++
			continue // Skip to next iteration to retry stat
		}

		// Reset stat retry counter on successful stat
		currentStatRetries = 0

		// --- 4. Check for Rotation ---
		currentSize := (*fi).Size()
		t.Logf("Current log file size: %d bytes. Last recorded size: %d bytes.", currentSize, lastLogFileSize)

		if currentSize < lastLogFileSize {
			// Log file got rotated as current log file size < last log file size.
			t.Logf("SUCCESS: Log file rotated! Current size (%d) < last size (%d).", currentSize, lastLogFileSize)
			break
		}

		// If size increased or stayed the same, update the last recorded size
		lastLogFileSize = currentSize
		t.Logf("Log file size updated to: %d bytes. Continuing to next iteration.", lastLogFileSize)

		// Optional: Add a small sleep here to avoid thrashing the CPU if the loop is very fast
		// time.Sleep(10 * time.Millisecond)
	}
}

func runParallelOperationsInMountedDirectoryTillLogRotation(t *testing.T) {
	// Parallelly performs operations on 5 files in-order to generate logs.
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
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
	for i := 0; i < 4; i++ {
		runParallelOperationsInMountedDirectoryTillLogRotation(t)
	}
	// Adding 1-second sleep here because there is slight delay in compression
	// of log files.
	time.Sleep(1 * time.Second)

	// Validate log files generated.
	dirEntries := operations.ReadDirectory(logDirPath, t)
	if len(dirEntries) != logFileCount {
		t.Errorf("Expected log files in dirEntries folder: %d, got: %d",
			logFileCount, len(dirEntries))
	}
	rotatedCompressedFileCtr := 0
	logFileCtr := 0
	rotatedUncompressedFileCtr := 0
	for i := 0; i < logFileCount; i++ {
		if dirEntries[i].Name() == logFileName {
			logFileCtr++
			validateLogFileSize(t, dirEntries[i])
		} else if strings.Contains(dirEntries[i].Name(), "txt.gz") {
			rotatedCompressedFileCtr++
		} else if !strings.Contains(dirEntries[i].Name(), ".stderr") {
			rotatedUncompressedFileCtr++
			validateLogFileSize(t, dirEntries[i])
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

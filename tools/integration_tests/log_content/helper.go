// Copyright 2024 Google Inc. All Rights Reserved.
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

// Provides integration tests for write large files sequentially and randomly.
package log_content

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	MiB      = 1024 * 1024
	FileName = "fileName.txt"
)

var (
	logFileOffset int
)

func extractRelevantLogsFromLogFile(t *testing.T, logFile string, logFileOffset *int) (logString string) {
	// Read the entire log file at once. This can be optimized by reading
	// a bunch of lines at once, then eliminating the found
	// expected substrings one by one.
	bytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed in reading logfile %q: %v", logFile, err)
	}
	completeLogString := string(bytes)

	logString = completeLogString[*logFileOffset:]
	*logFileOffset = len(completeLogString)
	return
}

func uploadFile(t *testing.T, dirNamePrefix string, fileSize int64) {
	testDir, err := os.MkdirTemp(setup.MntDir(), dirNamePrefix+"-*")
	if err != nil || testDir == "" {
		t.Fatalf("Error in creating test-directory:%v", err)
	}
	// Clean up.
	defer operations.RemoveDir(testDir)

	filePath := path.Join(testDir, FileName)

	// Sequentially write the data in file.
	err = operations.WriteFileSequentially(filePath, fileSize, fileSize)
	if err != nil {
		t.Fatalf("Error in writing file: %v", err)
	}
}

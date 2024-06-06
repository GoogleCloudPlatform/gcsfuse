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

package log_content

import (
	"fmt"
	"math"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	BigFileSize                  int64 = 50 * operations.MiB
	SmallFileSize                int64 = operations.MiB
	DirForBigFileUploadLogTest         = "dirForBigFileUploadLogTest"
	DirForSmallFileUploadLogTest       = "dirForSmallFileUploadLogTest"
	FileName                           = "fileName.txt"
)

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

func extractRelevantLogsFromLogFile(t *testing.T, logFile string, logFileOffset int64) (logString string) {
	// Read the entire log file at once. This can be optimized by reading
	// a bunch of lines at once, then eliminating the found
	// expected substrings one by one.
	bytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed in reading logfile %q: %v", logFile, err)
	}
	completeLogString := string(bytes)

	logString = completeLogString[logFileOffset:]
	return
}

func uploadFileAndReturnLogs(t *testing.T, dirName string, fileSize int64) string {
	var err error
	var logFileOffset int64
	if logFileOffset, err = operations.SizeOfFile(setup.LogFile()); err != nil {
		t.Fatal(err)
	}

	uploadFile(t, dirName, fileSize)
	return extractRelevantLogsFromLogFile(t, setup.LogFile(), logFileOffset)
}

func TestBigFileUploadLog(t *testing.T) {
	logString := uploadFileAndReturnLogs(t, DirForBigFileUploadLogTest, BigFileSize)

	// Big files (> 16 MiB) are uploaded sequentially in chunks of size
	// 16 MiB each and each chunk's successful upload generates a log.
	gcsWriteChunkSize := 16 * operations.MiB
	numTotalChunksToBeCompleted := int(math.Floor(float64(BigFileSize) / float64(gcsWriteChunkSize)))
	var expectedSubstrings []string
	for numChunksCompletedSoFar := 1; numChunksCompletedSoFar <= numTotalChunksToBeCompleted; numChunksCompletedSoFar++ {
		expectedSubstrings = append(expectedSubstrings, fmt.Sprintf("%d bytes uploaded so far", numChunksCompletedSoFar*gcsWriteChunkSize))
	}

	operations.VerifyExpectedSubstrings(t, logString, expectedSubstrings)
}

func TestSmallFileUploadLog(t *testing.T) {
	logString := uploadFileAndReturnLogs(t, DirForSmallFileUploadLogTest, SmallFileSize)

	// The file being uploaded is too small (<16 MB) for progress logs
	// to be printed.
	unexpectedLogSubstrings := []string{"bytes uploaded so far"}
	operations.VerifyUnexpectedSubstrings(t, logString, unexpectedLogSubstrings)
}

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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	BigFileSize            = 50 * MiB
	DirForBigFileUploadLog = "dirForiBigFileUploadLog"
)

func TestBigFileUploadLog(t *testing.T) {
	testDir := path.Join(setup.MntDir(), DirForBigFileUploadLog)
	err := os.Mkdir(testDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory:%v", err)
	}
	// Clean up.
	defer operations.RemoveDir(testDir)

	filePath := path.Join(testDir, FileName)

	// Sequentially write the data in file.
	err = operations.WriteFileSequentially(filePath, BigFileSize, BigFileSize)
	if err != nil {
		t.Fatalf("Error in writing file: %v", err)
	}

	logFile := setup.LogFile()
	if logFile == "" {
		t.Fatalf("No log-file found")
	}

	// Read the entire log file at once. This can be optimized by reading
	// a bunch of lines at once, then eliminating the found
	// expected substrings one by one.
	bytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed in reading logfile %q: %v", logFile, err)
	}
	completeLogString := string(bytes)

	logString := completeLogString[logFileOffset:]

	// Big files (> 16 MiB) are uploaded sequentially in chunks of size
	// 16 MiB and each chunk's successful upload generates a log.
	gcsWriteChunkSize := 16 * MiB
	numTotalChunksToBeCompleted := int(math.Floor(float64(BigFileSize) / float64(gcsWriteChunkSize)))
	var expectedSubstrings []string
	for numChunksCompletedSoFar := 1; numChunksCompletedSoFar <= numTotalChunksToBeCompleted; numChunksCompletedSoFar++ {
		expectedSubstrings = append(expectedSubstrings, fmt.Sprintf("%d bytes uploaded so far", numChunksCompletedSoFar*gcsWriteChunkSize))
	}
	for _, expectedSubstring := range expectedSubstrings {
		if !strings.Contains(logString, expectedSubstring) {
			t.Errorf("Logfile %s does not contain expected substring (%q)", logFile, expectedSubstring)
		}
	}

	logFileOffset = len(completeLogString)
}

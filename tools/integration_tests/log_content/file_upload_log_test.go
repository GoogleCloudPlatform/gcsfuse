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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	BigFileSize              int64 = 50 * MiB
	SmallFileSize            int64 = MiB
	DirForBigFileUploadLog         = "dirForiBigFileUploadLog"
	DirForSmallFileUploadLog       = "dirForSmallFileUploadLog"
)

func TestBigFileUploadLog(t *testing.T) {
	uploadFile(t, DirForBigFileUploadLog, BigFileSize)

	// Big files (> 16 MiB) are uploaded sequentially in chunks of size
	// 16 MiB each and each chunk's successful upload generates a log.
	gcsWriteChunkSize := 16 * MiB
	numTotalChunksToBeCompleted := int(math.Floor(float64(BigFileSize) / float64(gcsWriteChunkSize)))
	var expectedSubstrings []string
	for numChunksCompletedSoFar := 1; numChunksCompletedSoFar <= numTotalChunksToBeCompleted; numChunksCompletedSoFar++ {
		expectedSubstrings = append(expectedSubstrings, fmt.Sprintf("%d bytes uploaded so far", numChunksCompletedSoFar*gcsWriteChunkSize))
	}

	logString := extractRelevantLogsFromLogFile(t, setup.LogFile(), &logFileOffset)
	operations.VerifyExpectedSubstrings(t, logString, expectedSubstrings)
}

func TestSmallFileUploadFileLog(t *testing.T) {
	uploadFile(t, DirForBigFileUploadLog, SmallFileSize)

	// The file being uploaded is too small (<16 MB) for progress logs
	// to be printed.
	unexpectedLogSubstrings := []string{"bytes uploaded so far"}
	logString := extractRelevantLogsFromLogFile(t, setup.LogFile(), &logFileOffset)
	operations.VerifyUnexpectedSubstrings(t, logString, unexpectedLogSubstrings)
}

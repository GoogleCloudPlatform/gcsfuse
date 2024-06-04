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

package log_content

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	SmallFileSize            = MiB
	DirForSmallFileUploadLog = "dirForSmallFileUploadLog"
)

func TestSmallFileUploadFileLog(t *testing.T) {
	testDir := path.Join(setup.MntDir(), DirForSmallFileUploadLog)
	err := os.Mkdir(testDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory:%v", err)
	}
	// Clean up.
	defer operations.RemoveDir(testDir)

	filePath := path.Join(testDir, FileName)

	// Sequentially write the data in file.
	err = operations.WriteFileSequentially(filePath, SmallFileSize, SmallFileSize)
	if err != nil {
		t.Fatalf("Error in writing file: %v", err)
	}

	logFile := setup.LogFile()
	if logFile == "" {
		t.Fatalf("No log-file found")
	}

	// Read the entire bytes file at once. This can be optimized by reading
	// a bunch of lines at once, then eliminating the found
	// expected substrings one by one.
	bytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Failed in reading logfile %q: %v", logFile, err)
	}
	completeLogString := string(bytes)
	logString := completeLogString[logFileOffset:]

	unexpectedLogSubstring := "bytes uploaded so far"
	if strings.Contains(logString, unexpectedLogSubstring) {
		t.Errorf("Logfile %s contains unexpected substring: %q", logFile, unexpectedLogSubstring)
	}

	logFileOffset = len(completeLogString)
}

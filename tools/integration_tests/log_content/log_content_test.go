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
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	MiB      = 1024 * 1024
	FileName = "fileName.txt"
)

var (
	logFileOffset int
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfMountedDirectoryIsSetOrTestBucketIsNotSet()

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	// No flags to be set. Only debugs log are to be enabled,
	// which are enabled by default by static_mounting.RunTests .
	flagsSet := [][]string{{}}

	logFile, err := os.CreateTemp(setup.TestDir(), "log_content_test-*.log")
	if err != nil || logFile == nil {
		log.Fatalf("Failed to create temp-file for logging: %v", err)
	}
	defer logFile.Close()
	setup.SetLogFile(logFile.Name())

	successCode := 0
	if successCode == 0 {
		successCode = static_mounting.RunTests(flagsSet, m)
	}

	os.Exit(successCode)
}

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

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// This test supports the scenario where only a testBucket has been passed.
	// If a user passes a mountedDirectory, then the
	// test cannot ensure that logs are generated for it,
	// and thus does not support that scenario.
	setup.ExitWithFailureIfMountedDirectoryIsSetOrTestBucketIsNotSet()

	// Enable tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	// Set up a log file.
	logFile, err := os.CreateTemp(setup.TestDir(), "log_content_test-*.log")
	if err != nil || logFile == nil {
		log.Fatalf("Failed to create temp-file for logging: %v", err)
	}
	defer logFile.Close()
	setup.SetLogFile(logFile.Name())

	// No explicit flags need to be set. Only debugs log are to be enabled,
	// which are enabled by default by static_mounting.RunTests
	// and by the above call to set log-file.
	flagsSet := [][]string{{}}

	successCode := 0
	if successCode == 0 {
		successCode = static_mounting.RunTests(flagsSet, m)
	}

	os.Exit(successCode)
}

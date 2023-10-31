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

// Provides integration tests for log rotation of gcsfuse logs.

package log_rotation

import (
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirName  = "LogRotationTest"
	testFileName = "file1.txt"
	logFileName  = "log.txt"
	MiB          = 1024 * 1024
	filePerms    = 0644
	logDirName   = "gcsfuse_integration_test_logs"
)

var (
	logFilePath, logDirPath string
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getMountConfigForLogRotation(maxFileSizeMB, fileCount int, compress bool,
	logFilePath string) config.MountConfig {
	mountConfig := config.MountConfig{
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
			FilePath: logFilePath,
			LogRotateConfig: config.LogRotateConfig{
				MaxFileSizeMB: maxFileSizeMB,
				FileCount:     fileCount,
				Compress:      compress,
			},
		},
	}
	return mountConfig
}

func generateRandomData(t *testing.T, sizeInBytes int64) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]byte, sizeInBytes)
	_, err := r.Read(data)
	if err != nil {
		t.Errorf("error while generating random data to write to file.")
	}
	return data
}

func writeRandomDataToFileTillLogRotation(t *testing.T) {
	testDirPath := setup.SetupTestDirectory(testDirName)

	// Generate random data to write to file.
	randomData := generateRandomData(t, MiB)
	// Setup file to write to.
	filePath := path.Join(testDirPath, testFileName)
	fh := operations.CreateFile(filePath, filePerms, t)

	// Keep performing operations in mounted directory until log file is rotated.
	var lastLogFileSize int64 = 0
	for {
		operations.WriteAt(string(randomData), 0, fh, t)
		operations.ReadDirectory(testDirPath, t)
		err := fh.Sync()
		if err != nil {
			t.Errorf("sync failed: %v", err)
		}
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

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestLogRotation(t *testing.T) {
	// Perform log rotation 2 times.
	for i := 0; i < 2; i++ {
		writeRandomDataToFileTillLogRotation(t)
	}

	// Validate that in the end we have one compressed rotated log file and one
	// active log file.
	dirEntries := operations.ReadDirectory(logDirPath, t)
	if len(dirEntries) != 2 {
		t.Errorf("Expected log files in dirEntries folder: 2, got: %d", len(dirEntries))
	}
	countOfRotatedCompressedFiles := 0
	countOfLogFile := 0
	for i := 0; i < 2; i++ {
		if dirEntries[i].Name() == logFileName {
			countOfLogFile++
		}
		if strings.Contains(dirEntries[i].Name(), "txt.gz") {
			countOfRotatedCompressedFiles++
		}
	}
	if countOfLogFile != 1 {
		t.Errorf("expected countOfLogFile: 1, got: %d", countOfLogFile)
	}
	if countOfRotatedCompressedFiles != 1 {
		t.Errorf("expected countOfRotatedCompressedFiles: 1, got: %d",
			countOfRotatedCompressedFiles)
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	logDirPath = path.Join("/tmp", logDirName)
	logFilePath = path.Join(logDirPath, logFileName)
	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	logDirPath = setup.SetUpLogDir(logDirName)
	logFilePath = path.Join(logDirPath, logFileName)

	configFile := setup.YAMLConfigFile(
		getMountConfigForLogRotation(1, 2, true, logFilePath),
	)

	// Set up flags to run tests on.
	// Not setting config file explicitly with 'create-empty-file: false' as it is default.
	flags := [][]string{
		{"--config-file=" + configFile},
	}

	successCode := static_mounting.RunTests(flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}

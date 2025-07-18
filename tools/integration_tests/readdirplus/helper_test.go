// Copyright 2025 Google LLC
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

// Provides integration tests for long listing directory with Readdirplus
package readdirplus

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedEntries is a helper struct that stores list of attributes to be validated.
type expectedEntries struct {
	name  string
	isDir bool
	mode  os.FileMode
}

// createDirectoryStructure creates a directory structure for testing.
// testBucket/target_dir/                                                       -- Dir
// testBucket/target_dir/file		                                            -- File
// testBucket/target_dir/emptySubDirectory                                      -- Dir
// testBucket/target_dir/subDirectory                                           -- Dir
// testBucket/target_dir/subDirectory/file1                                     -- File
func createDirectoryStructure(t *testing.T) []expectedEntries {
	t.Helper()

	targetDir := path.Join(testDirPath, targetDirName)
	operations.CreateDirectory(targetDir, t)
	// Create a file in the target directory.
	f1 := operations.CreateFile(path.Join(targetDir, "file"), setup.FilePermission_0600, t)
	operations.CloseFileShouldNotThrowError(t, f1)
	// Create an empty subdirectory
	operations.CreateDirectory(path.Join(targetDir, "emptySubDirectory"), t)
	// Create a subdirectory with file
	operations.CreateDirectoryWithNFiles(1, path.Join(targetDir, "subDirectory"), "file", t)
	expectedEntries := []expectedEntries{
		{name: "emptySubDirectory", isDir: true, mode: os.ModeDir | 0755},
		{name: "file", isDir: false, mode: 0644},
		{name: "subDirectory", isDir: true, mode: os.ModeDir | 0755},
	}
	return expectedEntries
}

// validateEntries validates the entries against the expected entries.
func validateEntries(entries []os.FileInfo, expectedEntries []expectedEntries, t *testing.T) {
	t.Helper()
	// Verify the entries.
	assert.Equal(t, len(expectedEntries), len(entries), "Number of entries mismatch")
	for i, expected := range expectedEntries {
		entry := entries[i]
		assert.Equal(t, expected.name, entry.Name(), "Name mismatch for entry %d", i)
		assert.Equal(t, expected.isDir, entry.IsDir(), "IsDir mismatch for entry %s", entry.Name())
		assert.Equal(t, expected.mode, entry.Mode(), "Mode mismatch for entry %s", entry.Name())
	}
}

func loadLogLines(reader io.Reader) ([]string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// validateLogsForReaddirplus checks that ReadDirPlus was called and ReadDir was not.
// It also checks that LookUpInode is not called when dentry cache is enabled.
func validateLogsForReaddirplus(t *testing.T, logFile string, dentryCacheEnabled bool, startTime, endTime time.Time) {
	t.Helper()

	logForReadDirPlus := "ReadDirPlus ("
	logForReadDir := "ReadDir ("
	logForLookUpInode := "LookUpInode ("

	file, err := os.Open(logFile)
	require.NoError(t, err, "Failed to open log file")
	defer file.Close()

	logLines, err := loadLogLines(file)
	require.NoError(t, err, "Failed to read log file")

	foundReadDirPlus := false
	foundReadDir := false
	foundLookUpInode := false
	for _, line := range logLines {
		logEntry, err := read_logs.ParseJsonLogLineIntoLogEntryStruct(line) // Assuming read_logs can parse general log lines too or a more generic parser is available.
		// If parsing fails, it might be a non-JSON line or a different structured log.
		// For this specific message, we expect it to be in the "Message" field of a structured log.

		if err == nil && logEntry != nil {
			// Check if the log entry's timestamp is within the expected window.
			if (logEntry.Timestamp.After(startTime) || logEntry.Timestamp.Equal(startTime)) &&
				(logEntry.Timestamp.Before(endTime) || logEntry.Timestamp.Equal(endTime)) {
				if strings.Contains(logEntry.Message, logForReadDirPlus) {
					foundReadDirPlus = true
				}
				if strings.Contains(logEntry.Message, logForReadDir) {
					foundReadDir = true
				}
				if strings.Contains(logEntry.Message, logForLookUpInode) {
					foundLookUpInode = true
				}
			}
		}
	}

	require.True(t, foundReadDirPlus, "ReadDirPlus not called")
	require.False(t, foundReadDir, "ReadDir called unexpectedly")
	if dentryCacheEnabled {
		require.False(t, foundLookUpInode, "LookUpInode called unexpectedly")
	} else {
		require.True(t, foundLookUpInode, "LookUpInode not called")
	}
}

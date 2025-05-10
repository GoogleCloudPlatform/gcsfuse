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

package inactive_read_timeout

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/stretchr/testify/require"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
)

func setupFile(ctx context.Context, storageClient *storage.Client, fileName string, fileSize int, t *testing.T) string {
	t.Helper()
	client.SetupFileInTestDirectory(ctx, storageClient, kTestDirName, fileName, int64(fileSize), t)
	return path.Join(gTestDirPath, fileName)
}

func loadLogLines(reader io.Reader) ([]string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// validateInactiveReaderClosedLog checks if the "Closing reader for object ... due to inactivity"
// log message is present (or absent) for the given objectName, b/w the [startTime, endTime] interval.
// Also expects based on the shouldBePresent value.
func validateInactiveReaderClosedLog(t *testing.T, logFile, objectName string, shouldBePresent bool, startTime, endTime time.Time) {
	t.Helper()
	// Be specific about the object name in the expected message.
	expectedMsgSubstring := fmt.Sprintf("Closing reader for object %q due to inactivity.", objectName)

	file, err := os.Open(logFile)
	require.NoError(t, err, "Failed to open log file")
	defer file.Close()

	logLines, err := loadLogLines(file)
	require.NoError(t, err, "Failed to read log file")

	found := false
	for _, line := range logLines {
		logEntry, err := read_logs.ParseJsonLogLineIntoLogEntryStruct(line) // Assuming read_logs can parse general log lines too or a more generic parser is available.
		// If parsing fails, it might be a non-JSON line or a different structured log.
		// For this specific message, we expect it to be in the "Message" field of a structured log.

		if err == nil && logEntry != nil {
			// Check if the log entry's timestamp is within the expected window.
			if (logEntry.Timestamp.After(startTime) || logEntry.Timestamp.Equal(startTime)) &&
				(logEntry.Timestamp.Before(endTime) || logEntry.Timestamp.Equal(endTime)) {
				if strings.Contains(logEntry.Message, expectedMsgSubstring) {
					found = true
					break
				}
			}
		}
	}

	if shouldBePresent {
		require.True(t, found, "Expected log message substring '%s' not found between %v and %v", expectedMsgSubstring, startTime, endTime)
	} else {
		require.False(t, found, "Unexpected log message substring '%s' found between %v and %v", expectedMsgSubstring, startTime, endTime)
	}
}

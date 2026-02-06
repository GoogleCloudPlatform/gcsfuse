// Copyright 2024 Google LLC
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

package read_logs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

/*
parseJobLogsFromLogFile takes an io.Reader and returns structured map of Job ID
and Job logs sorted by timestamp.
*/
func parseJobLogsFromLogFile(reader io.Reader) (map[string]*Job, error) {
	// structuredLogs map stores is a mapping between job id and JobData.
	structuredLogs := make(map[string]*Job)

	lines, err := loadLogLines(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading log file: %v", err)
	}

	for _, line := range lines {
		if err := filterAndParseJobLogLine(line, structuredLogs); err != nil {
			return nil, fmt.Errorf("filterAndParseJobLogLine failed for %s: %v", line, err)
		}
	}

	return structuredLogs, nil
}

func filterAndParseJobLogLine(logLine string, structuredLogs map[string]*Job) error {

	jsonLog := make(map[string]any)
	if err := json.Unmarshal([]byte(logLine), &jsonLog); err != nil {
		return nil // Silently ignore the structuredLogs which are not in JSON format.
	}

	// Get timestamp from the jsonLog
	timestampSeconds := int64(jsonLog["timestamp"].(map[string]any)["seconds"].(float64))
	timestampNanos := int64(jsonLog["timestamp"].(map[string]any)["nanos"].(float64))
	// Normalize whitespace in the log message.
	logMessage := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(jsonLog["message"].(string), " "))
	// Parse the logs based on type.
	switch {
	case strings.Contains(logMessage, "downloaded till"):
		if err := parseJobFileLog(timestampSeconds, timestampNanos, logMessage, structuredLogs); err != nil {
			return fmt.Errorf("parseJobFileLog failed: %v", err)
		}
	case strings.Contains(logMessage, "downloaded range"):
		if err := parseChunkDownloadLog(timestampSeconds, timestampNanos, logMessage, structuredLogs); err != nil {
			return fmt.Errorf("parseChunkDownloadLog failed: %v", err)
		}
	}
	return nil
}

/*
GetJobLogsSortedByTimestamp is used in read cache logs parsing for functional tests.
This method takes gcsfuse logs file path (json format) as input and parses it
into array of structured job log entries sorted by timestamp.
*/
func GetJobLogsSortedByTimestamp(logFilePath string, t *testing.T) []*Job {
	// Open and parse log file.
	file, err := os.Open(logFilePath)
	if err != nil {
		t.Errorf("Failed to open log file")
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			t.Logf("error closing log file: %s", logFilePath)
		}
	}(file)
	logsMap, err := parseJobLogsFromLogFile(file)
	if err != nil {
		t.Errorf("Failed to parse logs %s correctly: %v", setup.LogFile(), err)
	}

	// Create array from structured logs map.
	structuredJobLogs := make([]*Job, len(logsMap))
	var i = 0
	for _, val := range logsMap {
		structuredJobLogs[i] = val
		i++
	}
	return structuredJobLogs
}

// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http:#www.apache.org/licenses/LICENSE-2.0
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
	"sort"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func filterAndParseLogLine(logLine string,
	structuredLogs map[int64]*StructuredReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	jsonLog := make(map[string]interface{})
	if err := json.Unmarshal([]byte(logLine), &jsonLog); err != nil {
		return nil // Silently ignore the structuredLogs which are not in JSON format.
	}

	// Get timestamp from the jsonLog
	timestampSeconds := int64(jsonLog["timestamp"].(map[string]interface{})["seconds"].(float64))
	timestampNanos := int64(jsonLog["timestamp"].(map[string]interface{})["nanos"].(float64))
	// Normalize whitespace in the log message.
	logMessage := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(jsonLog["message"].(string), " "))
	// Tokenize log message.
	tokenizedLogs := strings.Split(logMessage, " ")

	// Parse the logs based on type.
	switch {
	// TODO: change this to regex matching instead of strings.Contains.
	case strings.Contains(logMessage, "ReadFile"):
		if err := parseReadFileLog(timestampSeconds, timestampNanos, tokenizedLogs, structuredLogs); err != nil {
			return fmt.Errorf("parseReadFileLog failed: %v", err)
		}
	case strings.Contains(logMessage, "FileCache("):
		if err := parseFileCacheRequestLog(timestampSeconds, timestampNanos, tokenizedLogs, structuredLogs, opReverseMap); err != nil {
			return fmt.Errorf("parseFileCacheRequestLog failed: %v", err)
		}
	case strings.Contains(logMessage, "OK (isSeq") && !strings.Contains(logMessage, "fuse_debug"):
		if err := parseFileCacheResponseLog(tokenizedLogs, structuredLogs, opReverseMap); err != nil {
			return fmt.Errorf("parseFileCacheResponseLog failed: %v", err)
		}
	}
	return nil
}

/*
ParseReadLogsFromLogFile is originally written for read cache logs parsing for functional tests.
This method takes gcsfuse logs file path (json format) as input and parses it
into map of following structure:

	{
	  "25"(file handle): {
	    "handle": 25,
	    "StartTime": 1704444226937309952,
	    "ProcessID": 2270282,
	    "InodeID": 2,
	    "BucketName": "bucket_name",
	    "ObjectName": "object/name",
	    "Chunks": [
	      {
	        "StartTime": 1704444226937457664,
	        "StartOffset": 0,
	        "Size": 26214,
	        "CacheHit": false,
	        "IsSequential": true,
	        "OpID": "310f589d-20bf",
	        "ExecutionTime": "1.907320375s"
	      },
				...
	    ]
		},
		...
	}
*/
func ParseReadLogsFromLogFile(reader io.Reader) (map[int64]*StructuredReadLogEntry, error) {
	// structuredLogs map stores is a mapping between file handle and StructuredReadLogEntry.
	structuredLogs := make(map[int64]*StructuredReadLogEntry)
	opReverseMap := make(map[string]*handleAndChunkIndex)

	lines, err := loadLogLines(reader)
	if err != nil {
		fmt.Println("Error reading log file:", err)
		os.Exit(1)
	}

	for _, line := range lines {
		if err := filterAndParseLogLine(line, structuredLogs, opReverseMap); err != nil {
			return nil, fmt.Errorf("filterAndParseLogLine failed for %s: %v", line, err)
		}
	}

	return structuredLogs, nil
}

/*
GetStructuredLogsSortedByTimestamp is originally written for read cache logs parsing for functional tests.
This method takes gcsfuse logs file path (json format) as input and parses it
into array of structured log entries sorted by timestamp.
*/
func GetStructuredLogsSortedByTimestamp(logFilePath string, t *testing.T) []*StructuredReadLogEntry {
	// Open and parse log file.
	file, err := os.Open(logFilePath)
	if err != nil {
		t.Errorf("Failed to open log file")
	}
	logsMap, err := ParseReadLogsFromLogFile(file)
	if err != nil {
		t.Errorf("Failed to parse logs %s correctly: %v", setup.LogFile(), err)
	}

	// Create array from structured logs map.
	structuredReadLogs := make([]*StructuredReadLogEntry, len(logsMap))
	var i = 0
	for _, val := range logsMap {
		structuredReadLogs[i] = val
		i++
	}

	// Sort the logs based on start time stamp.
	sort.Slice(structuredReadLogs, func(i, j int) bool {
		if structuredReadLogs[i].StartTimeSeconds < structuredReadLogs[j].StartTimeSeconds {
			return true
		} else if structuredReadLogs[i].StartTimeSeconds == structuredReadLogs[j].StartTimeSeconds &&
			structuredReadLogs[i].StartTimeNanos < structuredReadLogs[j].StartTimeNanos {
			return true
		}
		return false
	})

	return structuredReadLogs
}

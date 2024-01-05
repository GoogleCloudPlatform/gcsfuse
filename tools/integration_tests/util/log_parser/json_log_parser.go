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

/*
Originally written for read cache logs parsing.
This script takes gcsfuse logs in json format and parse them into the following
structure:
{
	"2": {
	"handle": "2",
	"start_time": 1704345315,
	"process_id": "153607",
	"inode_id": "3",
	"object_name": "bucket_name/object_name",
	"chunks": [
							{
							"start_time": 1704345315,
							"start_offset": "0",
							"size": "1048576",
							"cache_hit": "false,",
							"is_sequential": "true,"
							},
							...
					]
	},
	...
}
*/

package log_parser

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func readFileLineByLine(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func filterAndParseLogLine(logLine string, structuredLogs *map[int]StructuredLogEntry, opReverseMap *map[string]HandleAndChunkIndex) error {
	jsonLog := make(map[string]interface{})
	if err := json.Unmarshal([]byte(logLine), &jsonLog); err != nil {
		return nil // Silently ignore the structuredLogs which are not in JSON format.
	}

	// Get timestamp from the jsonLog
	timestamp := int64(jsonLog["time"].(map[string]interface{})["timestampNanos"].(float64))
	// Normalize whitespace in the log message.
	logMessage := strings.TrimSpace(regexp.MustCompile("\\s+").ReplaceAllString(jsonLog["msg"].(string), " "))
	// Tokenize log message.
	tokenizedLogs := strings.Split(logMessage, " ")

	// Parse the logs based on type.
	switch {
	case strings.Contains(logMessage, "ReadFile"):
		if err := parseReadFileLog(timestamp, tokenizedLogs, structuredLogs); err != nil {
			return fmt.Errorf("parseReadFileLog failed: %v", err)
		}
	case strings.Contains(logMessage, "FileCache"):
		if err := parseFileCacheLog(timestamp, tokenizedLogs, structuredLogs, opReverseMap); err != nil {
			return fmt.Errorf("parseReadFileLog failed: %v", err)
		}
	case strings.Contains(logMessage, "OK (isSeq") && !strings.Contains(logMessage, "fuse_debug"):
		parseFileCacheResponseLogs(tokenizedLogs, structuredLogs, opReverseMap)
	}
	return nil
}

func ParseLogFile(logFilePath string) (map[int]StructuredLogEntry, error) {
	// structuredLogs map stores is a mapping between file handle and StructuredLogEntry.
	structuredLogs := make(map[int]StructuredLogEntry)
	opReverseMap := make(map[string]HandleAndChunkIndex)

	lines, err := readFileLineByLine(logFilePath)
	if err != nil {
		fmt.Println("Error reading log file:", err)
		os.Exit(1)
	}

	for _, line := range lines {
		if err := filterAndParseLogLine(line, &structuredLogs, &opReverseMap); err != nil {
			return nil, fmt.Errorf("filterAndParseLogLine failed for %s: %v", line, err)
		}
	}

	return structuredLogs, nil
}

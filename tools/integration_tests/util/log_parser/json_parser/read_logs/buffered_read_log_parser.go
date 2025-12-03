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

package read_logs

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var readFileRegex = regexp.MustCompile(`fuse_debug: Op (0x[0-9a-fA-F]+)\s+connection\.go:\d+\] <- ReadFile \(inode (\d+), PID (\d+), handle (\d+), offset (\d+), (\d+) bytes\)`)
var readAtReqRegex = regexp.MustCompile(`([a-f0-9-]+) <- ReadAt\(([^:]+):/([^,]+), (\d+), (\d+), (\d+), (\d+)\)`)
var readAtSimpleRespRegex = regexp.MustCompile(`([a-f0-9-]+) -> ReadAt\(\): Ok\(([0-9.]+(?:s|ms|µs))\)`)
var fallbackFromHandleRegex = regexp.MustCompile(`Fallback to another reader for object "[^"]+", handle (\d+)\.(?: Random seek count (\d+) exceeded threshold \d+.*)?`)
var restartFromHandleRegex = regexp.MustCompile(`Restarting buffered reader.*handle (\d+)`)

// ParseBufferedReadLogsFromLogReader parses buffered read logs from an io.Reader and
// returns a map of BufferedReadLogEntry keyed by file handle.
// BufferedReadLogEntry contains the common read log information and a slice of
// BufferedReadChunkData representing the chunk read from buffered reader.
// Example:
//
//	{
//	  "25": {
//	    "CommonReadLog": {
//	      "Handle": 25,
//	      "StartTimeSeconds": 1704444226,
//	      "StartTimeNanos": 937309952,
//	      "ProcessID": 2270282,
//	      "InodeID": 2,
//	      "BucketName": "bucket_name",
//	      "ObjectName": "object/name"
//	    },
//	    "Chunks": [
//	      {
//	        "StartTimeSeconds": 1704444226,
//	        "StartTimeNanos": 937457664,
//	        "RequestID": "310f589d-20bf",
//	        "Offset": 0,
//	        "Size": 26214,
//	        "BlockIndex": 0,
//	        "ExecutionTime": "1.907320375s"
//	      },
//	      ...
//	    ]
//	  },
//	  ...
//	}
func ParseBufferedReadLogsFromLogReader(reader io.Reader) (map[int64]*BufferedReadLogEntry, error) {
	// file-handle to BufferedReadLogEntry map
	bufferedReadLogsMap := make(map[int64]*BufferedReadLogEntry)

	// opReverseMap is used to map request ID to handle and chunk index.
	opReverseMap := make(map[string]*handleAndChunkIndex)

	lines, err := loadLogLines(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to load log lines: %v", err)
	}

	for _, line := range lines {
		if err := filterAndParseLogLineForBufferedRead(line, bufferedReadLogsMap, opReverseMap); err != nil {
			return nil, fmt.Errorf("filterAndParseLogLineForBufferedRead failed for %s: %v", line, err)
		}
	}

	// Filter out entries that have no chunks, as they represent file handles
	// that were opened but never read from using the buffered reader.
	filteredLogsMap := make(map[int64]*BufferedReadLogEntry)
	for handle, entry := range bufferedReadLogsMap {
		if len(entry.Chunks) > 0 {
			filteredLogsMap[handle] = entry
		}
	}
	return filteredLogsMap, nil
}

// filterAndParseLogLineForBufferedRead filters and parses a log line for buffered read logs.
func filterAndParseLogLineForBufferedRead(
	logLine string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	jsonLog := make(map[string]any)
	if err := json.Unmarshal([]byte(logLine), &jsonLog); err != nil {
		return nil // Silently ignore the logs which are not in JSON format.
	}

	if _, ok := jsonLog["timestamp"]; !ok {
		return fmt.Errorf("filterAndParseLogLineForBufferedRead: log line does not contain timestamp: %s", logLine)
	}
	timestampSeconds := int64(jsonLog["timestamp"].(map[string]any)["seconds"].(float64))
	timestampNanos := int64(jsonLog["timestamp"].(map[string]any)["nanos"].(float64))

	// Log message is expected to be in the "message" field.
	if _, ok := jsonLog["message"]; !ok {
		return fmt.Errorf("filterAndParseLogLineForBufferedRead: log line does not contain message: %s", logLine)
	}
	logMessage := jsonLog["message"].(string)

	// Parse the logs based on type.
	switch {
	case strings.Contains(logMessage, "<- ReadFile"):
		if err := parseReadFileLogsUsingRegex(timestampSeconds, timestampNanos, logMessage, bufferedReadLogsMap); err != nil {
			return fmt.Errorf("parseReadFileLog failed: %v", err)
		}
	case strings.Contains(logMessage, "<- ReadAt("):
		if err := parseReadAtRequestLog(timestampSeconds, timestampNanos, logMessage, bufferedReadLogsMap, opReverseMap); err != nil {
			return fmt.Errorf("parseReadAtLog failed: %v", err)
		}
	case strings.Contains(logMessage, "-> ReadAt("):
		if err := parseReadAtResponseLog(logMessage, bufferedReadLogsMap, opReverseMap); err != nil {
			return fmt.Errorf("parseReadAtResponseLog failed: %v", err)
		}
	case strings.Contains(logMessage, "Fallback to another reader for object"):
		if err := parseFallbackLogFromHandle(logMessage, bufferedReadLogsMap); err != nil {
			return fmt.Errorf("parseFallbackLogFromHandle failed: %v", err)
		}
	case strings.Contains(logMessage, "Restarting buffered reader"):
		if err := parseRestartLogFromHandle(logMessage, bufferedReadLogsMap); err != nil {
			return fmt.Errorf("parseRestartLogFromHandle failed: %v", err)
		}
	}
	return nil
}

func parseFallbackLogFromHandle(
	logMessage string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry) error {

	matches := fallbackFromHandleRegex.FindStringSubmatch(logMessage)
	if len(matches) < 2 {
		// Not a fallback log we are interested in, might be from a different reader.
		return nil
	}

	handleID, err := parseToInt64(matches[1])
	if err != nil {
		return fmt.Errorf("invalid handle ID in fallback log: %w", err)
	}

	logEntry, ok := bufferedReadLogsMap[handleID]
	if !ok {
		return fmt.Errorf("log entry for handle %d not found for fallback log", handleID)
	}

	logEntry.Fallback = true
	if len(matches) > 2 && matches[2] != "" {
		randomSeekCount, err := parseToInt64(matches[2])
		if err != nil {
			return fmt.Errorf("invalid random seek count in fallback log: %v", err)
		}
		logEntry.RandomSeekCount = randomSeekCount
	}
	return nil
}

func parseRestartLogFromHandle(
	logMessage string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry) error {

	matches := restartFromHandleRegex.FindStringSubmatch(logMessage)
	if len(matches) < 2 {
		return nil
	}

	handleID, err := parseToInt64(matches[1])
	if err != nil {
		return fmt.Errorf("invalid handle ID in restart log: %w", err)
	}

	if logEntry, ok := bufferedReadLogsMap[handleID]; ok {
		logEntry.Restarted = true
	}
	return nil
}

// parseReadFileLogsUsingRegex parses the ReadFile log using regex and updates the bufferedReadLogsMap map.
// It extracts the handle, PID, inode ID from the log message.
func parseReadFileLogsUsingRegex(
	startTimeStampSec, startTimeStampNanos int64,
	logMessage string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry) error {

	matches := readFileRegex.FindStringSubmatch(logMessage)
	if len(matches) != 7 {
		return fmt.Errorf("invalid ReadFile log format: %s", logMessage)
	}

	handle, err := parseToInt64(matches[4])
	if err != nil {
		return fmt.Errorf("invalid handle: %v", err)
	}
	pid, err := parseToInt64(matches[3])
	if err != nil {
		return fmt.Errorf("invalid process ID: %v", err)
	}
	inodeID, err := parseToInt64(matches[2])
	if err != nil {
		return fmt.Errorf("invalid inode ID: %v", err)
	}

	// ReadFile log entries can come multiple times.
	// Check if log entry exists in the map for file handle.
	// If log entry doesn't exist, add it to the map.
	_, ok := bufferedReadLogsMap[handle]
	if !ok {
		bufferedReadLogsMap[handle] = &BufferedReadLogEntry{
			CommonReadLog: CommonReadLog{
				Handle:           handle,
				StartTimeSeconds: startTimeStampSec,
				StartTimeNanos:   startTimeStampNanos,
				ProcessID:        pid,
				InodeID:          inodeID,
			},
			Chunks: []BufferedReadChunkData{},
		}
	}
	return nil
}

// parseReadAtRequestLog parses the ReadAt request log and updates the bufferedReadLogsMap map.
// It extracts the request ID, offset, size, and block index from the log message.
// It also populates the bucket and object name if they are not already set in the BufferedReadLogEntry.
func parseReadAtRequestLog(
	startTimeStampSec, startTimeStampNanos int64,
	logMessage string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	matches := readAtReqRegex.FindStringSubmatch(logMessage)
	if len(matches) != 8 {
		return fmt.Errorf("invalid ReadAt log format: %s", logMessage)
	}

	handle, err := parseToInt64(matches[4]) // "1072693248"
	if err != nil {
		return fmt.Errorf("invalid handle: %v", err)
	}

	logEntry, ok := bufferedReadLogsMap[handle]
	if !ok || logEntry == nil {
		return fmt.Errorf("BufferedReadLogEntry for handle %d not found", handle)
	}

	if logEntry.BucketName == "" || logEntry.ObjectName == "" {
		logEntry.BucketName = matches[2] // "bucket_name"
		logEntry.ObjectName = matches[3] // "object/name"
	}

	requestID := matches[1] // "37623d67-b6ee"

	offset, err := parseToInt64(matches[5]) // "0"
	if err != nil {
		return fmt.Errorf("invalid offset: %v", err)
	}
	size, err := parseToInt64(matches[6]) // "1048576"
	if err != nil {
		return fmt.Errorf("invalid size: %v", err)
	}
	blockIndex, err := parseToInt64(matches[7]) // "63"
	if err != nil {
		return fmt.Errorf("invalid block index: %v", err)
	}

	chunkData := BufferedReadChunkData{
		StartTimeSeconds: startTimeStampSec,
		StartTimeNanos:   startTimeStampNanos,
		RequestID:        requestID,
		Offset:           offset,
		Size:             size,
		BlockIndex:       blockIndex,
		ExecutionTime:    "", // Execution time will be filled in the response log.
	}
	logEntry.Chunks = append(logEntry.Chunks, chunkData)
	opReverseMap[requestID] = &handleAndChunkIndex{handle: handle, chunkIndex: len(logEntry.Chunks) - 1}
	return nil
}

// parseReadAtResponseLog parses the ReadAt response log and updates the bufferedReadLogsMap map.
// It extracts the request ID and execution time from the log message.
// It updates the corresponding chunk in the bufferedReadLogsMap map with the execution time.
// The request ID is looked up in the opReverseMap to find the corresponding handle and chunk index.
func parseReadAtResponseLog(
	logMessage string,
	bufferedReadLogsMap map[int64]*BufferedReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	matches := readAtSimpleRespRegex.FindStringSubmatch(logMessage)
	if len(matches) != 3 {
		return fmt.Errorf("invalid simple ReadAt response log format: %s", logMessage)
	}

	requestID := matches[1]     // "d88d347c-1b8c"
	executionTime := matches[2] // "179.94µs"

	// Look up the request in the reverse map
	handleAndChunk, exists := opReverseMap[requestID]
	if !exists {
		return fmt.Errorf("request ID %s not found in reverse map", requestID)
	}

	// Update the execution time in the corresponding chunk
	logEntry := bufferedReadLogsMap[handleAndChunk.handle]
	if logEntry != nil && handleAndChunk.chunkIndex < len(logEntry.Chunks) {
		logEntry.Chunks[handleAndChunk.chunkIndex].ExecutionTime = executionTime
	}

	return nil
}

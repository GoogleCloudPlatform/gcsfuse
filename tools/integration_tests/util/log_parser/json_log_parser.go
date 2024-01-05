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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// StructuredLogEntry stores the structured format to be created from logs.
type StructuredLogEntry struct {
	Handle     int
	StartTime  int64
	ProcessID  string
	InodeID    string
	ObjectName string
	Chunks     []ChunkData
}

// ChunkData stores the format of chunk to be stored StructuredLogEntry.
type ChunkData struct {
	StartTime    int64
	StartOffset  string
	Size         string
	CacheHit     string
	IsSequential string
}

func readFileLineByLine(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func filterAndParseLogLine(logLine string, logs *map[int]StructuredLogEntry) error {
	jsonLog := make(map[string]interface{})
	if err := json.Unmarshal([]byte(logLine), &jsonLog); err != nil {
		return fmt.Errorf("could not unmarshal json log entry %s: %v", logLine, err)
	}

	// Normalize whitespace in the log message.
	logMessage := strings.TrimSpace(regexp.MustCompile("\\s+").ReplaceAllString(jsonLog["msg"].(string), " "))
	// Tokenize and parse
	//tokenizedLogs := strings.Split(logMessage, " ")

	switch {
	case strings.Contains(logMessage, "ReadFile"):
		//parseReadFileLog(jsonLog, tokenizedLogs, logs)
	case strings.Contains(logMessage, "FileCache"):
		// do something
	case strings.Contains(logMessage, "OK") && !strings.Contains(logMessage, "fuse_debug"):
		//parseCacheResponseLogs(jsonLog, tokenizedLogs, logs)
	}
	return nil
}

//func parseCacheLog(jsonLog map[string]interface{}, tokenizedLogs []string, logs *map[int]StructuredLogEntry) error {
//	startTimestamp := int64(jsonLog["time"].(map[string]interface{})["timestampNanos"].(float64))
//	handle, err := strconv.Atoi(tokenizedLogs[13][:len(tokenizedLogs[13])-1])
//	if err != nil {
//		return fmt.Errorf("could not parse handle to int  %s: %v", tokenizedLogs[13][:len(tokenizedLogs[13])-1], err)
//	}
//
//	chunkData := ChunkData{
//		StartTime:    startTimestamp,
//		StartOffset:  tokenizedLogs[15][:len(tokenizedLogs[15])-1],
//		Size:         tokenizedLogs[17][:len(tokenizedLogs[17])-1],
//		CacheHit:     tokenizedLogs[7][:len(tokenizedLogs[7])-1],
//		IsSequential: tokenizedLogs[5][:len(tokenizedLogs[5])-1],
//	}
//
//	logEntry := (*logs)[handle]
//	if logEntry.Handle == 0 {
//		logEntry = StructuredLogEntry{
//			Handle:     handle,
//			StartTime:  startTimestamp,
//			ProcessID:  tokenizedLogs[9][:len(tokenizedLogs[9])-1],
//			InodeID:    tokenizedLogs[11][:len(tokenizedLogs[11])-1],
//			ObjectName: tokenizedLogs[19][:len(tokenizedLogs[19])-2],
//			Chunks:     []ChunkData{},
//		}
//		(*logs)[handle] = logEntry
//	}
//	logEntry.Chunks = append(logEntry.Chunks, chunkData)
//	return nil
//}
//
//func parseCacheLog(jsonLog map[string]interface{}, tokenizedLogs []string, logs *map[int]StructuredLogEntry) error {
//	startTimestamp := int64(jsonLog["time"].(map[string]interface{})["timestampNanos"].(float64))
//	handle, err := strconv.Atoi(tokenizedLogs[13][:len(tokenizedLogs[13])-1])
//	if err != nil {
//		return fmt.Errorf("could not parse handle to int  %s: %v", tokenizedLogs[13][:len(tokenizedLogs[13])-1], err)
//	}
//
//	chunkData := ChunkData{
//		StartTime:    startTimestamp,
//		StartOffset:  tokenizedLogs[15][:len(tokenizedLogs[15])-1],
//		Size:         tokenizedLogs[17][:len(tokenizedLogs[17])-1],
//		CacheHit:     tokenizedLogs[7][:len(tokenizedLogs[7])-1],
//		IsSequential: tokenizedLogs[5][:len(tokenizedLogs[5])-1],
//	}
//
//	logEntry := (*logs)[handle]
//	if logEntry.Handle == 0 {
//		logEntry = StructuredLogEntry{
//			Handle:     handle,
//			StartTime:  startTimestamp,
//			ProcessID:  tokenizedLogs[9][:len(tokenizedLogs[9])-1],
//			InodeID:    tokenizedLogs[11][:len(tokenizedLogs[11])-1],
//			ObjectName: tokenizedLogs[19][:len(tokenizedLogs[19])-2],
//			Chunks:     []ChunkData{},
//		}
//		(*logs)[handle] = logEntry
//	}
//	logEntry.Chunks = append(logEntry.Chunks, chunkData)
//	return nil
//}

func ParseLogFile(logFilePath string) (map[int]StructuredLogEntry, error) {
	// structuredLogs map stores is a mapping between file handle and StructuredLogEntry.
	structuredLogs := make(map[int]StructuredLogEntry)

	lines, err := readFileLineByLine(logFilePath)
	if err != nil {
		fmt.Println("Error reading log file:", err)
		os.Exit(1)
	}

	for _, line := range lines {
		if err := filterAndParseLogLine(line, &structuredLogs); err != nil {
			return nil, fmt.Errorf("filterAndParseLogLine failed: %v", err)
		}
	}

	return structuredLogs, nil
}

func main() {
	mapstruct, err := ParseLogFile("~/Documents/log.json")
	if err != nil {
		fmt.Println(err)
	}

	jsonObject, _ := json.MarshalIndent(mapstruct, "", "  ")
	fmt.Println(jsonObject)
}

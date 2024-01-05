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
	"strconv"
	"strings"
)

// StructuredLogEntry stores the structured format to be created from logs.
type StructuredLogEntry struct {
	Handle     int
	StartTime  int64
	ProcessID  string
	InodeID    string
	BucketName string
	ObjectName string
	Chunks     []ChunkData
}

// ChunkData stores the format of chunk to be stored StructuredLogEntry.
type ChunkData struct {
	StartTime     int64
	StartOffset   string
	Size          string
	CacheHit      string
	IsSequential  string
	OpID          string
	ExecutionTime string
}

type HandleAndChunkIndex struct {
	Handle     int
	ChunkIndex int
}

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

func parseReadFileLog(startTimeStamp int64, tokenizedLogs []string, logs *map[int]StructuredLogEntry) error {
	handle, err := strconv.Atoi(tokenizedLogs[11][:len(tokenizedLogs[11])-1])
	if err != nil {
		return fmt.Errorf("could not parse handle%s to int: %v", tokenizedLogs[11][:len(tokenizedLogs[11])-1], err)
	}

	logEntry, ok := (*logs)[handle]
	if !ok {
		logEntry = StructuredLogEntry{
			Handle:    handle,
			StartTime: startTimeStamp,
			ProcessID: tokenizedLogs[9][:len(tokenizedLogs[9])-1],
			InodeID:   tokenizedLogs[7][:len(tokenizedLogs[7])-1],
			Chunks:    []ChunkData{},
		}
		(*logs)[handle] = logEntry
	}
	return nil
}

func parseFileCacheLog(startTimeStamp int64, tokenizedLogs []string,
	structuredLogs *map[int]StructuredLogEntry, opReverseMap *map[string]HandleAndChunkIndex) error {

	opID := tokenizedLogs[0]
	handle, err := strconv.Atoi(tokenizedLogs[8][:len(tokenizedLogs[8])-1])
	if err != nil {
		return fmt.Errorf("could not parse handle to int  %s: %v", tokenizedLogs[13][:len(tokenizedLogs[13])-1], err)
	}

	logEntry, ok := (*structuredLogs)[handle]
	if !ok {
		return fmt.Errorf("LogEntry for handle %d not found", handle)
	}
	if logEntry.ObjectName == "" && logEntry.BucketName == "" {
		bucketAndObjectName := tokenizedLogs[2][10 : len(tokenizedLogs[2])-1]
		logEntry.BucketName = strings.Split(bucketAndObjectName, ":")[0]
		logEntry.ObjectName = strings.Split(bucketAndObjectName, ":")[1][1:]
	}

	chunkData := ChunkData{
		StartTime:   startTimeStamp,
		StartOffset: tokenizedLogs[4][:len(tokenizedLogs[4])-1],
		Size:        tokenizedLogs[6][:len(tokenizedLogs[6])-1],
		OpID:        opID,
	}
	logEntry.Chunks = append(logEntry.Chunks, chunkData)
	(*structuredLogs)[handle] = logEntry
	(*opReverseMap)[opID] = HandleAndChunkIndex{Handle: handle, ChunkIndex: len(logEntry.Chunks) - 1}
	return nil
}

func parseFileCacheResponseLogs(tokenizedLogs []string, structuredLogs *map[int]StructuredLogEntry,
	opReverseMap *map[string]HandleAndChunkIndex) error {

	opID := tokenizedLogs[0]
	handle := (*opReverseMap)[opID].Handle
	chunkIndex := (*opReverseMap)[opID].ChunkIndex

	logEntry, ok := (*structuredLogs)[handle]
	if !ok {
		return fmt.Errorf("LogEntry for handle %d not found", handle)
	}
	logEntry.Chunks[chunkIndex].IsSequential = tokenizedLogs[4][:len(tokenizedLogs[4])-1]
	logEntry.Chunks[chunkIndex].CacheHit = tokenizedLogs[6][:len(tokenizedLogs[6])-1]
	logEntry.Chunks[chunkIndex].ExecutionTime = tokenizedLogs[7][1 : len(tokenizedLogs[7])-1]
	(*structuredLogs)[handle] = logEntry
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

func main() {
	mapstruct, err := ParseLogFile("/usr/local/google/home/ashmeen/Documents/log.json")
	if err != nil {
		fmt.Println(err)
	}

	jsonObject, _ := json.MarshalIndent(mapstruct, "", "  ")
	fmt.Println(string(jsonObject))
}

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

package log_parser

import (
	"fmt"
	"strconv"
	"strings"
)

func parseReadFileLog(startTimeStamp int64, tokenizedLogs []string, logs *map[int]StructuredLogEntry) error {
	handle, err := strconv.Atoi(tokenizedLogs[11][:len(tokenizedLogs[11])-1])
	if err != nil {
		return fmt.Errorf("could not parse handle%s to int: %v", tokenizedLogs[11][:len(tokenizedLogs[11])-1], err)
	}

	pid, _ := strconv.ParseInt(tokenizedLogs[9][:len(tokenizedLogs[9])-1], 10, 64)
	inodeID, _ := strconv.ParseInt(tokenizedLogs[7][:len(tokenizedLogs[7])-1], 10, 64)
	logEntry, ok := (*logs)[handle]
	if !ok {
		logEntry = StructuredLogEntry{
			Handle:    handle,
			StartTime: startTimeStamp,
			ProcessID: pid,
			InodeID:   inodeID,
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

	size, _ := strconv.ParseInt(tokenizedLogs[6][:len(tokenizedLogs[6])-1], 10, 64)
	startOffset, _ := strconv.ParseInt(tokenizedLogs[4][:len(tokenizedLogs[4])-1], 10, 64)
	chunkData := ChunkData{
		StartTime:   startTimeStamp,
		StartOffset: startOffset,
		Size:        size,
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
	logEntry.Chunks[chunkIndex].IsSequential, _ = strconv.ParseBool(tokenizedLogs[4][:len(tokenizedLogs[4])-1])
	logEntry.Chunks[chunkIndex].CacheHit, _ = strconv.ParseBool(tokenizedLogs[6][:len(tokenizedLogs[6])-1])
	logEntry.Chunks[chunkIndex].ExecutionTime = tokenizedLogs[7][1 : len(tokenizedLogs[7])-1]
	(*structuredLogs)[handle] = logEntry
	return nil
}

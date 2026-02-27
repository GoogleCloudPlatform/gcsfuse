// Copyright 2023 Google LLC
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
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func parseToInt64(token string) (int64, error) {
	res, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s to int64: %v", token, err)
	}
	return res, nil
}

func loadLogLines(reader io.Reader) ([]string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// parseReadFileLog parses a tokenized read file log message and adds details
// (handle, processId, inodeId) corresponding to the file handle in the
// structuredLogs map.
func parseReadFileLog(startTimeStampSec, startTimeStampNanos int64, logs []string,
	structuredLogs map[int64]*StructuredReadLogEntry) error {

	// Fetch file handle, process id and inode id from the logs.
	handle, err := parseToInt64(logs[11][:len(logs[11])-1]) //Remove trailing ","
	if err != nil {
		return fmt.Errorf("file handle: %v", err)
	}
	pid, err := parseToInt64(logs[9][:len(logs[9])-1]) //Remove trailing ","
	if err != nil {
		return fmt.Errorf("process id: %v", err)
	}
	inodeID, err := parseToInt64(logs[7][:len(logs[7])-1]) //Remove trailing ","
	if err != nil {
		return fmt.Errorf("inode id: %v", err)
	}

	// ReadFile log entries can come multiple times.
	// Check if log entry exists in the map for file handle.
	// If log entry doesn't exist, add it to the map.
	_, ok := structuredLogs[handle]
	if !ok {
		structuredLogs[handle] = &StructuredReadLogEntry{
			CommonReadLog: CommonReadLog{
				Handle:           handle,
				StartTimeSeconds: startTimeStampSec,
				StartTimeNanos:   startTimeStampNanos,
				ProcessID:        pid,
				InodeID:          inodeID,
			},
			Chunks: []ReadChunkData{},
		}
	}
	return nil
}

// parseFileCacheRequestLog parses a tokenized file cache log message and performs the
// following operations:
// 1. Populates object and bucket name in the structured log entry in case of
// first Filecache log for the read operation.
// 2. adds read operation chunk (opId, size, offset) corresponding to the file
// handle in the structuredLogs map.
// 3. Stores a reverse mapping of FileCache operation id to file handle and
// chunk index in a map, to be re-used while mapping file cache response logs to
// read chunk.
//
// Note: It is expected that parseFileCacheRequestLog will always come after ReadFile
// log. If corresponding ReadFile log is missing, this function throws an error.
func parseFileCacheRequestLog(startTimeStampSec, startTimeStampNanos int64, logs []string,
	structuredLogs map[int64]*StructuredReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	// Fetch file handle from the tokenized logs.
	handle, err := parseToInt64(logs[8][:len(logs[8])-1]) //Remove trailing ","
	if err != nil {
		return fmt.Errorf("file handle: %v", err)
	}
	// Fetch the log entry for the particular file handle from the structuredLogs map.
	logEntry, ok := structuredLogs[handle]
	if !ok {
		return fmt.Errorf("ReadFile LogEntry for handle %d not found", handle)
	}

	// For the first file cache log, log entry will not have object and bucket
	// name, so populate it.
	if logEntry.ObjectName == "" && logEntry.BucketName == "" {
		bucketAndObjectName := logs[2][10 : len(logs[2])-1] // Remove prefix "FileCache(" and suffix ","
		// bucketAndObjectName will be stored in format <bucketName>:/<objectName>
		logEntry.BucketName = strings.Split(bucketAndObjectName, ":")[0]
		logEntry.ObjectName = strings.Split(bucketAndObjectName, ":")[1][1:] // Remove prefix "/"
	}

	// Fetch operation id, read size and offset from the logs.
	opID := logs[0]
	size, err := parseToInt64(logs[6])
	if err != nil {
		return fmt.Errorf("size: %v", err)
	}
	startOffset, err := parseToInt64(logs[4][:len(logs[4])-1]) //Remove trailing ","
	if err != nil {
		return fmt.Errorf("start offset: %v", err)
	}

	// Create chunk data entry and append it to the filecache logs.
	chunkData := ReadChunkData{
		StartTimeSeconds: startTimeStampSec,
		StartTimeNanos:   startTimeStampNanos,
		StartOffset:      startOffset,
		Size:             size,
		OpID:             opID,
	}
	logEntry.Chunks = append(logEntry.Chunks, chunkData)

	// Store the file handle and chunk index in the operation reverse map.
	// This is required to map file cache response log back to log entry chunk.
	opReverseMap[opID] = &handleAndChunkIndex{handle: handle, chunkIndex: len(logEntry.Chunks) - 1}

	return nil
}

// parseFileCacheResponseLog parses a tokenized file cache response log message
// and performs the following operations:
// 1. Fetches the structured log entry's chunk using filecache operation ID leveraging
// opReverseMap (which stores a mapping of filecache operation id -> filehandle, chunk).
// 2. Fetches IsSequential, CacheHit and Execution time from the log and
// populates it in the chunk.
func parseFileCacheResponseLog(logs []string,
	structuredLogs map[int64]*StructuredReadLogEntry,
	opReverseMap map[string]*handleAndChunkIndex) error {

	opID := logs[0]
	handleAndChunkIndex, ok := opReverseMap[opID]
	if !ok {
		return fmt.Errorf("FileCache log entry not found for opID %s", opID)
	}
	handle := handleAndChunkIndex.handle
	chunkIndex := handleAndChunkIndex.chunkIndex

	// Fetch the log entry for the particular file handle from the structuredLogs map.
	logEntry, ok := structuredLogs[handle]
	if !ok {
		return fmt.Errorf("ReadFile LogEntry for handle %d not found", handle)
	}

	// Populate chunk IsSequential, CacheHit and Execution time
	chunk := &logEntry.Chunks[chunkIndex]
	chunk.IsSequential, _ = strconv.ParseBool(logs[4][:len(logs[4])-1]) //Remove trailing ","
	chunk.CacheHit, _ = strconv.ParseBool(logs[6][:len(logs[6])-1])     //Remove trailing ","
	chunk.ExecutionTime = logs[7][1 : len(logs[7])-1]                   //Remove prefix "(" and suffix ")"
	return nil
}

// parseJobFileLog parses a job file log message and adds details
// (bucket name, offset, timestamps) corresponding to the job id to the
// structuredLogs map.
func parseJobFileLog(startTimeStampSec, startTimeStampNanos int64, logsMessage string, structuredLogs map[string]*Job) error {

	// Fetch bucket name, object name and offset from the logs.
	pattern := `Job:(\w+) \(([\w./_-]+):/([\w./_-]+)\) downloaded till (\d+) offset.`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(logsMessage)

	var jobID, bucketName, objectName string
	var offset int64
	var err error
	if len(matches) == 5 { // 0th element is the full match, we want 4 captures
		jobID = matches[1]
		bucketName = matches[2]
		objectName = matches[3]
		offset, err = strconv.ParseInt(matches[4], 10, 64)
		if err != nil {
			return fmt.Errorf("error while parsing offset: %v", err)
		}
	} else {
		return fmt.Errorf("string did not match the expected pattern")
	}

	// Job log entries can come multiple times.
	// Check if log entry exists in the map for job id.
	// If job entry doesn't exist, add it to the map, else append an entry.
	jobEntry, ok := structuredLogs[jobID]
	if !ok {
		structuredLogs[jobID] = &Job{
			JobID:      jobID,
			ObjectName: objectName,
			BucketName: bucketName,
			JobEntries: []JobData{
				{
					StartTimeSeconds: startTimeStampSec,
					StartTimeNanos:   startTimeStampNanos,
					Offset:           offset,
				},
			},
		}
	} else {
		jobEntry.JobEntries = append(jobEntry.JobEntries, JobData{
			StartTimeSeconds: startTimeStampSec,
			StartTimeNanos:   startTimeStampNanos,
			Offset:           offset,
		})
	}
	return nil
}

// parseChunkDownloadLog parses a chunk download log message and adds details
// to the structuredLogs map.
func parseChunkDownloadLog(startTimeStampSec, startTimeStampNanos int64, logsMessage string, structuredLogs map[string]*Job) error {
	// Fetch bucket name, object name and offsets from the logs.
	// Example: Job:0xc000aa65b0 (bucket:/obj) downloaded range [0, 10), added 10 bytes to sparse file
	pattern := `Job:(\w+) \(([\w./_-]+):/([\w./_-]+)\) downloaded range \[(\d+), (\d+)\), added (\d+) bytes to sparse file`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(logsMessage)

	var jobID, bucketName, objectName string
	var startOffset, endOffset, bytesAdded int64
	var err error

	if len(matches) == 7 {
		jobID = matches[1]
		bucketName = matches[2]
		objectName = matches[3]
		startOffset, err = strconv.ParseInt(matches[4], 10, 64)
		if err != nil {
			return fmt.Errorf("error while parsing start offset: %v", err)
		}
		endOffset, err = strconv.ParseInt(matches[5], 10, 64)
		if err != nil {
			return fmt.Errorf("error while parsing end offset: %v", err)
		}
		bytesAdded, err = strconv.ParseInt(matches[6], 10, 64)
		if err != nil {
			return fmt.Errorf("error while parsing bytes added: %v", err)
		}
	} else {
		return fmt.Errorf("string did not match the expected pattern for sparse download")
	}

	entry := ChunkDownloadLogEntry{
		StartTimeSeconds: startTimeStampSec,
		StartTimeNanos:   startTimeStampNanos,
		StartOffset:      startOffset,
		EndOffset:        endOffset,
		BytesAdded:       bytesAdded,
	}

	jobEntry, ok := structuredLogs[jobID]
	if !ok {
		structuredLogs[jobID] = &Job{
			JobID:               jobID,
			ObjectName:          objectName,
			BucketName:          bucketName,
			ChunkCacheDownloads: []ChunkDownloadLogEntry{entry},
		}
	} else {
		jobEntry.ChunkCacheDownloads = append(jobEntry.ChunkCacheDownloads, entry)
	}
	return nil
}

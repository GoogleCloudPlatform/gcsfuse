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
	"time"
)

type CommonReadLog struct {
	Handle           int64
	StartTimeSeconds int64
	StartTimeNanos   int64
	ProcessID        int64
	InodeID          int64
	BucketName       string
	ObjectName       string
}

// StructuredReadLogEntry stores the structured format to be created from logs.
type StructuredReadLogEntry struct {
	CommonReadLog

	// It can be safely assumed that the Chunks will be sorted on timestamp as logs
	// are parsed in the order of timestamps.
	Chunks []ReadChunkData
	// ChunkCacheReads contains logs related to chunk cache hit checks.
	ChunkCacheReads []ChunkCacheReadLogEntry
}

// ReadChunkData stores the format of chunk to be stored StructuredReadLogEntry.
type ReadChunkData struct {
	StartTimeSeconds int64
	StartTimeNanos   int64
	StartOffset      int64
	Size             int64
	CacheHit         bool
	IsSequential     bool
	OpID             string
	ExecutionTime    string
}

type Job struct {
	JobID      string
	BucketName string
	ObjectName string
	JobEntries []JobData
	// ChunkCacheDownloads contains logs related to chunk downloads.
	ChunkCacheDownloads []ChunkDownloadLogEntry
}

// JobData stores the job timestamp and offsets for a particular file.
type JobData struct {
	StartTimeSeconds int64
	StartTimeNanos   int64
	Offset           int64
}

// ChunkCacheReadLogEntry stores the details of a chunk cache hit check.
type ChunkCacheReadLogEntry struct {
	StartTimeSeconds int64
	StartTimeNanos   int64
	StartOffset      int64
	EndOffset        int64
	CacheHit         bool
}

// ChunkDownloadLogEntry stores the details of a chunk download.
type ChunkDownloadLogEntry struct {
	StartTimeSeconds int64
	StartTimeNanos   int64
	StartOffset      int64
	EndOffset        int64
	BytesAdded       int64
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// handleAndChunkIndex is used to store reverse mapping of FileCache operation id to
// handle and chunk index stored in structure.
type handleAndChunkIndex struct {
	handle     int64
	chunkIndex int
}

// LogEntry struct to match the JSON structure
type LogEntry struct {
	Timestamp time.Time `json:"time"`
	Message   string    `json:"message"`
}

type BufferedReadLogEntry struct {
	CommonReadLog

	Chunks          []BufferedReadChunkData
	Fallback        bool
	RandomSeekCount int64
	Restarted       bool
}

type BufferedReadChunkData struct {
	StartTimeSeconds int64
	StartTimeNanos   int64
	RequestID        string
	Offset           int64
	Size             int64
	BlockIndex       int64
	ExecutionTime    string
}

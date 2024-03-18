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

package read_logs_test

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	. "github.com/jacobsa/ogletest"
)

const (
	readTimestampSeconds  = 1704458059
	readTimestampNanos    = 975956234
	chunkTimestampSeconds = 1704458060
	chunkTimestampNanos   = 976093794
	pid                   = 2382526
	executionTime         = "293.935998ms"
	opId                  = "f41c82a2-c891"
	fileName              = "smallfile.txt"
	bucketName            = "redacted"
	inodeId               = 6
	size                  = 4096
	handleId              = 29
)

var chunkData = read_logs.ReadChunkData{
	StartTimeSeconds: chunkTimestampSeconds,
	StartTimeNanos:   chunkTimestampNanos,
	StartOffset:      0,
	Size:             size,
	CacheHit:         false,
	IsSequential:     true,
	OpID:             opId,
	ExecutionTime:    executionTime,
}

type testCase struct {
	name        string // Name of the test case
	reader      io.Reader
	expected    map[int64]*read_logs.StructuredReadLogEntry
	errorString string
}

func TestParseLogFileSuccessful(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	tests := []testCase{
		{
			name: "Test file cache logs with 1 chunk",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}`),
			),
			expected: map[int64]*read_logs.StructuredReadLogEntry{
				handleId: {
					Handle:           handleId,
					StartTimeSeconds: readTimestampSeconds,
					StartTimeNanos:   readTimestampNanos,
					ProcessID:        pid,
					InodeID:          inodeId,
					BucketName:       bucketName,
					ObjectName:       fileName,
					Chunks: []read_logs.ReadChunkData{
						chunkData,
					},
				},
			},
		},
		{
			name: "Test file cache logs with multiple chunks",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}
{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}
{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}`),
			),
			expected: map[int64]*read_logs.StructuredReadLogEntry{
				29: {
					Handle:           29,
					StartTimeSeconds: readTimestampSeconds,
					StartTimeNanos:   readTimestampNanos,
					ProcessID:        pid,
					InodeID:          inodeId,
					BucketName:       bucketName,
					ObjectName:       fileName,
					Chunks: []read_logs.ReadChunkData{
						chunkData, chunkData, chunkData,
					},
				},
			},
		},
		{
			name: "Test file cache logs with no parsable logs",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity":"TRACE","message":"fuse_debug: Op 0x00000182        connection.go:497] -> OK ()"}
{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity":"TRACE","message":"fuse_debug: Op 0x00000184        connection.go:415] <- FlushFile (inode 6, PID 2382526)"}
{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity":"TRACE","message":"fuse_debug: Op 0x00000184        connection.go:497] -> OK ()"}`),
			),
			expected: make(map[int64]*read_logs.StructuredReadLogEntry),
		},
		{
			name:     "Test file cache logs with no JSON logs",
			reader:   bytes.NewReader([]byte(`hello 123`)),
			expected: make(map[int64]*read_logs.StructuredReadLogEntry),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := read_logs.ParseReadLogsFromLogFile(tc.reader)
			AssertEq(nil, err)
			AssertTrue(reflect.DeepEqual(actual, tc.expected))
		})
	}
}

func TestParseLogFileUnsuccessful(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	tests := []testCase{
		{
			name: "Test file cache logs without Read File log",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}`),
			),
			errorString: fmt.Sprintf("parseFileCacheRequestLog failed: ReadFile LogEntry for handle %d not found", handleId),
		},
		{
			name: "Test file cache response log without file cache log",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
		{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
		{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}`),
			),
			errorString: fmt.Sprintf("FileCache log entry not found for opID %s", opId),
		},
		{
			name:        "Test invalid read file log",
			reader:      bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode abc, PID 2382526, handle 29, offset 0, 4096 bytes)"}`)),
			errorString: "inode id: could not parse abc to int64",
		},
		{
			name:        "Test invalid read file log",
			reader:      bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID abc, handle 29, offset 0, 4096 bytes)"}`)),
			errorString: "process id: could not parse abc to int64",
		},
		{
			name:        "Test invalid read file log",
			reader:      bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle abc, offset 0, 4096 bytes)"}`)),
			errorString: "handle: could not parse abc to int64",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := read_logs.ParseReadLogsFromLogFile(tc.reader)
			AssertNe(nil, err)
			AssertTrue(strings.Contains(err.Error(), tc.errorString))
		})
	}
}

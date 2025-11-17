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

package read_logs_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

func TestParseBufferedReadLogsFromLogReaderSuccessful(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	tests := []struct {
		name     string // Name of the test case
		reader   io.Reader
		expected map[int64]*read_logs.BufferedReadLogEntry
	}{
		{
			name: "Test buffered read logs with 1 chunk",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			expected: map[int64]*read_logs.BufferedReadLogEntry{
				0: {
					CommonReadLog: read_logs.CommonReadLog{
						Handle:           0,
						StartTimeSeconds: 1754207548,
						StartTimeNanos:   733110719,
						ProcessID:        564246,
						InodeID:          2,
						BucketName:       "princer-working-dirs",
						ObjectName:       "10G_file",
					},
					Chunks: []read_logs.BufferedReadChunkData{
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733199657,
							RequestID:        "2e4645d9-19a8",
							Offset:           34603008,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "223.643µs",
						},
					},
				},
			},
		},
		{
			name: "Test buffered read logs with multiple chunks",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}
{"timestamp":{"seconds":1754207548,"nanos":733776221},"severity":"TRACE","message":"fuse_debug: Op 0x00000050        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 35651584, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733853084},"severity":"TRACE","message":"a8517095-54c0 <- ReadAt(princer-working-dirs:/10G_file, 0, 35651584, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":734027808},"severity":"TRACE","message":"a8517095-54c0 -> ReadAt(): Ok(173.476µs)"}
{"timestamp":{"seconds":1754207548,"nanos":734048914},"severity":"TRACE","message":"fuse_debug: Op 0x00000050        connection.go:548] -> ReadFile ()"}
{"timestamp":{"seconds":1754207548,"nanos":734299231},"severity":"TRACE","message":"fuse_debug: Op 0x00000052        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 36700160, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":734358133},"severity":"TRACE","message":"4e8b1c9c-0012 <- ReadAt(princer-working-dirs:/10G_file, 0, 36700160, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":734532341},"severity":"TRACE","message":"4e8b1c9c-0012 -> ReadAt(): Ok(179.8µs)"}`),
			),
			expected: map[int64]*read_logs.BufferedReadLogEntry{
				0: {
					CommonReadLog: read_logs.CommonReadLog{
						Handle:           0,
						StartTimeSeconds: 1754207548,
						StartTimeNanos:   733110719,
						ProcessID:        564246,
						InodeID:          2,
						BucketName:       "princer-working-dirs",
						ObjectName:       "10G_file",
					},
					Chunks: []read_logs.BufferedReadChunkData{
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733199657,
							RequestID:        "2e4645d9-19a8",
							Offset:           34603008,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "223.643µs",
						},
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733853084,
							RequestID:        "a8517095-54c0",
							Offset:           35651584,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "173.476µs",
						},
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   734358133,
							RequestID:        "4e8b1c9c-0012",
							Offset:           36700160,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "179.8µs",
						},
					},
				},
			},
		},
		{
			name: "Test buffered read logs with no fallback",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			expected: map[int64]*read_logs.BufferedReadLogEntry{
				0: {
					CommonReadLog: read_logs.CommonReadLog{
						Handle:           0,
						StartTimeSeconds: 1754207548,
						StartTimeNanos:   733110719,
						ProcessID:        564246,
						InodeID:          2,
						BucketName:       "princer-working-dirs",
						ObjectName:       "10G_file",
					},
					Chunks: []read_logs.BufferedReadChunkData{
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733199657,
							RequestID:        "2e4645d9-19a8",
							Offset:           34603008,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "223.643µs",
						},
					},
					Fallback:        false,
					RandomSeekCount: 0,
				},
			},
		},
		{
			name: "Test buffered read logs with no parsable logs",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":742190759},"severity":"TRACE","message":"Scheduling block: (10G_file, 9, false)."}
{"timestamp":{"seconds":1754207548,"nanos":742296187},"severity":"TRACE","message":"Scheduling block: (10G_file, 10, false)."}
{"timestamp":{"seconds":1754207548,"nanos":742300356},"severity":"TRACE","message":"Download: <- block (10G_file, 9)."}
{"timestamp":{"seconds":1754207548,"nanos":742315339},"severity":"TRACE","message":"Scheduling block: (10G_file, 11, false)."}
{"timestamp":{"seconds":1754207548,"nanos":742323114},"severity":"TRACE","message":"Scheduling block: (10G_file, 12, false)."}`),
			),
			expected: make(map[int64]*read_logs.BufferedReadLogEntry),
		},
		{
			name: "Test buffered read logs with fallback",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733200000},"severity":"WARN","message":"Fallback to another reader for object \"10G_file\", handle 0. Random seek count 4 exceeded threshold 3."}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			expected: map[int64]*read_logs.BufferedReadLogEntry{
				0: {
					CommonReadLog: read_logs.CommonReadLog{
						Handle:           0,
						StartTimeSeconds: 1754207548,
						StartTimeNanos:   733110719,
						ProcessID:        564246,
						InodeID:          2,
						BucketName:       "princer-working-dirs",
						ObjectName:       "10G_file",
					},
					Chunks: []read_logs.BufferedReadChunkData{
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733199657,
							RequestID:        "2e4645d9-19a8",
							Offset:           34603008,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "223.643µs",
						},
					},
					Fallback:        true,
					RandomSeekCount: 4,
				},
			},
		},
		{
			name: "Test buffered read logs with generic fallback",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733200000},"severity":"WARN","message":"Fallback to another reader for object \"10G_file\", handle 0. Due to freshStart failure: some error"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			expected: map[int64]*read_logs.BufferedReadLogEntry{
				0: {
					CommonReadLog: read_logs.CommonReadLog{
						Handle:           0,
						StartTimeSeconds: 1754207548,
						StartTimeNanos:   733110719,
						ProcessID:        564246,
						InodeID:          2,
						BucketName:       "princer-working-dirs",
						ObjectName:       "10G_file",
					},
					Chunks: []read_logs.BufferedReadChunkData{
						{
							StartTimeSeconds: 1754207548,
							StartTimeNanos:   733199657,
							RequestID:        "2e4645d9-19a8",
							Offset:           34603008,
							Size:             1048576,
							BlockIndex:       2,
							ExecutionTime:    "223.643µs",
						},
					},
					Fallback:        true,
					RandomSeekCount: 0,
				},
			},
		},
		{
			name:     "Test buffered read logs with no JSON logs",
			reader:   bytes.NewReader([]byte(`hello 123`)),
			expected: make(map[int64]*read_logs.BufferedReadLogEntry),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := read_logs.ParseBufferedReadLogsFromLogReader(tc.reader)

			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual, fmt.Sprintf("Expected: %v, Actual: %v", tc.expected, actual))
		})
	}
}

func TestBufferedReadLogsFromLogReaderUnsuccessful(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	tests := []struct {
		name        string // Name of the test case
		reader      io.Reader
		errorString string
	}{
		{
			name: "Test buffered read logs without Read File log",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733199657},"severity":"TRACE","message":"2e4645d9-19a8 <- ReadAt(princer-working-dirs:/10G_file, 0, 34603008, 1048576, 2)"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			errorString: "BufferedReadLogEntry for handle 0 not found",
		},
		{
			name: "Test buffered read logs response log without request log",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode 2, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}
{"timestamp":{"seconds":1754207548,"nanos":733417812},"severity":"TRACE","message":"2e4645d9-19a8 -> ReadAt(): Ok(223.643µs)"}
{"timestamp":{"seconds":1754207548,"nanos":733444394},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:548] -> ReadFile ()"}`),
			),
			errorString: "request ID 2e4645d9-19a8 not found in reverse map",
		},
		{
			name:        "Test invalid read file log - invalid Inode ID",
			reader:      bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733110719},"severity":"TRACE","message":"fuse_debug: Op 0x0000004e        connection.go:453] <- ReadFile (inode abc, PID 564246, handle 0, offset 34603008, 1048576 bytes)"}`)),
			errorString: "parseReadFileLog failed: invalid ReadFile log",
		},
		{
			name:        "Test invalid read file log - invalid PID",
			reader:      bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID abc, handle 29, offset 0, 4096 bytes)"}`)),
			errorString: "parseReadFileLog failed: invalid ReadFile log format",
		},
		{
			name:        "Test invalid read file log - invalid Handle",
			reader:      bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle abc, offset 0, 4096 bytes)"}`)),
			errorString: "parseReadFileLog failed: invalid ReadFile log format",
		},
		{
			name:        "Test fallback log for unknown handle",
			reader:      bytes.NewReader([]byte(`{"timestamp":{"seconds":1754207548,"nanos":733200000},"severity":"WARN","message":"Fallback to another reader for object \"10G_file\", handle 99. Random seek count 4 exceeded threshold 3."}`)),
			errorString: "log entry for handle 99 not found for fallback log",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := read_logs.ParseBufferedReadLogsFromLogReader(tc.reader)

			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), tc.errorString), fmt.Sprintf("Unexpected error: %s", err))
		})
	}
}

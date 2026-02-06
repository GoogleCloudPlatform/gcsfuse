// Copyright 2024 Google LLC
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
	"bytes"
	"io"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name     string // Name of the test case
	reader   io.Reader
	expected map[string]*Job
}

func TestParseJobLogsSuccessful(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	tests := []testCase{
		{
			name: "Test file cache logs with 1 job log",
			reader: bytes.NewReader([]byte(`{"timestamp": {"seconds": 1704458059, "nanos": 975956234}, "severity": "TRACE", "message": "fuse_debug: Op 0x00000182        connection.go:415] <- ReadFile (inode 6, PID 2382526, handle 29, offset 0, 4096 bytes)"}
{"timestamp": {"seconds": 1704458060, "nanos": 976093794}, "severity": "TRACE", "message": "f41c82a2-c891 <- FileCache(redacted:/smallfile.txt, offset: 0, size: 4096 handle: 29)"}
{"timestamp": {"seconds": 1704458061, "nanos": 269924363}, "severity": "TRACE", "message": "Job:0xc000aa65b0 (redacted:/smallfile.txt) downloaded till 6 offset."}
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}`),
			),
			expected: map[string]*Job{
				"0xc000aa65b0": {
					BucketName: "redacted",
					ObjectName: "smallfile.txt",
					JobID:      "0xc000aa65b0",
					JobEntries: []JobData{{
						StartTimeSeconds: 1704458061,
						StartTimeNanos:   269924363,
						Offset:           6,
					}},
				},
			},
		},
		{
			name: "Test file cache logs with multiple job logs",
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
{"timestamp": {"seconds": 1704458061, "nanos": 270075223}, "severity": "TRACE", "message": "f41c82a2-c891 -> OK (isSeq: true, hit: false) (293.935998ms)"}
{"timestamp":{"seconds":1721228431,"nanos":993427325},"severity":"TRACE","message":"Job:0xc000af6000 (redacted-a-b-c:/ReadCacheTest/foo4ckc) downloaded till 8388608 offset."}`),
			),
			expected: map[string]*Job{
				"0xc000aa65b0": {
					BucketName: "redacted",
					ObjectName: "smallfile.txt",
					JobID:      "0xc000aa65b0",
					JobEntries: []JobData{
						{
							StartTimeSeconds: 1704458061,
							StartTimeNanos:   269924363,
							Offset:           6,
						},
						{
							StartTimeSeconds: 1704458061,
							StartTimeNanos:   269924363,
							Offset:           6,
						},
						{
							StartTimeSeconds: 1704458061,
							StartTimeNanos:   269924363,
							Offset:           6,
						}},
				},
				"0xc000af6000": {
					BucketName: "redacted-a-b-c",
					ObjectName: "ReadCacheTest/foo4ckc",
					JobID:      "0xc000af6000",
					JobEntries: []JobData{
						{
							StartTimeSeconds: 1721228431,
							StartTimeNanos:   993427325,
							Offset:           8388608,
						}},
				},
			},
		},
		{
			name:   "Test chunk download logs",
			reader: bytes.NewReader([]byte(`{"timestamp":{"seconds":1721228431,"nanos":993427325},"severity":"TRACE","message":"Job:0xc000af6000 (bucket:/obj) downloaded range [0, 10), added 10 bytes to sparse file"}`)),
			expected: map[string]*Job{
				"0xc000af6000": {
					BucketName: "bucket",
					ObjectName: "obj",
					JobID:      "0xc000af6000",
					ChunkCacheDownloads: []ChunkDownloadLogEntry{
						{
							StartTimeSeconds: 1721228431,
							StartTimeNanos:   993427325,
							StartOffset:      0,
							EndOffset:        10,
							BytesAdded:       10,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseJobLogsFromLogFile(tc.reader)
			require.Nil(t, err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

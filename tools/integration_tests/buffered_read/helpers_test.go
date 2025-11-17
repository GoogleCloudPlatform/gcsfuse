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

package buffered_read

import (
	"bytes"
	"context"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

// Expected is a helper struct that stores list of attributes to be validated from logs.
type Expected struct {
	StartTimeStampSeconds int64
	EndTimeStampSeconds   int64
	BucketName            string
	ObjectName            string
}

func readFileAndValidate(ctx context.Context, storageClient *storage.Client, testDir, fileName string, readFullFile bool, offset int64, chunkSizeToRead int64, t *testing.T) *Expected {
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(path.Base(testDir), fileName),
	}
	if setup.DynamicBucketMounted() != "" {
		expected.BucketName = setup.DynamicBucketMounted()
	}
	if readFullFile {
		content, err := operations.ReadFileSequentially(path.Join(testDir, fileName), chunkSizeToRead)
		require.NoError(t, err, "Failed to read file sequentially")
		obj := storageClient.Bucket(expected.BucketName).Object(expected.ObjectName)
		attrs, err := obj.Attrs(ctx)
		require.NoError(t, err, "obj.Attrs")
		localCRC32C, err := operations.CalculateCRC32(bytes.NewReader(content))
		require.NoError(t, err, "Error while calculating crc for the content read from mounted file")
		assert.Equal(t, attrs.CRC32C, localCRC32C, "CRC32C mismatch. GCS: %d, Local: %d", attrs.CRC32C, localCRC32C)
	} else {
		content, err := operations.ReadChunkFromFile(path.Join(testDir, fileName), chunkSizeToRead, offset, os.O_RDONLY|syscall.O_DIRECT)
		require.NoError(t, err, "Failed to read random file chunk")
		client.ValidateObjectChunkFromGCS(ctx, storageClient, path.Base(testDir), fileName, offset, chunkSizeToRead, string(content), t)
	}
	expected.EndTimeStampSeconds = time.Now().Unix()
	return expected
}

func validate(expected *Expected, logEntry *read_logs.BufferedReadLogEntry, fallback bool, t *testing.T) {
	t.Helper()
	assert.GreaterOrEqual(t, logEntry.StartTimeSeconds, expected.StartTimeStampSeconds, "start time in logs %d less than actual start time %d.", logEntry.StartTimeSeconds, expected.StartTimeStampSeconds)

	assert.Equal(t, expected.BucketName, logEntry.BucketName, "Bucket names don't match! Expected: %s, Got from logs: %s",
		expected.BucketName, logEntry.BucketName)

	assert.Equal(t, expected.ObjectName, logEntry.ObjectName, "Object names don't match! Expected: %s, Got from logs: %s",
		expected.ObjectName, logEntry.ObjectName)

	assert.Equal(t, fallback, logEntry.Fallback, "Expected Fallback: %t, Got from logs: %t", fallback, logEntry.Fallback)
}

func setupFileInTestDir(ctx context.Context, storageClient *storage.Client, testDir string, fileSize int64, t *testing.T) (fileName string) {
	fileName = testFileName + setup.GenerateRandomString(4)
	client.SetupFileInTestDirectory(ctx, storageClient, path.Base(testDir), fileName, fileSize, t)
	return fileName
}

func parseBufferedReadLogs(t *testing.T) map[int64]*read_logs.BufferedReadLogEntry {
	t.Helper()
	f := operations.OpenFile(setup.LogFile(), t)
	defer operations.CloseFileShouldNotThrowError(t, f)

	logEntries, err := read_logs.ParseBufferedReadLogsFromLogReader(f)
	require.NoError(t, err, "Failed to parse log file")
	return logEntries
}

func parseAndValidateSingleBufferedReadLog(t *testing.T) *read_logs.BufferedReadLogEntry {
	t.Helper()
	logEntries := parseBufferedReadLogs(t)
	// The test is expected to generate exactly one buffered read log entry because
	// all reads are performed through a single file handle.
	require.Len(t, logEntries, 1, "Expected one buffered read log entry for the single file handle.")

	for _, entry := range logEntries {
		return entry
	}
	return nil // Unreachable.
}

func readAndValidateChunk(f *os.File, testDir, fileName string, offset, chunkSize int64, t *testing.T) {
	t.Helper()
	readBuffer := make([]byte, chunkSize)

	_, err := f.ReadAt(readBuffer, offset)

	require.NoError(t, err, "ReadAt failed at offset %d", offset)
	client.ValidateObjectChunkFromGCS(ctx, storageClient, testDir, fileName, offset, chunkSize, string(readBuffer), t)
}

// induceRandomReadFallback performs a sequence of reads designed to trigger the
// random read fallback mechanism in the buffered reader. It starts with a
// sequential read and then alternates between a distant offset and offset 0.
func induceRandomReadFallback(t *testing.T, f *os.File, testDir, fileName string, chunkSize, distantOffset int64, randomReadsThreshold int) {
	t.Helper()
	// We need 1 sequential read + randomReadsThreshold successful random reads before
	// the final one that triggers fallback.
	offset := int64(0)
	for i := 0; i < randomReadsThreshold+1; i++ {
		readAndValidateChunk(f, testDir, fileName, offset, chunkSize, t)
		offset = distantOffset ^ offset
	}

	// The next read should trigger the fallback but still succeed from the user's perspective.
	readAndValidateChunk(f, testDir, fileName, offset, chunkSize, t)
}

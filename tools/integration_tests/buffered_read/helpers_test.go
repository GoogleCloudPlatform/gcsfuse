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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

// Expected is a helper struct that stores list of attributes to be validated from logs.
type Expected struct {
	StartTimeStampSeconds int64
	EndTimeStampSeconds   int64
	BucketName            string
	ObjectName            string
}

func readFileAndValidate(ctx context.Context, storageClient *storage.Client, fileName string, readFullFile bool, offset int64, chunkSizeToRead int64, t *testing.T) *Expected {
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, fileName),
	}
	if setup.DynamicBucketMounted() != "" {
		expected.BucketName = setup.DynamicBucketMounted()
	}

	var content []byte
	var err error

	if readFullFile {
		content, err = operations.ReadFileSequentially(path.Join(testDirPath, fileName), chunkSizeToRead)
		require.NoError(t, err, "Failed to read file sequentially")
		// Get checksum from GCS object.
		obj := storageClient.Bucket(expected.BucketName).Object(expected.ObjectName)
		attrs, err := obj.Attrs(ctx)
		require.NoError(t, err, "obj.Attrs")
		// Calculate checksum of read content and compare with GCS object's checksum.
		localCRC32C, err := operations.CalculateCRC32(bytes.NewReader(content))
		require.NoError(t, err, "Error while calculating crc for the content read from mounted file")
		assert.Equal(t, attrs.CRC32C, localCRC32C, "CRC32C mismatch. GCS: %d, Local: %d", attrs.CRC32C, localCRC32C)
	} else {
		content, err = operations.ReadChunkFromFile(path.Join(testDirPath, fileName), chunkSizeToRead, offset, os.O_RDONLY|syscall.O_DIRECT)
		require.NoError(t, err, "Failed to read random file chunk")
		client.ValidateObjectChunkFromGCS(ctx, storageClient, testDirName, fileName, offset, chunkSizeToRead, string(content), t)
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

	if len(logEntry.Chunks) > 0 {
		assert.LessOrEqual(t, logEntry.Chunks[len(logEntry.Chunks)-1].StartTimeSeconds, expected.EndTimeStampSeconds, "end time in logs more than actual end time.")
	}
	assert.Equal(t, fallback, logEntry.Fallback, "Expected Fallback: %t, Got from logs: %t", fallback, logEntry.Fallback)
}

func setupFileInTestDir(ctx context.Context, storageClient *storage.Client, fileSize int64, t *testing.T) (fileName string) {
	fileName = testFileName + setup.GenerateRandomString(4)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, fileName, fileSize, t)
	return
}

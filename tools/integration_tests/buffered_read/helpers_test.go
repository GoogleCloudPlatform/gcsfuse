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
	"context"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func validate(expected *client.Expected, logEntry *read_logs.BufferedReadLogEntry, fallback bool, t *testing.T) {
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

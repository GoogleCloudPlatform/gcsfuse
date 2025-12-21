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

package streaming_writes

import (
	"os"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
)

func (t *StreamingWritesSuite) TestOutOfOrderWriteSyncsFileToGcs() {
	// Write
	operations.WriteWithoutClose(t.f1, "foobar", t.T())
	operations.VerifyStatFile(t.filePath, int64(len("foobar")), FilePerms, t.T())

	// Perform out of order write.
	operations.WriteAt("foo", 3, t.f1, t.T())

	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, "foobar", t.T())
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, "foofoo", t.T())
}

func (t *StreamingWritesSuite) TestOutOfOrderWriteSyncsFileToGcsAndDeletingFileDeletesFileFromGcs() {
	// Write
	operations.WriteWithoutClose(t.f1, "foobar", t.T())
	operations.VerifyStatFile(t.filePath, int64(len("foobar")), FilePerms, t.T())
	// Perform out of order write.
	operations.WriteAt("foo", 3, t.f1, t.T())
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, "foobar", t.T())

	err := os.Remove(t.filePath)

	require.NoError(t.T(), err)
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, t.T())
}

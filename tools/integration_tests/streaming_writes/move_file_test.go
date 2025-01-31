// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package streaming_writes

import (
	"path"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
)

func (t *defaultMountEmptyGCSFile) TestMoveBeforeFileIsFlushed() {
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	operations.VerifyStatFile(t.filePath, int64(2*len(FileContents)), FilePerms, t.T())
	err := t.f1.Sync()
	require.NoError(t.T(), err)

	newFile := "newFile.txt"
	destDirPath := path.Join(testDirPath, newFile)
	err = operations.Move(t.filePath, destDirPath)

	// Validate that move didn't throw any error.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, newFile, FileContents+FileContents, t.T())
	require.NoError(t.T(), t.f1.Close())
	// Check if old object is deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())
}

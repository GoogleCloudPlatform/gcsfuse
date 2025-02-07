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
	"path"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
)

func (t *defaultMountEmptyGCSFile) TestClobberedFileCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	// Replace the underlying object with a new generation.
	err := WriteToObject(ctx, storageClient, path.Join(testDirName, t.fileName), GCSFileContent, storage.Conditions{})
	assert.NoError(t.T(), err)

	err = t.f1.Close()

	operations.ValidateStaleNFSFileHandleError(t.T(), err)
}

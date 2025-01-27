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
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/stretchr/testify/assert"
)

func (t *defaultMountLocalFile) TestReadLocalFileFails() {
	// Write some content to local file.
	t.f1.WriteAt([]byte(FileContents), 0)

	// Reading the local file content fails.
	buf := make([]byte, len(FileContents))
	_, err := t.f1.ReadAt(buf, 0)
	assert.Error(t.T(), err)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, FileContents, t.T())
}

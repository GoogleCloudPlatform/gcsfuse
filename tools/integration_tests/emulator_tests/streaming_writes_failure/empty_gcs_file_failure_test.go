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

package streaming_writes_failure

import (
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type emptyGcsFileFailureTestSuite struct {
	commonFailureTestSuite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *emptyGcsFileFailureTestSuite) SetupTest() {
	t.configPath = "../configs/empty_gcs_file_2nd_chunk_upload_returns412.yaml"
	t.setupTest()
	// Create an empty file on GCS.
	CreateObjectInGCSTestDir(t.ctx, t.storageClient, testDirName, FileName1, "", t.T())
	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, testDirName, FileName1, "", t.T())
	t.filePath = path.Join(t.testDirPath, FileName1)
	t.fh1 = operations.OpenFile(t.filePath, t.T())
}

func (t *emptyGcsFileFailureTestSuite) validateGcsObject() {
	// Validate empty gcs file is found on GCS.
	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, testDirName, FileName1, "", t.T())
}

// Executes all failure tests for empty gcs files.
func TestEmptyGcsFileFailureTestSuite(t *testing.T) {
	s := new(emptyGcsFileFailureTestSuite)
	s.gcsObjectValidator = s
	suite.Run(t, s)
}

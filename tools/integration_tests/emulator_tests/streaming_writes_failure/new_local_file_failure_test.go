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
	"log"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type newLocalFileFailureTestSuite struct {
	commonFailureTestSuite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *newLocalFileFailureTestSuite) SetupSuite() {
	t.configPath = "../proxy_server/configs/local_file_2nd_chunk_upload_returns412.yaml"
	t.flags = []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=1", "--custom-endpoint=" + proxyEndpoint}
	// Generate 5 MB random data.
	var err error
	t.data, err = operations.GenerateRandomData(5 * operations.MiB)
	require.NoError(t.T(), err)
	log.Printf("Running tests with flags for new local files: %v", t.flags)
}

func (t *newLocalFileFailureTestSuite) SetupTest() {
	t.setupTest()
	// Create local file.
	t.filePath, t.fh1 = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())
}

func (t *newLocalFileFailureTestSuite) validateGcsObject() {
	// Validate file not found error from GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, FileName1, t.T())
}

// Executes all failure tests for new local files.
func TestNewLocalFileFailureTestSuite(t *testing.T) {
	s := new(newLocalFileFailureTestSuite)
	s.gcsObjectValidator = s
	suite.Run(t, s)
}

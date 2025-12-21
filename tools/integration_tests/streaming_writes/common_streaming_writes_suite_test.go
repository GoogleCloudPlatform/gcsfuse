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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StreamingWritesSuite struct {
	f1       *os.File
	fileName string
	// filePath of the above file in the mounted directory.
	filePath string
	data     string
	dirName  string
	test_suite.TestifySuite
}

func (t *StreamingWritesSuite) SetupSuite() {
	testEnv.testDirPath = setup.SetupTestDirectory(testDirName)
	t.data = setup.GenerateRandomString(5 * util.MiB)
}

func (t *StreamingWritesSuite) TearDownSuite() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
}

func (t *StreamingWritesSuite) validateReadCall(fh *os.File, content string) {
	readContent := make([]byte, len(content))
	n, err := fh.ReadAt(readContent, 0)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), len(content), n)
	assert.Equal(t.T(), content, string(readContent))
}

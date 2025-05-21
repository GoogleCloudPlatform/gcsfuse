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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/local_file"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StreamingWritesSuite struct {
	f1       *os.File
	fileName string
	// filePath of the above file in the mounted directory.
	filePath string
	data     string
	test_suite.TestifySuite
}

func (t *StreamingWritesSuite) SetupSuite() {
	// TODO(mohitkyadav): Make these part of test suite after refactoring.
	SetCtx(ctx)
	SetStorageClient(storageClient)
	SetTestDirName(testDirName)

	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	testDirPath = setup.SetupTestDirectory(testDirName)
	t.data = setup.GenerateRandomString(5 * util.MiB)
}

func (t *StreamingWritesSuite) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
}

func (t *StreamingWritesSuite) validateReadFromSymlink(filePath, content string) {
	readContent, err := os.ReadFile(filePath)
	// TODO(b/410698332): Fix validation once zb reads start working.
	if setup.IsZonalBucketRun() {
		operations.ValidateEOPNOTSUPPError(t.T(), err)
	} else {
		require.NoError(t.T(), err)
		assert.Equal(t.T(), content, string(readContent))
	}
}

func (t *StreamingWritesSuite) validateReadCall(fh *os.File, content string) {
	readContent := make([]byte, len(content))
	n, err := fh.Read(readContent)
	// TODO(b/410698332): Fix validation once zb reads start working.
	if setup.IsZonalBucketRun() {
		operations.ValidateEOPNOTSUPPError(t.T(), err)
	} else {
		require.NoError(t.T(), err)
		assert.Equal(t.T(), len(content), n)
		assert.Equal(t.T(), content, string(readContent))
	}
}

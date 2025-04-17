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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/require"
)

type defaultMountCommonTest struct {
	f1       *os.File
	fileName string
	// filePath of the above file in the mounted directory.
	filePath string
	data     string
	test_suite.TestifySuite
}

func (t *defaultMountCommonTest) SetupSuite() {
	// TODO(mohitkyadav): Make these part of test suite after refactoring.
	SetCtx(ctx)
	SetStorageClient(storageClient)
	SetTestDirName(testDirName)

	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	testDirPath = setup.SetupTestDirectory(testDirName)
	t.data = setup.GenerateRandomString(5 * util.MiB)
}

func (t *defaultMountCommonTest) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}

func (t *defaultMountCommonTest) validateReadCall(filePath string) {
	_, err := os.ReadFile(filePath)
	if setup.IsZonalBucketRun() {
		// TODO(b/410698332): Remove skip condition once reads start working.
		t.T().Skip("Skipping Zonal Bucket Read tests.")
		require.NoError(t.T(), err)
	} else {
		require.Error(t.T(), err)
	}
}

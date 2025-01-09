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

package negative_stat_cache

import (
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

type finiteNegativeStatCacheTest struct {
	flags []string
}

func (s *finiteNegativeStatCacheTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *finiteNegativeStatCacheTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *finiteNegativeStatCacheTest) TestFiniteNegativeStatCache(t *testing.T) {
	targetDir := path.Join(testDirPath, "explicit_dir")
	// Create test directory
	operations.CreateDirectory(targetDir, t)
	targetFile := path.Join(targetDir, "file1.txt")

	// Error should be returned as file does not exist
	_, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	assert.NotNil(t, err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(t, err, "explicit_dir/file1.txt: no such file or directory")

	// Adding the object with same name
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file1.txt"), "some-content", t)

	// Error should be returned again, as call will not be served from GCS due to finite gcsfuse stat cache
	_, err = os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	assert.NotNil(t, err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(t, err, "explicit_dir/file1.txt: no such file or directory")

	//Wait for Cache to expire
	time.Sleep(3 * time.Second)

	// File should be returned, as call will be served from GCS and gcsfuse should not return from cache
	f, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	//Assert File is found
	assert.NoError(t, err)
	assert.Contains(t, f.Name(), "explicit_dir/file1.txt")
	assert.Nil(t, f.Close())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestFiniteNegativeStatCacheTest(t *testing.T) {
	ts := &finiteNegativeStatCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := []string{"--metadata-cache-negative-ttl-secs=2"}

	// Run tests.
	ts.flags = flagsSet
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)
}

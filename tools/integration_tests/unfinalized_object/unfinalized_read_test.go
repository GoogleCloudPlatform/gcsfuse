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

package unfinalized_object

import (
	"context"
	"log"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type unfinalizedObjectReads struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	fileName      string
	suite.Suite
}

func (t *unfinalizedObjectReads) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (t *unfinalizedObjectReads) TeardownTest() {}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *unfinalizedObjectReads) TestUnfinalizedObjectsCantBeRead() {
	var size int = operations.MiB
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, t.fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(size), t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), fh)

	// Read un-finalized object.
	content, err := operations.ReadFileSequentially(path.Join(t.testDirPath, t.fileName), util.MiB)

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, syscall.ENOTSUP.Error())
	assert.Empty(t.T(), content, "expected empty content, but got size: %v", len(content))
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedObjectReadTest(t *testing.T) {
	ts := &unfinalizedObjectReads{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--enable-streaming-writes=true", "--metadata-cache-ttl-secs=-1"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		setup.MountGCSFuseWithGivenMountFunc(ts.flags, mountFunc)
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())
	}
}

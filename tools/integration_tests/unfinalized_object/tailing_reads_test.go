// Copyright 2026 Google LLC
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
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type unfinalizedObjectTailingReads struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	fileName      string
	suite.Suite
}

func (t *unfinalizedObjectTailingReads) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (s *unfinalizedObjectTailingReads) TearDownSuite() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *unfinalizedObjectTailingReads) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (t *unfinalizedObjectTailingReads) TestTailingRead() {
	// 1. Create file
	initialContent := setup.GenerateRandomString(initialSize)
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), initialContent)

	readPath := path.Join(t.testDirPath, t.fileName)

	// 2. Open file for reading
	readFile, err := os.OpenFile(readPath, os.O_RDONLY, setup.FilePermission_0600)
	require.NoError(t.T(), err)
	defer operations.CloseFileShouldNotThrowError(t.T(), readFile)

	// 3. Read initial content
	buf := make([]byte, len(initialContent))
	n, err := readFile.Read(buf)
	require.NoError(t.T(), err)
	require.Equal(t.T(), len(initialContent), n)
	require.Equal(t.T(), initialContent, string(buf))

	// 4. Loop appends
	numAppends := 2
	appendSize := 10
	for i := 0; i < numAppends; i++ {
		// Append
		appendData := setup.GenerateRandomString(appendSize)

		// Remotely append content to the object.
		obj, err := t.storageClient.Bucket(setup.TestBucket()).Object(path.Join(testDirName, t.fileName)).Attrs(t.ctx)
		require.NoError(t.T(), err)

		writer, err := client.AppendableWriter(t.ctx, t.storageClient, path.Join(testDirName, t.fileName), obj.Generation)
		require.NoError(t.T(), err)
		_, err = writer.Write([]byte(appendData))
		require.NoError(t.T(), err)
		err = writer.Close()
		require.NoError(t.T(), err)

		// Wait for metadata cache to expire if needed.
		// Since we are running with --metadata-cache-ttl-secs=2, we should wait slightly more than that.
		time.Sleep(3 * time.Second)

		// Check Stat (fstat on the read handle)
		fi, err := readFile.Stat()
		require.NoError(t.T(), err)

		expectedSize := int64(len(initialContent) + (i+1)*appendSize)
		require.Equal(t.T(), expectedSize, fi.Size(), "File size should update after append")

		// Read new data
		newBuf := make([]byte, len(appendData))
		n, err = readFile.Read(newBuf)
		require.NoError(t.T(), err)
		require.Equal(t.T(), len(appendData), n)
		require.Equal(t.T(), appendData, string(newBuf))
	}
}

func TestUnfinalizedObjectTailingReadTest(t *testing.T) {
	ts := &unfinalizedObjectTailingReads{ctx: context.Background(), storageClient: testEnv.storageClient}

	// Run tests for mounted directory if the flag is set.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

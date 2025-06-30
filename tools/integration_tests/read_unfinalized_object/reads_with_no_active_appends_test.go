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
	"math/rand"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
	content       string
	cacheDir      string
	suite.Suite
}

func (t *unfinalizedObjectReads) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (t *unfinalizedObjectReads) TeardownTest() {
	os.RemoveAll(path.Join(t.testDirPath, t.fileName))
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func (t *unfinalizedObjectReads) createUnfinalizedObject(size int64) {
	t.T().Helper()
	t.content = setup.GenerateRandomString(int(size))
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, t.fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, t.content, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), fh)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *unfinalizedObjectReads) Test_ReadUnfinalizedWithNoActiveAppends_SequentialRead() {
	t.createUnfinalizedObject(100 * util.MiB)

	// Read un-finalized object.
	readContent, err := operations.ReadFileSequentially(path.Join(t.testDirPath, t.fileName), util.MiB)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.content, string(readContent))
}

func (t *unfinalizedObjectReads) Test_ReadUnfinalizedWithNoActiveAppends_RandomRead() {
	numReads := 50
	fileSize := int64(500 * util.MiB)
	maxReadChunkSize := int64(200 * util.MiB)
	fullFilePath := path.Join(t.testDirPath, t.fileName)
	t.createUnfinalizedObject(fileSize)
	maxParallelReads := 10
	var wg sync.WaitGroup
	// Use a buffered channel as a semaphore to limit concurrent reads.
	sem := make(chan struct{}, maxParallelReads)

	// Read unfinalized object in random chunks.
	for i := range numReads {
		wg.Add(1)
		// Acquire a token. This will block if the semaphore is full.
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			// Release the token back to the semaphore.
			defer func() { <-sem }()

			// Create a new random source for each goroutine to avoid lock contention
			// on the global rand source.
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
			readChunkSize := 1 + r.Int63n(maxReadChunkSize-1)
			readOffset := r.Int63n(fileSize - readChunkSize)

			readContent, err := operations.ReadChunkFromFile(path.Join(t.testDirPath, t.fileName), readChunkSize, readOffset, os.O_RDONLY|syscall.O_DIRECT)

			endOffset := readOffset + readChunkSize
			require.NoErrorf(t.T(), err, "Failed to read %q from [%09d, %09d]: %v", fullFilePath, readOffset, readOffset+readChunkSize, err)
			expectedContent := t.content[readOffset:endOffset]
			assert.Equalf(t.T(), string(readContent), expectedContent, "Read of %q from [%09d, %09d] failed with content mismatch.", fullFilePath, readOffset, readOffset+readChunkSize)
		}(i)
	}
	// Wait for all concurrent reads to complete.
	wg.Wait()
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

	ts.cacheDir = path.Join("/tmp", "gcsfuse-cachedir-"+setup.GenerateRandomString(5))

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--metadata-cache-ttl-secs=0"},
		{"--metadata-cache-ttl-secs=0", "--cache-dir=" + ts.cacheDir, "--file-cache-max-size-mb=-1", "--file-cache-cache-file-for-range-read=true"},
		{"--metadata-cache-ttl-secs=300"},
		{"--metadata-cache-ttl-secs=300", "--cache-dir=" + ts.cacheDir, "--file-cache-max-size-mb=-1", "--file-cache-cache-file-for-range-read=true"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		require.NoError(os.Mkdir(ts.cacheDir, setup.DirPermission_0755))
		setup.MountGCSFuseWithGivenMountFunc(ts.flags, mountFunc)
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())
		require.NoError(os.RemoveAll(ts.cacheDir))
	}
}

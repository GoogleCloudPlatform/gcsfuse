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

package rapid_appends

import (
	"log"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numAppends            = 2  // Number of appends to perform on test file.
	appendSize            = 10 // Size in bytes for each append.
	unfinalizedObjectSize = 10 // Size in bytes of initial unfinalized Object.
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DualMountAppendsSuite) TestAppendSessionInvalidatedByAnotherClientUponTakeover() {
	const initialContent = "dummy content"
	const appendContent = "appended content"
	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		var err error
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			// Initiate an append session using the primary file handle opened in append mode.
			appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.primaryMount.testDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			_, err = appendFileHandle.WriteString(initialContent)
			require.NoError(t.T(), err)

			// Open a new file handle from the secondary mount to the same file.
			newAppendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.secondaryMount.testDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer operations.CloseFileShouldNotThrowError(t.T(), newAppendFileHandle)

			// Attempt to append using the newly opened file handle.
			// This append should succeed, confirming the takeover.
			_, err = newAppendFileHandle.WriteString(appendContent)
			assert.NoError(t.T(), err)

			// Attempt to append using the original file handle.
			// This should now fail, as its append session has been invalidated by the takeover.
			_, _ = appendFileHandle.WriteString(appendContent)
			err = appendFileHandle.Sync()
			operations.ValidateESTALEError(t.T(), err)

			// Syncing from the newly created file handle must succeed since it holds the active
			// append session.
			err = newAppendFileHandle.Sync()
			assert.NoError(t.T(), err)

			// Read directly using storage client to validate the contents which has persisted in
			// GCS after takeover from the secondary mount.
			// Close the open append handle before issuing read on the file as Sync() triggered on
			// ReadFile() due to BWH still being initialized, is expected to error out with stale NFS file handle.
			operations.CloseFileShouldThrowError(t.T(), appendFileHandle)
			expectedContent := t.fileContent + appendContent
			content, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), expectedContent, string(content))
		}()
	}
}

func (t *SingleMountAppendsSuite) TestContentAppendedInNonAppendModeNotVisibleTillClose() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()

	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			initialContent := t.fileContent
			// Append to the file from the primary mount in non-append mode
			wh, err := os.OpenFile(path.Join(t.primaryMount.testDirPath, t.fileName), os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
			require.NoError(t.T(), err)

			// Write sufficient data to the end of file.
			data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
			n, err := wh.WriteAt([]byte(data), int64(len(initialContent)))
			require.NoError(t.T(), err)
			require.Equal(t.T(), len(data), n)

			// Read from back-door to validate that appended content is yet not visible on GCS.
			contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), initialContent, contentBeforeClose)

			// Close() from primary mount to ensure data persists in GCS.
			err = wh.Close()
			require.NoError(t.T(), err)

			// Validate that appended content is visible in GCS.
			expectedContent := initialContent + data
			contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), expectedContent, contentAfterClose)
		}()
	}
}

func (t *SingleMountAppendsSuite) TestAppendsToFinalizedObjectNotVisibleUntilClose() {
	const initialContent = "dummy content"
	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
			// Create Finalized Object in the GCS bucket.
			client.CreateObjectInGCSTestDir(
				ctx, storageClient, testDirName, t.fileName, initialContent, t.T())

			// Append to the finalized object from the primary mount.
			data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
			filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
			fh, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
			require.NoError(t.T(), err)
			_, err = fh.Write([]byte(data))
			require.NoError(t.T(), err)

			// Read from back-door to validate that appended content is yet not visible on GCS.
			contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), initialContent, string(contentBeforeClose))

			// Close the file handle used for appending.
			require.NoError(t.T(), fh.Close())

			// Read from back-door to validate that appended content is now visible on GCS.
			expectedContent := initialContent + data
			contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), expectedContent, string(contentAfterClose))
		}()
	}
}

func (t *SingleMountAppendsSuite) TestAppendsVisibleInRealTimeWithConcurrentRPlusHandle() {
	const initialContent = "dummy content"
	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
			// Open first handle in append mode.
			appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer appendFileHandle.Close()

			// Open second handle in "r+" mode.
			readHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)
			defer readHandle.Close()

			// Write initial content using append handle to trigger BW workflow.
			n, err := appendFileHandle.Write([]byte(initialContent))
			require.NoError(t.T(), err)
			require.NotZero(t.T(), n)

			// Append additional content using "r+" handle.
			data := setup.GenerateRandomString(contentSizeForBW * blockSize)
			appendOffset := int64(unfinalizedObjectSize + len(initialContent))
			_, err = readHandle.WriteAt([]byte(data), appendOffset)
			require.NoError(t.T(), err)

			// Read from back-door to validate visibility on GCS.
			// The first 1MiB block is guaranteed to be flushed due to implicit behavior.
			// That block includes both the initial content (written via "a" file handle )
			// and some part of data written by the "r+" file handle.
			dataInBlockOffset := blockSize - len(initialContent)
			expectedContent := t.fileContent + initialContent + data[0:dataInBlockOffset]
			contentRead, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			// Assert only on the data which is guranteed to have been uploaded to GCS.
			require.GreaterOrEqual(t.T(), len(contentRead), len(expectedContent))
			assert.Equal(t.T(), expectedContent, string(contentRead[0:len(expectedContent)]))
		}()
	}
}

func (t *SingleMountAppendsSuite) TestRandomWritesVisibleAfterCloseWithConcurrentRPlusHandle() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()

	const initialContent = "dummy content"
	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
			// Open first handle in append mode.
			appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer appendFileHandle.Close()

			// Open second handle in "r+" mode.
			readHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)

			// Write initial content using append handle to trigger BW workflow.
			n, err := appendFileHandle.Write([]byte(initialContent))
			require.NoError(t.T(), err)
			require.NotZero(t.T(), n)

			// Random write additional content using "r+" handle by writing to incorrect offset.
			data := setup.GenerateRandomString(contentSizeForBW * blockSize)
			_, err = readHandle.WriteAt([]byte(data), int64(len(initialContent))+1)
			require.NoError(t.T(), err)

			// Read from back-door to validate that appended content is yet not visible on GCS before close().
			contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), initialContent, string(contentBeforeClose))

			// Close the file handle.
			readHandle.Close()

			// Read from back-door to validate that appended content is now visible on GCS after close().
			expectedContent := t.fileContent + initialContent + "\x00" + data
			contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), expectedContent, contentAfterClose)
		}()
	}
}

func (t *SingleMountAppendsSuite) TestFallbackHappensWhenNonAppendHandleDoesFirstWrite() {
	// Skipping test for now until CreateObject() is supported for unfinalized objects.
	// Ref: b/424253611
	t.T().Skip()

	for _, flags := range [][]string{
		{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	} {
		func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", flags)

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			primaryPath := path.Join(t.primaryMount.testDirPath, t.fileName)
			// Open first handle in append mode.
			appendFileHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer appendFileHandle.Close()

			// Open second handle in "r+" mode.
			readHandle := operations.OpenFileInMode(t.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)

			// Append content using "r+" handle.
			data := setup.GenerateRandomString(contentSizeForBW * blockSize)
			n, err := readHandle.WriteAt([]byte(data), int64(len(t.fileContent)))
			require.NoError(t.T(), err)
			assert.NotZero(t.T(), n)

			// Read from back-door to validate that appended content is yet not visible on GCS before close().
			contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), t.fileContent, contentBeforeClose)

			// Close the file handle.
			readHandle.Close()

			// Read from back-door to validate that appended content is now visible on GCS after close().
			expectedContent := t.fileContent + data
			contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, t.fileName))
			require.NoError(t.T(), err)
			assert.Equal(t.T(), expectedContent, contentAfterClose)
		}()
	}
}

func (t *SingleMountAppendsSuite) TestKernelShouldSeeUpdatedSizeOnAppends() {
	const initialContent = "dummy content"
	flags := []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"}
	log.Printf("Running test with flags: %v", flags)

	testCases := []struct {
		name        string
		expireCache bool
	}{
		{
			name:        "validStatCache",
			expireCache: false,
		},
		{
			name:        "expiredStatCache",
			expireCache: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.mountPrimaryMount(flags)
			defer t.unmountPrimaryMount()

			// Initially create an unfinalized object.
			t.createUnfinalizedObject()
			defer t.deleteUnfinalizedObject()

			filePath := path.Join(t.primaryMount.testDirPath, t.fileName)

			// Append to the unfinalized object and close the file handle.
			appendFileHandle := operations.OpenFileInMode(t.T(), filePath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			n, err := appendFileHandle.Write([]byte(initialContent))
			require.NoError(t.T(), err)
			require.NotZero(t.T(), n)
			appendFileHandle.Close()

			// Expire stat cache if required by the test case. By default, stat cache ttl is 1 sec.
			if tc.expireCache {
				time.Sleep(time.Second)
			}

			// stat() the file to assert on the file size as viewed by the kernel.
			expectedFileSize := int64(unfinalizedObjectSize + len(initialContent))
			fileInfo, err := operations.StatFile(filePath)
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), expectedFileSize, (*fileInfo).Size())
		})
	}
}

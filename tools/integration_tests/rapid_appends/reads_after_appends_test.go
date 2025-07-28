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
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// declare a function type for read and verify
type readAndVerifyFunc func(t *testing.T, filePath string, expectedContent []byte)

func readSequentiallyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	readContent, err := operations.ReadFileSequentially(filePath, 1024*1024)

	// For sequential reads, we expect the content to be exactly as expected.
	require.NoErrorf(t, err, "failed to read file %q sequentially: %v", filePath, err)
	require.Equal(t, readContent, expectedContent)
}

func readRandomlyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	file, err := operations.OpenFileAsReadonly(filePath)
	require.NoErrorf(t, err, "failed to open file %q: %v", filePath, err)
	defer operations.CloseFileShouldNotThrowError(t, file)
	if len(expectedContent) == 0 {
		t.SkipNow()
	}
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	fileSize := fileInfo.Size()
	require.GreaterOrEqualf(t, fileSize, int64(len(expectedContent)), "file %q is too small to read %d bytes", filePath, len(expectedContent))

	// Ensure offset and readSize are within bounds of expected content to be read.
	maxOffset := int(len(expectedContent))
	// Limit number of reads if the file is too small.
	numReads := min(maxOffset, 10)
	for i := range numReads {
		offset := rand.IntN(maxOffset)
		readSize := rand.IntN(int(fileSize - int64(offset)))
		// Read minimum 1 byte.
		if readSize < 1 {
			readSize = 1
		}
		buffer := make([]byte, readSize)

		n, err := file.ReadAt(buffer, int64(offset))

		require.NoErrorf(t, err, "Random-read failed at iter#%d to read file %q at [%d, %d): %v", i, filePath, offset, offset+readSize, err)
		require.Equalf(t, buffer[:n], expectedContent[offset:offset+n], "content mismatch in random read at iter#%d at offset [%d, %d): expected %q, got %q", i, offset, offset+readSize, expectedContent[offset:offset+n], buffer[:n])
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonAppendsSuite) TestAppendsAndReads() {
	const metadataCacheTTLSecs = 10
	metadataCacheEnableFlag := fmt.Sprintf("%s%v", "--metadata-cache-ttl-secs=", metadataCacheTTLSecs)
	fileCacheDirFlag := func() string {
		return "--cache-dir=" + getNewEmptyCacheDir(t.primaryMount.rootDir)
	}

	testCases := []struct {
		name          string
		readAndVerify readAndVerifyFunc
	}{
		{
			name:          "SequentialRead",
			readAndVerify: readSequentiallyAndVerify,
		},
		{
			name:          "RandomRead",
			readAndVerify: readRandomlyAndVerify,
		},
	}

	for _, scenario := range []struct {
		enableMetadataCache bool
		enableFileCache     bool
		flags               []string
	}{{
		// all cache disabled
		enableMetadataCache: false,
		enableFileCache:     false,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", "--metadata-cache-ttl-secs=0"},
	}, {
		enableMetadataCache: true,
		enableFileCache:     false,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", metadataCacheEnableFlag},
	}, {
		enableMetadataCache: true,
		enableFileCache:     true,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", metadataCacheEnableFlag, "--file-cache-max-size-mb=-1", fileCacheDirFlag()},
	}, {
		enableMetadataCache: false,
		enableFileCache:     true,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", "--metadata-cache-ttl-secs=0", "--file-cache-max-size-mb=-1", fileCacheDirFlag()},
	}} {
		func() {
			t.mountPrimaryMount(scenario.flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", scenario.flags)

			for _, tc := range testCases {
				t.Run(tc.name, func() {
					// Initially create an unfinalized object.
					t.createUnfinalizedObject()
					defer t.deleteUnfinalizedObject()

					// Open this object as a file for appending on the appropriate mount.
					appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.appendMountPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
					defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

					readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
					for i := range numAppends {
						sizeBeforeAppend := len(t.fileContent)
						t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
						sizeAfterAppend := len(t.fileContent)

						// If metadata cache is enabled, gcsfuse reads up to the cached file size.
						// For same-mount appends/reads, file size is always current.
						// The initial read (i=0) bypasses cache, seeing the latest file size.
						if !scenario.enableMetadataCache || !t.isSyncNeededAfterAppend || (i == 0) {
							tc.readAndVerify(t.T(), readPath, []byte(t.fileContent))
						} else {
							// Read only up to the cached file size (before append).
							tc.readAndVerify(t.T(), readPath, []byte(t.fileContent[:sizeBeforeAppend]))

							// Wait for metadata cache to expire to fetch the latest size for the next read.
							time.Sleep(time.Duration(metadataCacheTTLSecs) * time.Second)
							// Expect read up to the latest file size which is the size after the append.
							tc.readAndVerify(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
						}
					}
				})
			}
		}()
	}
}

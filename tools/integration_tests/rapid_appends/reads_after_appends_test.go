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
	"bytes"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// declare a function type for read and verify
type readAndVerifyFunc func(filePath string, expectedContent []byte) error

func readSequentiallyAndVerify(filePath string, expectedContent []byte) error {
	readContent, err := operations.ReadFileSequentially(filePath, 1024*1024)
	if err != nil {
		return fmt.Errorf("failed to read file %q sequentially: %w", filePath, err)
	}

	// For sequential reads, we expect the content to be exactly as expected.
	if !bytes.Equal(readContent, expectedContent) {
		return fmt.Errorf("Content mismatch in sequential read: expected %q, got %q", string(expectedContent), string(readContent))
	}
	// If the content matches, we return nil to indicate success.
	return nil
}

func readRandomlyAndVerify(filePath string, expectedContent []byte) error {
	file, err := operations.OpenFileAsReadonly(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer file.Close()
	if len(expectedContent) == 0 {
		return nil // Nothing to verify if expected content is empty
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Printf("Error closing file %q: %v", filePath, err)
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil
	}

	// Ensure offset and readSize are within bounds of both actual file and expected content
	maxOffset := int(fileSize)
	if maxOffset > len(expectedContent) {
		maxOffset = len(expectedContent)
	}

	const numReads int = 50
	for range numReads {
		offset := rand.IntN(maxOffset)
		readSize := rand.IntN(int(fileSize - int64(offset))) // Read from actual file
		if readSize == 0 {                                   // Ensure readSize is at least 1 if possible
			if int(fileSize)-offset > 0 {
				readSize = 1
			} else {
				break
			}
		} else if offset+readSize > int(fileSize) { // Adjust readSize if it goes beyond file end
			readSize = int(fileSize) - offset
		}
		buffer := make([]byte, readSize)

		n, err := file.ReadAt(buffer, int64(offset))

		if err != nil {
			return fmt.Errorf("failed to read file %q at offset %d: %w", filePath, offset, err)
		}
		if !bytes.Equal(buffer[:n], expectedContent[offset:offset+n]) {
			return fmt.Errorf("content mismatch in random read at offset %d: expected %q, got %q", offset, expectedContent[offset:offset+n], buffer[:n])
		}
	}
	return nil
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
		isRandomRead  bool
	}{
		{
			name:          "SequentialRead",
			readAndVerify: readSequentiallyAndVerify,
			isRandomRead:  false,
		},
		{
			name:          "RandomRead",
			readAndVerify: readRandomlyAndVerify,
			isRandomRead:  true,
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
					if tc.isRandomRead && !t.isSyncNeededAfterAppend {
						t.T().Skip("Skipping as random read is not supported for ZB right away")
					}
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
							err := tc.readAndVerify(readPath, []byte(t.fileContent))

							require.NoErrorf(t.T(), err, "failed to match full content in non-metadata-cache/single-mount after %v appends: %v", i+1, err)
						} else {
							// Read only up to the cached file size (before append).
							err := tc.readAndVerify(readPath, []byte(t.fileContent[:sizeBeforeAppend]))

							require.NoErrorf(t.T(), err, "failed to match partial content in metadata-cache dual-mount after %v appends: %v", i+1, err)

							// Wait for metadata cache to expire to fetch the latest size for the next read.
							time.Sleep(time.Duration(metadataCacheTTLSecs) * time.Second)
							// Expect read up to the latest file size which is the size after the append.
							err = tc.readAndVerify(readPath, []byte(t.fileContent[:sizeAfterAppend]))

							require.NoErrorf(t.T(), err, "failed to match full content in metadata-cache dual-mount after %v appends: %v", i+1, err)
						}
					}
				})
			}
		}()
	}
}

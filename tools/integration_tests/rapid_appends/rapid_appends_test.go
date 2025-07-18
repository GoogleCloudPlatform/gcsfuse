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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const numAppends = 3             // Number of appends to perform on test file.
const appendSize = 10            // Size in bytes for each append.
const unfinalizedObjectSize = 10 // Size in bytes of initial unfinalized Object.

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

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

	for i := 0; i < 50; i++ {
		if fileSize == 0 {
			break
		}

		// Ensure offset and readSize are within bounds of both actual file and expected content
		maxOffset := int(fileSize)
		if maxOffset > len(expectedContent) {
			maxOffset = len(expectedContent)
		}
		if maxOffset == 0 {
			break
		}

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

// TODO: Split the suite in two suites single mount and multi-mount.
type RapidAppendsSuite struct {
	suite.Suite
	fileName    string
	fileContent string
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) SetupSuite() {
	// Create secondary mount.
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	secondaryMount.testDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *RapidAppendsSuite) TearDownSuite() {
	// Undo secondary mount.
	setup.UnmountGCSFuse(secondaryMount.rootDir)
	// Clean up.
	if t.T().Failed() {
		setup.SetLogFile(secondaryMount.logFilePath)
		log.Printf("Saving secondary mount log file ...")
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())

		setup.SetLogFile(primaryMount.logFilePath)
		log.Println("Saving primary mount log file ...")
		setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	}
}

func (t *RapidAppendsSuite) SetupSubTest() {
	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	// Create unfinalized object.
	t.fileContent = setup.GenerateRandomString(unfinalizedObjectSize)
	client.CreateUnfinalizedObject(ctx, t.T(), storageClient, path.Join(testDirName, t.fileName), t.fileContent)
}

func (t *RapidAppendsSuite) TearDownSubTest() {
	err := os.Remove(path.Join(primaryMount.testDirPath, t.fileName))
	require.NoError(t.T(), err)
}

// appendToFile appends "appendContent" to the given file.
func (t *RapidAppendsSuite) appendToFile(file *os.File, appendContent string) {
	t.T().Helper()
	n, err := file.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileContent += appendContent
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RapidAppendsSuite) TestAppendsAndRead() {
	testCases := []struct {
		name                      string
		readMountPath             string
		syncNeeded                bool
		readAndVerify             readAndVerifyFunc
		ignoreMetadataEnabledCase bool // If true, skip this case when metadata cache is enabled.
	}{
		{
			name:          "seq_read_from_same_mount",
			readMountPath: primaryMount.testDirPath,
			syncNeeded:    false, // Sync is not required when reading from the same mount.
			readAndVerify: readSequentiallyAndVerify,
		},
		{
			name:                      "seq_read_from_different_mount",
			readMountPath:             secondaryMount.testDirPath,
			syncNeeded:                true, // Sync is required for writes to be visible on another mount.
			readAndVerify:             readSequentiallyAndVerify,
			ignoreMetadataEnabledCase: true, // Skip this case when metadata cache is enabled
		},
		{
			name:          "random_read_from_same_mount",
			readMountPath: primaryMount.testDirPath,
			syncNeeded:    false, // Sync is not required when reading from the same mount.
			readAndVerify: readRandomlyAndVerify,
		},
		{
			name:                      "random_read_from_different_mount",
			readMountPath:             secondaryMount.testDirPath,
			syncNeeded:                true, // Sync is required for writes to be visible on another mount.
			readAndVerify:             readRandomlyAndVerify,
			ignoreMetadataEnabledCase: true, // Skip this case when metadata cache is enabled
		},
	}

	for _, tc := range testCases {
		if scenario.enableMetadataCache && tc.ignoreMetadataEnabledCase {
			t.T().Skipf("Skipping test case %q as reading data written by secondary mount might not work if metadata-cache is enabled", tc.name)
		}
		t.Run(tc.name, func() {
			// Open the file for appending on the primary mount.
			appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(primaryMount.testDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)
			readPath := path.Join(tc.readMountPath, t.fileName)
			for range numAppends {
				t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
				// Sync the file if the test case requires it.
				if tc.syncNeeded {
					operations.SyncFile(appendFileHandle, t.T())
				}

				err := tc.readAndVerify(readPath, []byte(t.fileContent))

				require.NoError(t.T(), err)
			}

		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRapidAppendsSuite(t *testing.T) {
	rapidAppendsSuite := new(RapidAppendsSuite)
	suite.Run(t, rapidAppendsSuite)
}

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

// Provides integration tests for long listing directory with Readdirplus
package readdirplus

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
)

func createDirectoryStructureForTest(t *testing.T) {
	t.Helper()

	testDir := setup.SetupTestDirectory(DirForReaddirplusTest)

	// Directory structure
	// testBucket/dirForReaddirplusTest                                                        -- Dir
	// testBucket/dirForReaddirplusTest/file		                                           -- File
	// testBucket/dirForReaddirplusTest/emptySubDirectory                                      -- Dir
	// testBucket/dirForReaddirplusTest/subDirectory                                           -- Dir
	// testBucket/dirForReaddirplusTest/subDirectory/file1                                     -- File

	// Create a file in the test directory.
	filePath := path.Join(testDir, "file")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Create file at %q: %v", testDir, err)
	}
	err = file.Close()
	if err != nil {
		t.Fatalf("Close file at %q: %v", filePath, err)
	}

	// Create an empty subdirectory.
	subDirPath := path.Join(testDir, "emptySubDirectory")
	err = os.Mkdir(subDirPath, 0755)
	if err != nil {
		t.Fatalf("Create empty subdirectory at %q: %v", subDirPath, err)
	}

	// Create a subdirectory with file.
	subDirWithFilesPath := path.Join(testDir, "subDirectory")
	err = os.Mkdir(subDirWithFilesPath, 0755)
	if err != nil {
		t.Fatalf("Create subdirectory with files at %q: %v", subDirWithFilesPath, err)
	}
	filePath = path.Join(subDirWithFilesPath, "file1")
	file, err = os.Create(filePath)
	if err != nil {
		t.Fatalf("Create file in subdirectory at %q: %v", filePath, err)
	}
	err = file.Close()
	if err != nil {
		t.Fatalf("Close file in subdirectory at %q: %v", filePath, err)
	}
}

func TestLongListingWithReaddirplus(t *testing.T) {
	// Create directory structure for testing.
	createDirectoryStructureForTest(t)
	expectedEntries := []struct {
		name  string
		isDir bool
		mode  os.FileMode
	}{
		{name: "emptySubDirectory", isDir: true, mode: os.ModeDir | 0755},
		{name: "file", isDir: false, mode: 0644},
		{name: "subDirectory", isDir: true, mode: os.ModeDir | 0755},
	}

	// Call Readdirplus to list the directory.
	startTime := time.Now()
	entries, err := fusetesting.ReadDirPlusPicky(path.Join(setup.MntDir(), DirForReaddirplusTest))
	endTime := time.Now()

	if err != nil {
		t.Fatalf("ReadDirPlusPicky failed: %v", err)
	}
	// Verify the entries.
	assert.Equal(t, len(expectedEntries), len(entries), "Number of entries mismatch")
	for i, expected := range expectedEntries {
		entry := entries[i]
		assert.Equal(t, expected.name, entry.Name(), "Name mismatch for entry %d", i)
		assert.Equal(t, expected.isDir, entry.IsDir(), "IsDir mismatch for entry %s", entry.Name())
		assert.Equal(t, expected.mode, entry.Mode(), "Mode mismatch for entry %s", entry.Name())
	}
	// Validate logs to check that ReadDirPlus was called and ReadDir, LookUpInode were not called.
	validateLogsForReaddirplus(t, setup.LogFile(), startTime, endTime)
}

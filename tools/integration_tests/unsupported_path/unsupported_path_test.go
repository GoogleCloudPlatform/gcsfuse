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

package unsupported_path

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testDirName = "dirWithUnsupportedPaths"

// UnsupportedPathSuite is a test suite for operations on directories containing paths that
// are unsupported by the GCSFuse file system (but can exist as objects in GCS).
type UnsupportedPathSuite struct {
	suite.Suite
	// The local test dir path for the test bucket.
	testDir string
	// The path in the GCS bucket for this test suite.
	bucketPath string
}

func TestUnsupportedPathSuite(t *testing.T) {
	suite.Run(t, new(UnsupportedPathSuite))
}

func (s *UnsupportedPathSuite) SetupSuite() {
	s.testDir = setup.SetupTestDirectory(DirForUnsupportedPathTests)
	s.bucketPath = path.Join(DirForUnsupportedPathTests, testDirName)
}

func (s *UnsupportedPathSuite) SetupTest() {
	s.createTestObjects()
}

func (s *UnsupportedPathSuite) TearDownTest() {
	require.NoError(s.T(), os.RemoveAll(path.Join(s.testDir, testDirName)))
}

func (s *UnsupportedPathSuite) TearDownSuite() {
	require.NoError(s.T(), os.RemoveAll(s.testDir))
}

// createTestObjects populates the GCS bucket with objects having supported and unsupported names.
func (s *UnsupportedPathSuite) createTestObjects() {
	s.T().Helper()
	// Define objects with names that contain characters or sequences not supported by POSIX file systems.
	unsupportedObjects := []string{
		s.bucketPath + "//unsupported_file1.txt",   // Contains "//" (double slash)
		s.bucketPath + "/./unsupported_file2.txt",  // Contains "/./" (dot segment)
		s.bucketPath + "/../unsupported_file3.txt", // Contains ".." (dot-dot segment)
		s.bucketPath + "/.",                        // Is "."
		s.bucketPath + "/..",                       // Is ".."
		// Nested unsupported paths
		s.bucketPath + "/unsupportedDir1//file2.txt",
		s.bucketPath + "/unsupportedDir1//unsupportedDir2//file3.txt",
	}

	for _, obj := range unsupportedObjects {
		if bucketType == setup.ZonalBucket {
			require.NoError(s.T(), client.CreateFinalizedObjectOnGCS(ctx, storageClient, obj, "unsupported content"))
		} else {
			require.NoError(s.T(), client.CreateObjectOnGCS(ctx, storageClient, obj, "unsupported content"))
		}
	}

	// Create objects with names that are supported and should be visible.
	supportedFile := path.Join(s.bucketPath, "supported_file.txt")
	supportedDir := path.Join(s.bucketPath, "supported_dir") + "/"

	if bucketType == setup.ZonalBucket {
		require.NoError(s.T(), client.CreateFinalizedObjectOnGCS(ctx, storageClient, supportedFile, "content"))
		require.NoError(s.T(), client.CreateFinalizedObjectOnGCS(ctx, storageClient, supportedDir, ""))
	} else {
		require.NoError(s.T(), client.CreateObjectOnGCS(ctx, storageClient, supportedFile, "content"))
		require.NoError(s.T(), client.CreateObjectOnGCS(ctx, storageClient, supportedDir, ""))
	}
}

// --- Test Cases ---
// The core hypothesis is that GCSFuse will hide/ignore objects whose names
// contain unsupported path segments during file system operations.

// TestListDirWithUnsupportedPaths verifies that os.ReadDir only returns supported objects.
func (s *UnsupportedPathSuite) TestListDirWithUnsupportedPaths() {
	localPath := path.Join(s.testDir, testDirName)

	entries, err := os.ReadDir(localPath)

	require.NoError(s.T(), err, "os.ReadDir should succeed on a mounted directory.")
	// Expect only the supported file and supported directory.
	expectedEntriesCount := 3
	assert.Len(s.T(), entries, expectedEntriesCount, "The number of entries should only match supported objects.")
	entryNames := make([]string, len(entries))
	for i, entry := range entries {
		entryNames[i] = entry.Name()
	}
	expectedNames := []string{"supported_dir", "supported_file.txt", "unsupportedDir1"}
	s.Assert().ElementsMatch(expectedNames, entryNames, "Only supported object names should be returned.")
}

// TestCopyDirWithUnsupportedPaths verifies that operations.CopyDir only copies supported objects.
func (s *UnsupportedPathSuite) TestCopyDirWithUnsupportedPaths() {
	destDirName := "copiedDir"
	sourceLocalPath := path.Join(s.testDir, testDirName)
	destLocalPath := path.Join(s.testDir, destDirName)
	defer setup.CleanUpDir(destLocalPath)
	destBucketPath := path.Join(DirForUnsupportedPathTests, destDirName)
	expectedObjectNames := []string{
		path.Join(destBucketPath, "supported_file.txt"),
		path.Join(destBucketPath, "supported_dir") + "/",
		destBucketPath + "/unsupportedDir1/",
	}

	err := operations.CopyDir(sourceLocalPath, destLocalPath)

	require.NoError(s.T(), err, "CopyDir operation should succeed.")
	// List the contents of the destination directory in the GCS bucket (to check actual objects created).
	entries, err := client.ListDirectory(ctx, storageClient, setup.TestBucket(), destBucketPath)
	require.NoError(s.T(), err, "Listing the destination directory in GCS should succeed.")
	// Verify that only supported objects were copied (5 objects).
	assert.Len(s.T(), entries, 3, "The number of copied objects should only match supported objects.")
	s.Assert().ElementsMatch(expectedObjectNames, entries, "The copied object names must match the expected supported names.")
}

// TestRenameDirWithUnsupportedPaths verifies that operations.RenameDir successfully moves the directory
// and its contents, including the unsupported objects which exist in GCS.
func (s *UnsupportedPathSuite) TestRenameDirWithUnsupportedPaths() {
	destDirName := "renamedDir"
	sourceLocalPath := path.Join(s.testDir, testDirName)
	destLocalPath := path.Join(s.testDir, destDirName)
	destBucketPath := path.Join(DirForUnsupportedPathTests, destDirName)
	defer setup.CleanUpDir(destLocalPath)
	// In a rename operation, all GCS objects (supported and unsupported) are moved.
	// The unsupported objects are expected to exist at the new location, though
	// they will likely still be hidden from the fuse mount due to the unsupported
	// path components.
	expectedObjectNames := []string{
		// All objects are expected to be at the new destination path.
		path.Join(destBucketPath, "supported_file.txt"),
		path.Join(destBucketPath, "supported_dir") + "/",
		destBucketPath + "//unsupported_file1.txt",
		destBucketPath + "/./unsupported_file2.txt",
		destBucketPath + "/../unsupported_file3.txt",
		destBucketPath + "/../",
		destBucketPath + "/./",
		destBucketPath + "/.",
		destBucketPath + "/..",
		destBucketPath + "//",
		destBucketPath + "/unsupportedDir1/",
		destBucketPath + "/unsupportedDir1//file2.txt",
		destBucketPath + "/unsupportedDir1//unsupportedDir2//file3.txt",
	}

	err := operations.RenameDir(sourceLocalPath, destLocalPath)

	require.NoError(s.T(), err, "RenameDir operation should succeed.")
	// List all objects under the destination prefix recursively to verify the move.
	entries, err := client.ListDirectory(ctx, storageClient, setup.TestBucket(), destBucketPath)
	require.NoError(s.T(), err, "Listing the destination directory in GCS should succeed.")
	// Verify that ALL GCS objects (supported and unsupported) were moved.
	assert.Len(s.T(), entries, 13, "The number of renamed objects should match all original GCS objects.")
	s.Assert().ElementsMatch(expectedObjectNames, entries, "All GCS objects, including unsupported ones, should be moved.")
}

// TestDeleteDirWithUnsupportedPaths verifies that os.RemoveAll successfully deletes the mounted directory
// and all corresponding GCS objects, including the unsupported ones.
func (s *UnsupportedPathSuite) TestDeleteDirWithUnsupportedPaths() {
	localPath := path.Join(s.testDir, testDirName)

	err := os.RemoveAll(localPath)

	require.NoError(s.T(), err, "os.RemoveAll operation should succeed.")
	// Verify the directory no longer exists in the mounted file system.
	_, err = os.Stat(localPath)
	require.Error(s.T(), err, "os.Stat on the removed directory should fail.")
	assert.True(s.T(), os.IsNotExist(err), "The error should indicate 'no such file or directory'.")
	// EXTRA: Verify all objects are deleted in GCS as well.
	entries, err := client.ListDirectory(ctx, storageClient, setup.TestBucket(), s.bucketPath)
	require.NoError(s.T(), err, "Listing the deleted directory prefix in GCS should succeed.")
	assert.Empty(s.T(), entries, "The GCS directory prefix should be empty after RemoveAll.")
}

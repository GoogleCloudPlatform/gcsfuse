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

package unsupported_objects

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UnsupportedObjectsTest struct {
	suite.Suite
	testDir string
}

func TestUnsupportedObjectsTestSuite(t *testing.T) {
	suite.Run(t, new(UnsupportedObjectsTest))
}

func (t *UnsupportedObjectsTest) SetupTest() {
	t.testDir = setup.SetupTestDirectory(DirForUnsupportedObjectsTests)
	t.createTestObjects()
}

func (t *UnsupportedObjectsTest) TearDownTest() {
	setup.CleanUpDir(t.testDir)
}

func (t *UnsupportedObjectsTest) createTestObjects() {
	// Create objects with supported and unsupported names.
	unsupportedObjects := []string{
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects//unsupported_file1.txt",   // Contains "//"
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/./unsupported_file2.txt",  // Contains "/./"
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/../unsupported_file3.txt", // Contains ".."
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/.",                        // Is "."
		DirForUnsupportedObjectsTests + "/dirWithUnsupportedObjects/..",                       // Is ".."
		DirForUnsupportedObjectsTests + "//leading_slash.txt",                                 // Starts with "/"
	}
	for _, obj := range unsupportedObjects {
		client.CreateObjectOnGCS(ctx, storageClient, obj, "unsupported")
	}
	client.CreateObjectOnGCS(ctx, storageClient,  path.Join(DirForUnsupportedObjectsTests, "dirWithUnsupportedObjects", "supported_file.txt"), "content")
	client.CreateObjectOnGCS(ctx, storageClient, path.Join(DirForUnsupportedObjectsTests, "dirWithUnsupportedObjects", "supported_dir") + "/", "")
}

func (t *UnsupportedObjectsTest) TestListDirWithUnsupportedObjects() {
	// List the directory containing both supported and unsupported objects.
	entries, err := os.ReadDir(path.Join(t.testDir, "dirWithUnsupportedObjects"))

	// Verify that listing succeeds and only returns supported objects.
	require.NoError(t.T(), err)
	assert.Len(t.T(), entries, 2)
	assert.Equal(t.T(), "supported_dir", entries[0].Name())
	assert.Equal(t.T(), "supported_file.txt", entries[1].Name())
}

func (t *UnsupportedObjectsTest) TestCopyDirWithUnsupportedObjects() {
	var expectedObjectNames = []string{
		"dirForUnsupportedObjectsTests/copiedDir/",
		"dirForUnsupportedObjectsTests/copiedDir/supported_file.txt",
		"dirForUnsupportedObjectsTests/copiedDir/supported_dir/",
	}
	// Copy the directory containing both supported and unsupported objects.
	err := operations.CopyDir(path.Join(t.testDir, "dirWithUnsupportedObjects"), path.Join(t.testDir, "copiedDir"))

	// Verify that listing succeeds and only returns supported objects.
	require.NoError(t.T(), err)
	// List the destination directory.
	entries, err := client.ListDirectory(ctx, storageClient, setup.TestBucket(), path.Join(DirForUnsupportedObjectsTests, "copiedDir"))
	// Verify that only supported objects are copied.
	require.NoError(t.T(), err)
	assert.Len(t.T(), entries, 3)
	t.Assert().ElementsMatch(expectedObjectNames, entries)
}

func (t *UnsupportedObjectsTest) TestRenameDirWithUnsupportedObjects() {
	var expectedObjectNames = []string{
		"dirForUnsupportedObjectsTests/renamedDir/",
		"dirForUnsupportedObjectsTests/renamedDir/.",
		"dirForUnsupportedObjectsTests/renamedDir/..",
		"dirForUnsupportedObjectsTests/renamedDir/supported_file.txt",
		"dirForUnsupportedObjectsTests/renamedDir/../",
		"dirForUnsupportedObjectsTests/renamedDir/./",
		"dirForUnsupportedObjectsTests/renamedDir//",
		"dirForUnsupportedObjectsTests/renamedDir/supported_dir/",
	}
	// Rename the directory containing both supported and unsupported objects.
	err := operations.RenameDir(path.Join(t.testDir, "dirWithUnsupportedObjects"), path.Join(t.testDir, "renamedDir"))

	// Verify that listing succeeds and only returns supported objects.
	require.NoError(t.T(), err)
	// List the destination directory.
	entries, err := client.ListDirectory(ctx, storageClient, setup.TestBucket(), path.Join(DirForUnsupportedObjectsTests, "renamedDir"))
	// Verify that only supported objects are copied.
	require.NoError(t.T(), err)
	assert.Len(t.T(), entries, 8)
	t.Assert().ElementsMatch(expectedObjectNames, entries)
}

func (t *UnsupportedObjectsTest) TestDeleteDirWithUnsupportedObjects() {
	// Remove the directory containing both supported and unsupported objects.
	err := os.RemoveAll(path.Join(t.testDir, "dirWithUnsupportedObjects"))

	// Verify that listing succeeds and only returns supported objects.
	require.NoError(t.T(), err)
	// List the destination directory.
	_, err = os.Stat(path.Join(t.testDir, "dirWithUnsupportedObjects"))
	// Verify that only supported objects are copied.
	require.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

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

// A collection of tests to check Readdirplus operation.
package fs_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
)

////////////////////////////////////////////////////////////////////////
// ReadDirPlusTest
////////////////////////////////////////////////////////////////////////

type ReadDirPlusTest struct {
	suite.Suite
	fsTest
}

func TestReadDirPlusTestSuite(t *testing.T) {
	suite.Run(t, new(ReadDirPlusTest))
}

func (t *ReadDirPlusTest) SetupSuite() {
	t.mountCfg.EnableReaddirplus = true
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.InodeAttributeCacheTTL = 500 * time.Millisecond
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			ExperimentalEnableDentryCache: true,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *ReadDirPlusTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *ReadDirPlusTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *ReadDirPlusTest) TestEmptyDirectory() {
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 0, len(entries))
}

func (t *ReadDirPlusTest) TestDirectoryWithVariousEntryTypes() {
	// Set up contents.
	assert.Nil(t.T(), t.createObjects(
		map[string]string{
			"file.txt":          "taco",
			"dir/":              "",
			"dir/baz":           "burrito",
			"implicit/file.txt": "content",
		}))
	// Set up a symlink.
	err := os.Symlink("file.txt", path.Join(mntDir, "symlink_to_file"))
	assert.Nil(t.T(), err)
	expectedEntries := []string{
		"dir",
		"file.txt",
		"implicit",
		"symlink_to_file",
	}

	// Read the directory.
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), len(expectedEntries), len(entries))
	// Check entries.
	for i, entry := range entries {
		assert.Equal(t.T(), expectedEntries[i], entry.Name())
		switch entry.Name() {
		case "file.txt":
			assert.False(t.T(), entry.IsDir())
			assert.Equal(t.T(), int64(len("taco")), entry.Size())
			assert.Equal(t.T(), filePerms, entry.Mode())
		case "dir":
			assert.True(t.T(), entry.IsDir())
			assert.Equal(t.T(), dirPerms|os.ModeDir, entry.Mode())
		case "implicit":
			assert.True(t.T(), entry.IsDir())
			assert.Equal(t.T(), dirPerms|os.ModeDir, entry.Mode())
		case "symlink_to_file":
			assert.False(t.T(), entry.IsDir())
			assert.Equal(t.T(), os.ModeSymlink, entry.Mode()&os.ModeType)
		default:
			assert.FailNow(t.T(), "unexpected entry: %s", entry.Name())
		}
	}
}

// Test that stat after Readdirplus return the same attributes as Readdirplus even if data on GCS has changed
func (t *ReadDirPlusTest) TestStatAfterReaddirplus() {
	// Set up contents.
	testFileName := "file.txt"
	filePath := path.Join(mntDir, testFileName)
	initialContent := generateRandomString(10)
	updatedContent := generateRandomString(5)
	assert.Nil(t.T(), t.createObjects(map[string]string{testFileName: initialContent}))

	// Read the directory with Readdirplus.
	_, _ = fusetesting.ReadDirPlusPicky(mntDir)
	// Modify the file content in GCS.
	assert.Nil(t.T(), t.createObjects(map[string]string{testFileName: updatedContent}))
	// Stat the file before entry expires in cache.
	// This should return the same attributes as Readdirplus.
	fileInfo, err := os.Stat(filePath)

	// Check that the stat returns the old attributes.
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), testFileName, fileInfo.Name())
	assert.Equal(t.T(), int64(len(initialContent)), fileInfo.Size())
	// Check stat after cache expiry.
	// Wait for a duration longer than the metadata cache TTL.
	time.Sleep(time.Second)
	// Stat the file again.
	// This should return the updated attributes.
	fileInfo, err = os.Stat(filePath)
	// Check that the stat returns the updated attributes.
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), testFileName, fileInfo.Name())
	assert.Equal(t.T(), int64(len(updatedContent)), fileInfo.Size())
}

////////////////////////////////////////////////////////////////////////
// LocalFileEntriesReadDirPlusTest
////////////////////////////////////////////////////////////////////////

type LocalFileEntriesReadDirPlusTest struct {
	suite.Suite
	fsTest
}

func TestLocalFileEntriesReadDirPlusTestSuite(t *testing.T) {
	suite.Run(t, new(LocalFileEntriesReadDirPlusTest))
}

func (t *LocalFileEntriesReadDirPlusTest) SetupSuite() {
	t.mountCfg.EnableReaddirplus = true
	t.serverCfg.InodeAttributeCacheTTL = 60 * time.Second
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			ExperimentalEnableDentryCache: true,
		},
		Write: cfg.WriteConfig{
			CreateEmptyFile: false,
		}}
	t.fsTest.SetUpTestSuite()
}

func (t *LocalFileEntriesReadDirPlusTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *LocalFileEntriesReadDirPlusTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *LocalFileEntriesReadDirPlusTest) TestDirectoryWithLocalFile() {
	// Create a local file that is not yet synced to GCS.
	f, err := os.Create(path.Join(mntDir, "local_file"))
	assert.Nil(t.T(), err)
	defer f.Close()

	// Read the directory.
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 1, len(entries))
	// Check the entry for the local file.
	entry := entries[0]
	assert.Equal(t.T(), "local_file", entry.Name())
	assert.False(t.T(), entry.IsDir())
	assert.Equal(t.T(), filePerms, entry.Mode())
}

func (t *LocalFileEntriesReadDirPlusTest) TestDirWithOneLocalAndOneGCSEntry() {
	// Create a remote object on GCS
	assert.Nil(t.T(), t.createObjects(map[string]string{"gcs_file": "content"}))
	// Create a local-only file
	f, err := os.Create(path.Join(mntDir, "local_file"))
	assert.Nil(t.T(), err)
	defer f.Close()

	// Read the directory
	entries, err := fusetesting.ReadDirPlusPicky(mntDir)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(entries))
	assert.Equal(t.T(), "gcs_file", entries[0].Name())
	assert.False(t.T(), entries[0].IsDir())
	assert.Equal(t.T(), int64(len("content")), entries[0].Size())
	assert.Equal(t.T(), "local_file", entries[1].Name())
	assert.False(t.T(), entries[1].IsDir())
}

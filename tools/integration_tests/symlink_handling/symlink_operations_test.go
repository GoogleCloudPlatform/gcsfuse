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

package symlink_handling

import (
	"os"
	"os/exec"
	"path"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// TestCreateSymlink tests the creation of symlinks.
func (s *BaseSymlinkSuite) TestCreateSymlink() {
	// Create the symlink
	_ = s.createSymlink(s.linkName, s.targetPath)

	// Validate the underlying GCS Object
	s.validateBackingGCSObjectForSymlink(s.linkName, s.targetPath, s.isStandardSymlink)
}

// TestReadSymlinkTest tests reading a symlink's target.
func (s *BaseSymlinkSuite) TestReadSymlinkTest() {
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	result, err := os.Readlink(linkPath)

	s.Require().NoError(err)
	s.Assert().Equal(s.targetPath, result)
}

// TestReadFileViaSymlink tests reading a file through a symlink.
func (s *BaseSymlinkSuite) TestReadFileViaSymlink() {
	const content = "hello world"
	// Create a target file with content.
	err := os.WriteFile(s.targetPath, []byte(content), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	// Read file via symlink.
	readContent, err := os.ReadFile(linkPath)
	s.Require().NoError(err)

	// Verify content.
	s.Assert().Equal(content, string(readContent))
}

// TestWriteFileViaSymlink tests writing to a file through a symlink.
func (s *BaseSymlinkSuite) TestWriteFileViaSymlink() {
	const content = "new content"
	// Create an empty target file.
	f, err := os.Create(s.targetPath)
	s.Require().NoError(err)
	s.Require().NoError(f.Close())
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	// Write to file via symlink.
	err = os.WriteFile(linkPath, []byte(content), 0644)
	s.Require().NoError(err)

	// Verify content of the original file.
	readContent, err := os.ReadFile(s.targetPath)
	s.Require().NoError(err)
	s.Assert().Equal(content, string(readContent))
}

// TestListDirViaSymlink tests listing a directory through a symlink.
func (s *BaseSymlinkSuite) TestListDirViaSymlink() {
	fileName := "file_in_dir.txt"
	// Create a target directory with a file.
	err := os.Mkdir(s.targetPath, 0755)
	s.Require().NoError(err)
	filePath := path.Join(s.targetPath, fileName)
	err = os.WriteFile(filePath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target directory.
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	// List directory via symlink.
	entries, err := os.ReadDir(linkPath)

	s.Require().NoError(err)
	// Verify contents.
	s.Assert().Len(entries, 1)
	s.Assert().Equal(fileName, entries[0].Name())
}

// TestRenameSymlink tests renaming a symlink.
func (s *BaseSymlinkSuite) TestRenameSymlink() {
	newLinkName := s.linkName + "_renamed"
	// Create a target file.
	err := os.WriteFile(s.targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	newLinkPath := path.Join(s.testDirPath, newLinkName)

	// Rename the symlink.
	err = os.Rename(linkPath, newLinkPath)
	s.Require().NoError(err)

	// Verify old link is gone.
	_, err = os.Lstat(linkPath)
	s.Assert().True(os.IsNotExist(err))
	// Verify new link exists, is a symlink, and points to the correct target.
	fi, err := os.Lstat(newLinkPath)
	s.Require().NoError(err)
	s.Assert().True(fi.Mode()&os.ModeSymlink != 0)
	readTargetName, err := os.Readlink(newLinkPath)
	s.Require().NoError(err)
	s.Assert().Equal(s.targetPath, readTargetName)
	// Verify target file is untouched.
	_, err = os.Stat(s.targetPath)
	s.Assert().NoError(err)
}

// TestCopySymlink tests copying a symlink without dereferencing.
func (s *BaseSymlinkSuite) TestCopySymlink() {
	newLinkName := s.linkName + "_copied"
	// Create a target file.
	err := os.WriteFile(s.targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)
	newLinkPath := path.Join(s.testDirPath, newLinkName)

	// Copy the symlink using cp -P to ensure no dereferencing.
	cmd := exec.Command("cp", "-P", linkPath, newLinkPath)
	err = cmd.Run()
	s.Require().NoError(err)

	// Verify old link still exists.
	_, err = os.Lstat(linkPath)
	s.Assert().NoError(err)
	// Verify new link exists, is a symlink, and points to the correct target.
	fi, err := os.Lstat(newLinkPath)
	s.Require().NoError(err)
	s.Assert().True(fi.Mode()&os.ModeSymlink != 0)
	readTargetName, err := os.Readlink(newLinkPath)
	s.Require().NoError(err)
	s.Assert().Equal(s.targetPath, readTargetName)
	// Verify target file is untouched.
	_, err = os.Stat(s.targetPath)
	s.Assert().NoError(err)
}

// TestReadStandardSymlinkInLegacyMode tests that a legacy mount can read a standard symlink.
func (s *LegacySymlinksTestSuite) TestReadStandardSymlinkInLegacyMode() {
	// Temporarily enable standard symlink creation to create a standard symlink object.
	s.isStandardSymlink = true
	defer func() { s.isStandardSymlink = false }()
	s.createGCSSymlinkObject(s.linkName, s.targetPath)
	linkPath := path.Join(s.testDirPath, s.linkName)

	// Read the symlink via the legacy mount.
	result, err := os.Readlink(linkPath)

	s.Require().NoError(err)
	s.Assert().Equal(s.targetPath, result)
}

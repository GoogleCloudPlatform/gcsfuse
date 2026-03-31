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

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// TestCreateSymlink tests the creation of symlinks.
func (s *BaseSymlinkSuite) TestCreateSymlink() {
	target := s.createTempFile()
	linkName := setup.GenerateRandomString(5)

	// Create the symlink
	_ = s.createSymlink(linkName, target)

	// Validate the underlying GCS Object
	s.validateBackingGCSObjectForSymlink(linkName, target, s.isStandardSymlink)
}

// TestReadSymlinkTest tests reading a symlink's target.
func (s *BaseSymlinkSuite) TestReadSymlinkTest() {
	linkName := setup.GenerateRandomString(5)
	target := "target_file_path"
	s.createGCSSymlinkObject(linkName, target)
	linkPath := path.Join(s.testDirPath, linkName)

	result, err := os.Readlink(linkPath)

	s.Require().NoError(err)
	s.Assert().Equal(target, result)
}

// TestReadFileViaSymlink tests reading a file through a symlink.
func (s *BaseSymlinkSuite) TestReadFileViaSymlink() {
	prefix := setup.GenerateRandomString(5)
	const content = "hello world"
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	// Create a target file with content.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte(content), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)

	// Read file via symlink.
	readContent, err := os.ReadFile(linkPath)
	s.Require().NoError(err)

	// Verify content.
	s.Assert().Equal(content, string(readContent))
}

// TestWriteFileViaSymlink tests writing to a file through a symlink.
func (s *BaseSymlinkSuite) TestWriteFileViaSymlink() {
	prefix := setup.GenerateRandomString(5)
	const content = "new content"
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	// Create an empty target file.
	targetPath := path.Join(s.testDirPath, targetName)
	f, err := os.Create(targetPath)
	s.Require().NoError(err)
	s.Require().NoError(f.Close())
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)

	// Write to file via symlink.
	err = os.WriteFile(linkPath, []byte(content), 0644)
	s.Require().NoError(err)

	// Verify content of the original file.
	readContent, err := os.ReadFile(targetPath)
	s.Require().NoError(err)
	s.Assert().Equal(content, string(readContent))
}

// TestListDirViaSymlink tests listing a directory through a symlink.
func (s *BaseSymlinkSuite) TestListDirViaSymlink() {
	prefix := setup.GenerateRandomString(5)
	targetDirName := prefix + "target_dir"
	linkName := prefix + "link"
	fileName := "file_in_dir.txt"
	// Create a target directory with a file.
	targetDirPath := path.Join(s.testDirPath, targetDirName)
	err := os.Mkdir(targetDirPath, 0755)
	s.Require().NoError(err)
	filePath := path.Join(targetDirPath, fileName)
	err = os.WriteFile(filePath, []byte(""), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target directory.
	s.createGCSSymlinkObject(linkName, targetDirName)
	linkPath := path.Join(s.testDirPath, linkName)

	// List directory via symlink.
	entries, err := os.ReadDir(linkPath)

	s.Require().NoError(err)
	// Verify contents.
	s.Assert().Len(entries, 1)
	s.Assert().Equal(fileName, entries[0].Name())
}

// TestRenameSymlink tests renaming a symlink.
func (s *BaseSymlinkSuite) TestRenameSymlink() {
	prefix := setup.GenerateRandomString(5)
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	newLinkName := prefix + "new_link"
	// Create a target file.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)
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
	readTarget, err := os.Readlink(newLinkPath)
	s.Require().NoError(err)
	s.Assert().Equal(targetName, readTarget)
	// Verify target file is untouched.
	_, err = os.Stat(targetPath)
	s.Assert().NoError(err)
}

// TestCopySymlink tests copying a symlink without dereferencing.
func (s *BaseSymlinkSuite) TestCopySymlink() {
	prefix := setup.GenerateRandomString(5)
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	newLinkName := prefix + "new_link"
	// Create a target file.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	s.createGCSSymlinkObject(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)
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
	readTarget, err := os.Readlink(newLinkPath)
	s.Require().NoError(err)
	s.Assert().Equal(targetName, readTarget)
	// Verify target file is untouched.
	_, err = os.Stat(targetPath)
	s.Assert().NoError(err)
}

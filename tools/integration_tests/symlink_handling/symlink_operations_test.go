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

type symlinkTestCase struct {
	name   string
	target string
}

var commonTestCases = []symlinkTestCase{
	{
		name:   "file_target",
		target: "target_file",
	},
	{
		name:   "dir_target",
		target: "target_dir",
	},
	{
		name:   "relative_path",
		target: "../target_file",
	},
	{
		name:   "absolute_path",
		target: "/etc/hosts",
	},
}

////////////////////////////////////////////////////////////////////////
// Common test methods
////////////////////////////////////////////////////////////////////////

// runReadSymlinkTests tests reading a symlink's target.
func (s *BaseSymlinkSuite) runReadSymlinkTests(testCases []symlinkTestCase, prefix string, createFunc func(string, string)) {
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			linkName := prefix + tc.name
			createFunc(linkName, tc.target)

			linkPath := path.Join(s.testDirPath, linkName)
			result, err := os.Readlink(linkPath)
			s.Require().NoError(err)
			s.Assert().Equal(tc.target, result)
		})
	}
}

// testReadFileViaSymlink tests reading a file through a symlink.
func (s *BaseSymlinkSuite) testReadFileViaSymlink(prefix string, createSymlinkFunc func(linkName, targetName string)) {
	const content = "hello world"
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	// Create a target file with content.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte(content), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	createSymlinkFunc(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)

	// Read file via symlink.
	readContent, err := os.ReadFile(linkPath)
	s.Require().NoError(err)

	// Verify content.
	s.Assert().Equal(content, string(readContent))
}

// testWriteFileViaSymlink tests writing to a file through a symlink.
func (s *BaseSymlinkSuite) testWriteFileViaSymlink(prefix string, createSymlinkFunc func(linkName, targetName string)) {
	const content = "new content"
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	// Create an empty target file.
	targetPath := path.Join(s.testDirPath, targetName)
	f, err := os.Create(targetPath)
	s.Require().NoError(err)
	s.Require().NoError(f.Close())
	// Create a symlink to the target file.
	createSymlinkFunc(linkName, targetName)
	linkPath := path.Join(s.testDirPath, linkName)

	// Write to file via symlink.
	err = os.WriteFile(linkPath, []byte(content), 0644)
	s.Require().NoError(err)

	// Verify content of the original file.
	readContent, err := os.ReadFile(targetPath)
	s.Require().NoError(err)
	s.Assert().Equal(content, string(readContent))
}

// testListDirViaSymlink tests listing a directory through a symlink.
func (s *BaseSymlinkSuite) testListDirViaSymlink(prefix string, createSymlinkFunc func(linkName, targetName string)) {
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
	createSymlinkFunc(linkName, targetDirName)
	linkPath := path.Join(s.testDirPath, linkName)

	// List directory via symlink.
	entries, err := os.ReadDir(linkPath)

	s.Require().NoError(err)
	// Verify contents.
	s.Assert().Len(entries, 1)
	s.Assert().Equal(fileName, entries[0].Name())
}

// testRenameSymlink tests renaming a symlink.
func (s *BaseSymlinkSuite) testRenameSymlink(prefix string, createSymlinkFunc func(linkName, targetName string)) {
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	newLinkName := prefix + "new_link"
	// Create a target file.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	createSymlinkFunc(linkName, targetName)
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

// testCopySymlink tests copying a symlink without dereferencing.
func (s *BaseSymlinkSuite) testCopySymlink(prefix string, createSymlinkFunc func(linkName, targetName string)) {
	targetName := prefix + "target.txt"
	linkName := prefix + "link"
	newLinkName := prefix + "new_link"
	// Create a target file.
	targetPath := path.Join(s.testDirPath, targetName)
	err := os.WriteFile(targetPath, []byte("content"), 0644)
	s.Require().NoError(err)
	// Create a symlink to the target file.
	createSymlinkFunc(linkName, targetName)
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

////////////////////////////////////////////////////////////////////////
// Test cases for Standard Symlinks
////////////////////////////////////////////////////////////////////////

func (s *StandardSymlinksTestSuite) TestCreateSymlink() {
	target := s.createTempFile()
	linkName := "standard_symlink"

	// Create the symlink
	_ = s.createSymlink(linkName, target)

	// Validate the underlying GCS Object
	s.validateBackingGCSObjectForSymlink(linkName, target, true)
}

func (s *StandardSymlinksTestSuite) TestReadSymlink() {
	s.runReadSymlinkTests(commonTestCases, "read_standard_symlink_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *StandardSymlinksTestSuite) TestReadFileViaSymlink() {
	s.testReadFileViaSymlink("read_standard_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *StandardSymlinksTestSuite) TestWriteFileViaSymlink() {
	s.testWriteFileViaSymlink("write_standard_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *StandardSymlinksTestSuite) TestListDirViaSymlink() {
	s.testListDirViaSymlink("listdir_standard_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *StandardSymlinksTestSuite) TestRenameSymlink() {
	s.testRenameSymlink("rename_standard_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

func (s *StandardSymlinksTestSuite) TestCopySymlink() {
	s.testCopySymlink("copy_standard_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, target, map[string]string{
			StandardSymlinkMetadataKey: "true",
		})
	})
}

////////////////////////////////////////////////////////////////////////
// Test cases for legacy Symlinks
////////////////////////////////////////////////////////////////////////

func (s *LegacySymlinksTestSuite) TestCreateSymlink() {
	target := s.createTempFile()
	linkName := "legacy_symlink"

	// Create the symlink
	_ = s.createSymlink(linkName, target)

	// Validate the underlying GCS Object
	s.validateBackingGCSObjectForSymlink(linkName, target, false)
}

func (s *LegacySymlinksTestSuite) TestReadSymlink() {
	s.runReadSymlinkTests(commonTestCases, "read_legacy_symlink_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

func (s *LegacySymlinksTestSuite) TestReadFileViaSymlink() {
	s.testReadFileViaSymlink("read_legacy_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

func (s *LegacySymlinksTestSuite) TestWriteFileViaSymlink() {
	s.testWriteFileViaSymlink("write_legacy_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

func (s *LegacySymlinksTestSuite) TestListDirViaSymlink() {
	s.testListDirViaSymlink("listdir_legacy_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

func (s *LegacySymlinksTestSuite) TestRenameSymlink() {
	s.testRenameSymlink("rename_legacy_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

func (s *LegacySymlinksTestSuite) TestCopySymlink() {
	s.testCopySymlink("copy_legacy_", func(linkName, target string) {
		s.createGCSSymlinkObject(linkName, "", map[string]string{
			SymlinkMetadataKey: target,
		})
	})
}

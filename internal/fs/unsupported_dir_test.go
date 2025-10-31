// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A collection of tests for a file system for unsupported object names.
package fs_test

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UnsupportedObjectNameTest struct {
	suite.Suite
	fsTest
}

func TestUnsupportedObjectNameTestSuite(t *testing.T) {
	suite.Run(t, new(UnsupportedObjectNameTest))
}

func (t *UnsupportedObjectNameTest) SetupTest() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		EnableUnsupportedDirSupport: true,
		EnableAtomicRenameObject:    true,
	}
	t.fsTest.SetUpTestSuite()
}

func (t *UnsupportedObjectNameTest) TearDownTest() {
	t.fsTest.TearDown()
	t.fsTest.TearDownTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *UnsupportedObjectNameTest) TestReadDirectory_WithUnsupportedNames() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"dir1/sub_dir1//file1": "",
		"dir1/sub_dir1/file2":  "content",
		"dir2//file3":          "",
		"dir2/file4":           "content",
		"dir3/./file5":         "content",
		"file6":                "content",
		"//a.txt":              "",
		"./b.txt":              "",
	})
	t.Require().NoError(err)
	var files []string
	var dirs []string

	// Walk the mounted directory.
	err = filepath.Walk(mntDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == mntDir {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, info.Name())
		} else {
			files = append(files, info.Name())
		}
		return nil
	})

	// ReadDir should only show the supported object.
	t.Require().NoError(err)
	t.Assert().ElementsMatch([]string{"dir1", "dir2", "dir3", "sub_dir1"}, dirs)
	t.Assert().ElementsMatch([]string{"file2", "file4", "file6"}, files)
}

func (t *UnsupportedObjectNameTest) TestCopyDirectory_WithUnsupportedNames() {
	err := t.createObjects(map[string]string{
		"src/file1":    "content1",
		"src//file2":   "content2",
		"src/./file3":  "content3",
		"src/ok":       "content4",
		"src/../file5": "content5",
		"src///file5":  "content6",
	})
	t.Require().NoError(err)
	srcPath := path.Join(mntDir, "src")
	destPath := path.Join(mntDir, "dest")

	// Execute copy command.
	cmd := exec.Command("cp", "-r", srcPath, destPath)
	err = cmd.Run()

	t.Require().NoError(err)
	// Verify the contents of the destination directory.
	entries, err := os.ReadDir(destPath)
	t.Require().NoError(err)
	// Only supported files and directories should be copied.
	t.Require().Len(entries, 2)
	t.Assert().Equal("file1", entries[0].Name())
	t.Assert().Equal("ok", entries[1].Name())
}

func (t *UnsupportedObjectNameTest) TestRenameDirectory_WithUnsupportedNames() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"src/file1":    "content1",
		"src//file2":   "content2",
		"src/./file3":  "content3",
		"src/ok/file4": "content4",
	})
	t.Require().NoError(err)
	srcPath := path.Join(mntDir, "src")
	destPath := path.Join(mntDir, "dest")

	// Attempt to rename the directory.
	err = os.Rename(srcPath, destPath)
	t.Require().NoError(err)

	// The old path should not exist.
	_, err = os.Stat(srcPath)
	t.Assert().True(os.IsNotExist(err))
	// Verify the contents of the destination directory.
	entries, err := os.ReadDir(destPath)
	t.Require().NoError(err)
	// Only supported files and directories are visible during list.
	t.Require().Len(entries, 2)
	t.Assert().Equal("file1", entries[0].Name())
	t.Assert().Equal("ok", entries[1].Name())
}

func (t *UnsupportedObjectNameTest) TestDeleteDirectory_WithUnsupportedNames() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"dir_to_delete/file1":    "content1",
		"dir_to_delete//file2":   "content2",
		"dir_to_delete/./file3":  "content3",
		"dir_to_delete/ok":       "content4",
		"dir_to_delete/../file5": "content5",
		"dir_to_delete///file6":  "content6",
	})
	t.Require().NoError(err)
	dirPath := path.Join(mntDir, "dir_to_delete")
	// Verify that listing only shows supported files.
	entries, err := os.ReadDir(dirPath)
	t.Require().NoError(err)
	t.Require().Len(entries, 2)
	t.Assert().Equal("file1", entries[0].Name())
	t.Assert().Equal("ok", entries[1].Name())

	// Execute rm -rf command.
	cmd := exec.Command("rm", "-rf", dirPath)
	err = cmd.Run()

	t.Require().NoError(err)
	_, err = os.Stat(dirPath)
	t.Assert().Error(err)
	t.Assert().True(strings.Contains(err.Error(), "no such file or directory"))
}

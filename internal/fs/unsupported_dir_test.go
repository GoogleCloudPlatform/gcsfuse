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
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/jacobsa/fuse/fusetesting"
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

func (t *UnsupportedObjectNameTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		EnableUnsupportedDirSupport: true,
	}
	t.fsTest.SetUpTestSuite()
}

func (t *UnsupportedObjectNameTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *UnsupportedObjectNameTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *UnsupportedObjectNameTest) TestReadDir_UnsupportedObjectName() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"foo//bar": "",
	})
	t.Require().NoError(err)

	// ReadDir should not show the unsupported object.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	t.Require().NoError(err)
	t.Assert().Empty(entries)
}

func (t *UnsupportedObjectNameTest) TestReadDir_UnsupportedObjectName_WithSupportedObjects() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"//a.txt":     "",
		"foo//bar":    "",
		"foo/../bar1": "",
		"foo/./bar2":  "",
		"foo//./bar3": "",
		"qux":         "taco",
	})
	t.Require().NoError(err)

	// ReadDir should only show the supported object.
	entries, err := fusetesting.ReadDirPicky(mntDir)
	t.Require().NoError(err)
	t.Require().Len(entries, 2)
	t.Assert().Equal("foo", entries[0].Name())
	t.Assert().Equal("qux", entries[1].Name())
}

func (t *UnsupportedObjectNameTest) TestListSubDirectory_WithUnsupportedNames() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"dir/sub_dir//file1": "",
		"dir/sub_dir/file2":  "content",
	})
	t.Require().NoError(err)

	// Listing the parent directory 'dir' should show 'sub_dir'.
	entries, err := fusetesting.ReadDirPicky(path.Join(mntDir, "dir"))
	t.Require().NoError(err)
	t.Require().Len(entries, 1)
	t.Assert().Equal("sub_dir", entries[0].Name())
	t.Assert().True(entries[0].IsDir())

	// Listing 'sub_dir' should only show the supported file.
	subDirEntries, err := fusetesting.ReadDirPicky(path.Join(mntDir, "dir/sub_dir"))
	t.Require().NoError(err)
	t.Require().Len(subDirEntries, 1)
	t.Assert().Equal("file2", subDirEntries[0].Name())
	t.Assert().False(subDirEntries[0].IsDir())
}

func (t *UnsupportedObjectNameTest) TestRecursiveList_WithUnsupportedNames() {
	// Set up contents.
	err := t.createObjects(map[string]string{
		"dir1/sub_dir1//file1": "",
		"dir1/sub_dir1/file2":  "content",
		"dir2//file3":          "",
		"dir2/file4":           "content",
		"dir3/./file5":         "content",
		"file6":                "content",
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

	t.Require().NoError(err)
	t.Assert().ElementsMatch([]string{"dir1", "dir2", "dir3", "sub_dir1"}, dirs)
	t.Assert().ElementsMatch([]string{"file2", "file4", "file6"}, files)
}

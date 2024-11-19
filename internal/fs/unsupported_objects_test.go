// Copyright 2024 Google LLC
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

package fs_test

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UnsupportedObjectsTest struct {
	suite.Suite
	fsTest
}

func (t *UnsupportedObjectsTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.fsTest.SetUpTestSuite()
}

func (t *UnsupportedObjectsTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *UnsupportedObjectsTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestUnsupportedObjectsTestSuite(t *testing.T) {
	suite.Run(t, new(UnsupportedObjectsTest))
}

type UnsupportedObjectsTestWithoutImplicitDirs struct {
	UnsupportedObjectsTest
}

func (t *UnsupportedObjectsTestWithoutImplicitDirs) SetupSuite() {
	t.serverCfg.ImplicitDirectories = false
	t.fsTest.SetUpTestSuite()
}
func TestUnsupportedObjectsWithoutImplicitDirsTestSuite(t *testing.T) {
	suite.Run(t, new(UnsupportedObjectsTestWithoutImplicitDirs))
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// /////////////////////////////////////////////////////////////////////
func verifyInvalidPath(t *UnsupportedObjectsTest, path string) {
	t.T().Helper()

	_, err := os.Stat(path)

	assert.Errorf(t.T(), err, "Failed to get error in stat of %q", path)
}

type expectedWalkedEntry struct {
	path  string
	name  string
	isDir bool
	found bool
}

// Create objects with unsupported object names and
// verify the behavior of mount using os.Stat and WalkDir.
func (t *UnsupportedObjectsTest) testGcsObjects(objectNames map[string]string, invalidPaths []string, expectedWalkedEntries []expectedWalkedEntry) {
	// Set up contents.
	assert.NoError(
		t.T(), t.createObjects(objectNames))

	// Verify that the unsupported objects fail os.Stat.
	for _, invalidPath := range invalidPaths {
		verifyInvalidPath(t, path.Join(mntDir, invalidPath))
	}

	// Verify that the supported objects appear in WalkDir.
	assert.Nil(t.T(), filepath.WalkDir(mntDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		foundMatchingExpectedWalkingEntry := false
		for i := range expectedWalkedEntries {
			expectedWalkedEntry := &expectedWalkedEntries[i]
			if expectedWalkedEntry.path == path && expectedWalkedEntry.name == d.Name() && d.IsDir() == expectedWalkedEntry.isDir {
				if expectedWalkedEntry.found {
					return fmt.Errorf("found duplicate walked entry: path=%s, name=%s, isDir=%v", path, d.Name(), d.IsDir())
				}

				foundMatchingExpectedWalkingEntry = true
				expectedWalkedEntry.found = true
			}
		}

		if !foundMatchingExpectedWalkingEntry {
			return fmt.Errorf("got unexpected walk entry: path=%s, name=%s, isDir=%v", path, d.Name(), d.IsDir())
		}

		return nil
	}))

	for _, expectedWalkedEntry := range expectedWalkedEntries {
		assert.Truef(t.T(), expectedWalkedEntry.found,
			"Missing walked entry: path=%s, name=%s, isDir=%v", expectedWalkedEntry.path, expectedWalkedEntry.name, expectedWalkedEntry.isDir)
	}

	assert.NoError(t.T(), os.RemoveAll(mntDir+"/*"))
}

// //////////////////////////////////////////////////////////////////////
// Tests
// /////////////////////////////////////////////////////////////////////
func (t *UnsupportedObjectsTest) Test_UnsupportedGcsObjectNames() {
	// Set up contents.
	objectNames :=
		map[string]string{
			"foo//0":   "", // unsupported
			"foo/1":    "", // supported, implicit path
			"foo/2//4": "", // unsupported
			"foo//2/5": "", // unsupported
			"foo/2/6":  "", // supported, implicit path
			"6":        "", // supported
			"/7":       "", // unsupported
			"/8/10":    "", // unsupported
			"/":        "", // unsupported
		}

	// Verify that the unsupported objects fail os.Stat.
	invalidPaths := []string{
		"foo//0",
		"foo/2//4",
		"foo//2/5",
		"/7",
		"/8/10"}

	// Verify that the supported objects appear in WalkDir.
	expectedWalkedEntries := []expectedWalkedEntry{{
		path:  mntDir,
		name:  mntDir[strings.LastIndex(mntDir, "/")+1:],
		isDir: true,
	}, {
		path:  path.Join(mntDir, "foo"),
		name:  "foo",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/1"),
		name: "1",
	}, {
		path:  path.Join(mntDir, "foo/2"),
		name:  "2",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/2/6"),
		name: "6",
	}, {
		path: path.Join(mntDir, "6"),
		name: "6",
	},
	}

	t.testGcsObjects(objectNames, invalidPaths, expectedWalkedEntries)
}

func (t *UnsupportedObjectsTestWithoutImplicitDirs) Test_UnsupportedGcsObjectNames() {
	// Set up contents.
	objectNames :=
		map[string]string{
			"foo/":     "", //supported
			"foo//0":   "", // unsupported
			"foo/1":    "", // supported, explicit path
			"foo/2//4": "", // unsupported
			"foo//2/5": "", // unsupported
			"foo/2/6":  "", // supported, implicit path
			"6":        "", // supported
			"/7":       "", // unsupported
			"/8/10":    "", // unsupported
			"/":        "", // unsupported
		}

	// Verify that the unsupported objects fail os.Stat.
	invalidPaths := []string{
		"foo//0",
		"foo/2//4",
		"foo//2/5",
		"/7",
		"/8/10"}

	// Verify that the supported objects appear in WalkDir.
	expectedWalkedEntries := []expectedWalkedEntry{{
		path:  mntDir,
		name:  mntDir[strings.LastIndex(mntDir, "/")+1:],
		isDir: true,
	}, {
		path:  path.Join(mntDir, "foo"),
		name:  "foo",
		isDir: true,
	}, {
		path: path.Join(mntDir, "foo/1"),
		name: "1",
	}, {
		path: path.Join(mntDir, "6"),
		name: "6",
	},
	}

	t.testGcsObjects(objectNames, invalidPaths, expectedWalkedEntries)
}

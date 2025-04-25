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

package operations_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/all_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OperationSuite struct {
	mountConfiguration *TestMountConfiguration
	suite.Suite
}

func (s *OperationSuite) SetupSuite() {
	err := s.mountConfiguration.Mount(s.T(), storageClient)
	require.NoError(s.T(), err)
}

func removeObjectsDirectories(rootDir string) error {
	return filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Propagate the error upwards, but continue walking if possible
			// (e.g., permission error on a specific file/dir shouldn't stop everything)
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return nil // or return err to stop walking
		}

		// Check if it's a directory and its name is "objects"
		// And ensure it's not the rootDir itself if rootDir is named "objects"
		if d.IsDir() && d.Name() == "objects" && path != rootDir {
			fmt.Printf("Removing directory: %s\n", path)
			err := os.RemoveAll(path)
			if err != nil {
				fmt.Printf("Failed to remove directory %q: %v\n", path, err)
				// Decide if you want to stop or continue on error
				return err // Stop on error
				// return nil // Continue on error
			}
			// If a directory is removed, skip walking its contents
			return filepath.SkipDir
		}
		return nil
	})
}

func (s *OperationSuite) TearDownTest() {
	err := removeObjectsDirectories(s.mountConfiguration.MntDir())
	require.NoError(s.T(), err)
}

func (s *OperationSuite) TestStatWithTrailingNewline() {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))

	_, err := os.Stat(testDir + "/\n")

	require.Error(s.T(), err)
	assert.Equal(s.T(), err.(*os.PathError).Err, syscall.ENOENT)
}

func TestOperationsSuite(t *testing.T) {
	t.Parallel()
	for i := range configurations {
		t.Run(configurations[i].MountType().String()+"_"+setup.GenerateRandomString(5), func(t *testing.T) {
			t.Parallel()
			suite.Run(t, &OperationSuite{mountConfiguration: &configurations[i]})
		})
	}
}

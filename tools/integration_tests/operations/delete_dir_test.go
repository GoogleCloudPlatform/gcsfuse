// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provides integration tests for delete directory.
package operations_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestDeleteEmptyExplicitDir(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	dirPath := path.Join(setup.MntDir(), EmptyExplicitDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(0, dirPath, "", t)

	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in deleting empty explicit directory.")
	}

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == EmptyExplicitDirectoryForDeleteTest && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}

func TestDeleteNonEmptyExplicitDir(t *testing.T) {
	// Clean the mountedDirectory before running tests.
	setup.CleanMntDir()

	dirPath := path.Join(setup.MntDir(), NonEmptyExplicitDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInNonEmptyExplicitDirectoryForDeleteTest, dirPath, PrefixFilesInNonEmptyExplicitDirectoryForDeleteTest, t)

	subDirPath := path.Join(dirPath, NonEmptyExplicitSubDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInNonEmptyExplicitSubDirectoryForDeleteTest, subDirPath, PrefixFilesInNonEmptyExplicitSubDirectoryForDeleteTest, t)

	err := os.RemoveAll(dirPath)
	if err != nil {
		t.Errorf("Error in deleting empty explicit directory.")
	}

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == NonEmptyExplicitDirectoryForDeleteTest && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}

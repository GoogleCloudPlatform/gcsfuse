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

// Provides integration tests for file operations with --o=ro flag set.
package readonly_test

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func checkIfFileCreationFailed(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_CREATE, setup.FilePermission_0600)

	if err == nil {
		t.Errorf("File is created in read-only file system.")
	}

	checkErrorForReadOnlyFileSystem(err, t)

	defer file.Close()
}

func TestCreateFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "testFile.txt")

	checkIfFileCreationFailed(filePath, t)
}

func TestCreateFileInDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "testFile.txt")

	checkIfFileCreationFailed(filePath, t)
}

func checkIfDirCreationFailed(dirPath string, t *testing.T) {
	err := os.Mkdir(dirPath, fs.ModeDir)

	if err == nil {
		t.Errorf("Directory is created in read-only file system.")
	}

	checkErrorForReadOnlyFileSystem(err, t)
}

func TestCreateDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), "test")

	checkIfDirCreationFailed(dirPath, t)
}

func TestCreateSubDirectoryInDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "test")

	checkIfDirCreationFailed(dirPath, t)
}

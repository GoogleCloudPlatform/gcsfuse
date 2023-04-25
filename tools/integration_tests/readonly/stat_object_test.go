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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func statExistingObj(objPath string, t *testing.T) (file os.FileInfo) {
	file, err := os.Stat(objPath)
	if err != nil {
		t.Errorf("Fail to stat the object.")
	}
	return file
}

func TestStatFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)
	file := statExistingObj(filePath, t)

	if file.Name() != FileNameInTestBucket || file.IsDir() != false {
		t.Errorf("Stat incorrrect file.")
	}
}

func TestStatFileInSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)
	file := statExistingObj(filePath, t)

	if file.Name() != FileNameInDirectoryTestBucket || file.IsDir() != false {
		t.Errorf("Stat incorrrect file.")
	}
}

func TestStatDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	dir := statExistingObj(DirPath, t)

	if dir.Name() != DirectoryNameInTestBucket || dir.IsDir() != true {
		t.Errorf("Stat incorrrect Directory.")
	}
}

func TestStatSubDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	dir := statExistingObj(DirPath, t)

	if dir.Name() != SubDirectoryNameInTestBucket || dir.IsDir() != true {
		t.Errorf("Stat incorrrect Directory.")
	}
}

func checkIfNonExistentFileOpenToStat(objPath string, t *testing.T) {
	_, err := os.Stat(objPath)
	if err == nil {
		t.Errorf("Object exist!!")
	}

	// It will throw an error no such file or directory.
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Throwing incorrect error.")
	}
}

func TestStatNotExistingFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	checkIfNonExistentFileOpenToStat(filePath, t)
}

func TestStatNotExistingFileFromSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileOpenToStat(filePath, t)
}

func TestStatNotExistingDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), DirNotExist)

	checkIfNonExistentFileOpenToStat(DirPath, t)
}

func TestStatNotExistingSubDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, DirNotExist)

	checkIfNonExistentFileOpenToStat(DirPath, t)
}

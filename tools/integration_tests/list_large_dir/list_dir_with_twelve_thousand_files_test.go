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

package list_large_dir_test

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
	// Clean the bucket for list testing.
	os.RemoveAll(setup.MntDir())

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)

	files, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	if len(files) != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(files))
	}
}

func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDir)
	//operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir, t)

	subDirPath := path.Join(dirPath, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir)
	for i := 0; i < 100; i++ {
		// Create 100 Explicit directory.
		operations.CreateDirectoryWithNFiles(1, subDirPath, PrefixFileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDir, t)
	}

	files, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfDirs int = 0
	var numberOfFiles int = 0
	for _, obj := range files {
		if obj.IsDir() {
			numberOfDirs++
		} else {
			numberOfFiles++
		}
	}
	if numberOfDirs != NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", numberOfDirs)
	}
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}
}

func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir)
	operations.CreateDirectoryWithNFiles(NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, dirPath, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, t)

	for i := 0; i < 100; i++ {
		subDirPath := path.Join(dirPath, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir+strconv.Itoa(i))
		// Create 100 Explicit directory.
		operations.CreateDirectoryWithNFiles(1, subDirPath, PrefixFileInSubDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, t)
	}

	//setup.RunScriptForTestData("testdata/create_implicit_dir.sh", subDirPath)

	files, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfDirs int = 0
	var numberOfFiles int = 0
	for _, obj := range files {
		if obj.IsDir() {
			numberOfDirs++
		} else {
			numberOfFiles++
		}
	}
	if numberOfDirs != NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", numberOfDirs)
	}
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}
}

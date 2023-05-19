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

package list_large_dir

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
	// Clean the bucket for list testing.
	os.RemoveAll(setup.MntDir())

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	setup.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, t)

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
	setup.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, t)

	// Create 100 Explicit directory.
	for i := 0; i < NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir; i++ {
		_, err := os.MkdirTemp(dirPath, "tmpDir")
		if err != nil {
			t.Errorf("Error in creating directory: %v", err)
		}
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
		t.Errorf("Incorrect number of directories.")
	}
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Incorrect number of Files.")
	}
}

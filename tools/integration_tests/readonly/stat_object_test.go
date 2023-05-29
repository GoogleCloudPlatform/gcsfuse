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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func statObject(objPath string, t *testing.T) (file os.FileInfo) {
	file, err := os.Stat(objPath)
	if err != nil {
		t.Errorf("Fail to stat the object.")
	}
	return file
}

func TestStatFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	file := statObject(filePath, t)

	// Name - testBucket/Test1.txt, Type - File
	if file.Name() != FileNameInTestBucket || file.IsDir() != false {
		t.Errorf("Object stated for file in bucket is incorrect.")
	}
}

func TestStatFileFromBucketDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	file := statObject(filePath, t)

	// Name - testBucket/Test/a.txt, Type - File
	if file.Name() != FileNameInDirectoryTestBucket || file.IsDir() != false {
		t.Errorf("Object stated for file in bucket directory is incorrect.")
	}
}

func TestStatFileFromBucketSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)

	file := statObject(filePath, t)

	// Name - testBucket/Test/b/b.txt, Type - File
	if file.Name() != FileNameInSubDirectoryTestBucket || file.IsDir() != false {
		t.Errorf("Object stated for file in bucket subdirectory is incorrect.")
	}
}

func TestStatDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)

	dir := statObject(dirPath, t)

	// Name - testBucket/Test/, Type - Dir
	if dir.Name() != DirectoryNameInTestBucket || dir.IsDir() != true {
		t.Errorf("Object stated for bucket directory is incorrect.")
	}
}

func TestStatSubDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)

	dir := statObject(dirPath, t)

	// Name - testBucket/Test/b, Type - Dir
	if dir.Name() != SubDirectoryNameInTestBucket || dir.IsDir() != true {
		t.Errorf("Object stated for bucket sub directory is incorrect.")
	}
}

func checkIfNonExistentObjectFailedToStat(objPath string, t *testing.T) {
	_, err := os.Stat(objPath)
	if err == nil {
		t.Errorf("Incorrect object exist!!")
	}

	checkErrorForObjectNotExist(err, t)
}

func TestStatNotExistingFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	checkIfNonExistentObjectFailedToStat(filePath, t)
}

func TestStatNotExistingFileFromBucketDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentObjectFailedToStat(filePath, t)
}

func TestStatNotExistingDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirNotExist)

	checkIfNonExistentObjectFailedToStat(dirPath, t)
}

func TestStatNotExistingSubDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, DirNotExist)

	checkIfNonExistentObjectFailedToStat(dirPath, t)
}

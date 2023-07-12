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

// Provides integration tests for file and directory attributes.
package operations_test

import (
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirAttrTest = "dirAttrTest"
const SubDirAttrTest = "subDirAttrTest"
const PrefixFileInDirAttrTest = "fileInDirAttrTest"
const NumberOfFilesInDirAttrTest = 2

func checkIfObjectAttrIsCorrect(objName string, preCreateTime time.Time, postCreateTime time.Time, t *testing.T) (oStat fs.FileInfo) {
	oStat, err := os.Stat(objName)

	if err != nil {
		t.Errorf("os.Stat error: %s, %v", objName, err)
	}
	statObjName := path.Join(setup.MntDir(), oStat.Name())
	if objName != statObjName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statObjName, objName)
	}
	if (preCreateTime.After(oStat.ModTime())) || (postCreateTime.Before(oStat.ModTime())) {
		t.Errorf("File modification time not in the expected time-range")
	}
	return
}

func TestFileAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	preCreateTime := time.Now()
	fileName := setup.CreateTempFile()
	postCreateTime := time.Now()

	// In the setup function we are writing 14 bytes.
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/integration_tests/util/setup/setup.go#L124
	fStat := checkIfObjectAttrIsCorrect(fileName, preCreateTime, postCreateTime, t)

	// The file size in createTempFile() is 14 bytes
	if fStat.Size() != 14 {
		t.Errorf("File size is not 14 bytes, found size: %d bytes", fStat.Size())
	}
}

func TestEmptyDirAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	preCreateTime := time.Now()
	dirName := path.Join(setup.MntDir(), DirAttrTest)
	operations.CreateDirectoryWithNFiles(0, dirName, "", t)
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, t)
}

func TestNonEmptyDirAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	preCreateTime := time.Now()
	dirName := path.Join(setup.MntDir(), DirAttrTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirAttrTest, dirName, PrefixFileInDirAttrTest, t)
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, t)
}

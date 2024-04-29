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
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/ogletest"
)

const DirAttrTest = "dirAttrTest"
const PrefixFileInDirAttrTest = "fileInDirAttrTest"
const NumberOfFilesInDirAttrTest = 2
const BytesWrittenInFile = 14

func checkIfObjectAttrIsCorrect(objName string, preCreateTime time.Time, postCreateTime time.Time, byteSize int64, t *testing.T) {
	oStat, err := os.Stat(objName)

	if err != nil {
		t.Errorf("os.Stat error: %s, %v", objName, err)
	}
	statObjName := path.Join(setup.MntDir(), DirForOperationTests, oStat.Name())
	if objName != statObjName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statObjName, objName)
	}
	ogletest.ExpectThat(oStat, fusetesting.MtimeIsWithin(preCreateTime, operations.TimeSlop))
	ogletest.ExpectThat(oStat, fusetesting.MtimeIsWithin(postCreateTime, operations.TimeSlop))

	if oStat.Size() != byteSize {
		t.Errorf("File size is not %v bytes, found size: %d bytes", BytesWrittenInFile, oStat.Size())
	}
}

func TestFileAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	preCreateTime := time.Now()
	fileName := path.Join(testDir, tempFileName)
	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)
	postCreateTime := time.Now()

	// The file size in createTempFile() is BytesWrittenInFile bytes
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/integration_tests/util/setup/setup.go#L124
	checkIfObjectAttrIsCorrect(fileName, preCreateTime, postCreateTime, BytesWrittenInFile, t)
}

func TestEmptyDirAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	preCreateTime := time.Now()
	dirName := path.Join(testDir, DirAttrTest)
	operations.CreateDirectoryWithNFiles(0, dirName, "", t)
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(path.Join(testDir, DirAttrTest), preCreateTime, postCreateTime, 0, t)
}

func TestNonEmptyDirAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	preCreateTime := time.Now()
	dirName := path.Join(testDir, DirAttrTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirAttrTest, dirName, PrefixFileInDirAttrTest, t)
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, 0, t)
}

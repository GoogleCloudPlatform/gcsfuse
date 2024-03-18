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
	statObjName := path.Join(setup.MntDir(), oStat.Name())
	if objName != statObjName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statObjName, objName)
	}
	statModTime := oStat.ModTime().Round(time.Second)
	if (preCreateTime.After(statModTime)) || (postCreateTime.Before(statModTime)) {
		t.Errorf("File modification time not in the expected time-range")
	}

	if oStat.Size() != byteSize {
		t.Errorf("File size is not %v bytes, found size: %d bytes", BytesWrittenInFile, oStat.Size())
	}
}

func TestFileAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// kernel time can be slightly out of sync of time.Now(), so rounding off
	// times to seconds. Ref: https://github.com/golang/go/issues/33510
	preCreateTime := time.Now().Round(time.Second)
	fileName := setup.CreateTempFile()
	postCreateTime := time.Now().Round(time.Second)

	// The file size in createTempFile() is BytesWrittenInFile bytes
	// https://github.com/googlecloudplatform/gcsfuse/v2/blob/master/tools/integration_tests/util/setup/setup.go#L124
	checkIfObjectAttrIsCorrect(fileName, preCreateTime, postCreateTime, BytesWrittenInFile, t)
}

func TestEmptyDirAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// kernel time can be slightly out of sync of time.Now(), so rounding off
	// times to seconds. Ref: https://github.com/golang/go/issues/33510
	preCreateTime := time.Now().Round(time.Second)
	dirName := path.Join(setup.MntDir(), DirAttrTest)
	operations.CreateDirectoryWithNFiles(0, dirName, "", t)
	postCreateTime := time.Now().Round(time.Second)

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, 0, t)
}

func TestNonEmptyDirAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// kernel time can be slightly out of sync of time.Now(), so rounding off
	// times to seconds. Ref: https://github.com/golang/go/issues/33510
	preCreateTime := time.Now().Round(time.Second)
	dirName := path.Join(setup.MntDir(), DirAttrTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirAttrTest, dirName, PrefixFileInDirAttrTest, t)
	postCreateTime := time.Now().Round(time.Second)

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, 0, t)
}

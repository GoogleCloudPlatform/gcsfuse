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

// Provides integration tests for file attributes.
package operations_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirAttrTest = "dirAttrTest"

func checkIfObjectAttrIsCorrect(objName string, preCreateTime time.Time, postCreateTime time.Time, t *testing.T) {
	oStat, err := os.Stat(objName)

	if err != nil {
		t.Errorf("os.Stat error: %s, %v", objName, err)
	}
	statFileName := path.Join(setup.MntDir(), oStat.Name())
	if objName != statFileName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statFileName, objName)
	}
	if (preCreateTime.After(oStat.ModTime())) || (postCreateTime.Before(oStat.ModTime())) {
		t.Errorf("File modification time not in the expected time-range")
	}
	// The file size in createTempFile() is 14 bytes
	if oStat.Size() != 14 {
		t.Errorf("File size is not 14 bytes, found size: %d bytes", oStat.Size())
	}
}

func TestFileAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	preCreateTime := time.Now()
	fileName := setup.CreateTempFile()
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(fileName, preCreateTime, postCreateTime, t)
}

func TestDirAttributes(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	preCreateTime := time.Now()
	dirName := path.Join(setup.MntDir(), DirAttrTest)
	err := os.Mkdir(dirName, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory:%v", err)
	}
	postCreateTime := time.Now()

	checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, t)
}

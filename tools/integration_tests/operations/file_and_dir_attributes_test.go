// Copyright 2023 Google LLC
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
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	DirAttrTest                = "dirAttrTest"
	PrefixFileInDirAttrTest    = "fileInDirAttrTest"
	NumberOfFilesInDirAttrTest = 2
	BytesWrittenInFile         = 14
	retryFrequency             = 10 * time.Second
	retryDuration              = 3 * time.Minute
)

func checkIfObjectAttrIsCorrect(objName string, preCreateTime time.Time, postCreateTime time.Time, byteSize int64, t *testing.T) error {
	oStat, err := os.Stat(objName)

	if err != nil {
		return fmt.Errorf("os.Stat error: %s, %v", objName, err)
	}
	statObjName := path.Join(setup.MntDir(), DirForOperationTests, oStat.Name())
	if objName != statObjName {
		return fmt.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statObjName, objName)
	}

	statModTime := oStat.ModTime()
	if (preCreateTime.After(statModTime)) || (postCreateTime.Before(statModTime)) {
		return fmt.Errorf("File modification time not in the expected time-range")
	}

	if oStat.Size() != byteSize {
		return fmt.Errorf("File size is not %v bytes, found size: %d bytes", BytesWrittenInFile, oStat.Size())
	}
	return nil
}

func TestFileAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	operations.RetryUntil(context.Background(), t, retryFrequency, retryDuration, func() (bool, error) {
		fileName := path.Join(testDir, operations.GetRandomName(t))
		// kernel time can be slightly out of sync of time.Now(), so using
		// operations.TimeSlop to adjust pre and post create time.
		// Ref: https://github.com/golang/go/issues/33510
		preCreateTime := time.Now().Add(-operations.TimeSlop)
		operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)
		postCreateTime := time.Now().Add(+operations.TimeSlop)

		// The file size in createTempFile() is BytesWrittenInFile bytes
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/integration_tests/util/setup/setup.go#L124
		err := checkIfObjectAttrIsCorrect(fileName, preCreateTime, postCreateTime, BytesWrittenInFile, t)
		return err == nil, err
	})
}

func TestEmptyDirAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	operations.RetryUntil(context.Background(), t, retryFrequency, retryDuration, func() (bool, error) {
		dirName := path.Join(testDir, operations.GetRandomName(t))
		// kernel time can be slightly out of sync of time.Now(), so using
		// operations.TimeSlop to adjust pre and post create time.
		// Ref: https://github.com/golang/go/issues/33510
		preCreateTime := time.Now().Add(-operations.TimeSlop)
		operations.CreateDirectoryWithNFiles(0, dirName, "", t)
		postCreateTime := time.Now().Add(operations.TimeSlop)

		err := checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, 0, t)
		return err == nil, err
	})
}

func TestNonEmptyDirAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	operations.RetryUntil(context.Background(), t, retryFrequency, retryDuration, func() (bool, error) {
		dirName := path.Join(testDir, operations.GetRandomName(t))
		// kernel time can be slightly out of sync of time.Now(), so using
		// operations.TimeSlop to adjust pre and post create time.
		// Ref: https://github.com/golang/go/issues/33510
		preCreateTime := time.Now().Add(-operations.TimeSlop)
		operations.CreateDirectoryWithNFiles(NumberOfFilesInDirAttrTest, dirName, PrefixFileInDirAttrTest, t)
		postCreateTime := time.Now().Add(operations.TimeSlop)

		err := checkIfObjectAttrIsCorrect(dirName, preCreateTime, postCreateTime, 0, t)
		return err == nil, err
	})
}

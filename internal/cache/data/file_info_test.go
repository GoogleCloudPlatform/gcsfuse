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

package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const TestDataFileSize uint64 = 23
const TestTimeInEpoch int64 = 1654041600
const TestBucketName = "test_bucket"
const TestObjectName = "test/a.txt"
const TestGeneration = 12345678
const ExpectedFileInfoKey = "test_bucket1654041600test/a.txt"

func getTestFileInfoKey() FileInfoKey {
	return FileInfoKey{
		BucketName:         TestBucketName,
		ObjectName:         TestObjectName,
		BucketCreationTime: time.Unix(TestTimeInEpoch, 0),
	}
}

func TestKeyMethod(t *testing.T) {
	fik := getTestFileInfoKey()

	key, err := fik.Key()

	assert.NoError(t, err)
	unixCreationTimeString := fmt.Sprintf("%d", fik.BucketCreationTime.Unix())
	assert.Equal(t, fik.BucketName+unixCreationTimeString+fik.ObjectName, key)
}

func TestKeyMethodWithEmptyBucketName(t *testing.T) {
	fik := getTestFileInfoKey()
	fik.BucketName = ""

	key, err := fik.Key()

	assert.Equal(t, InvalidKeyAttributes, err.Error())
	assert.Equal(t, "", key)
}

func TestKeyMethodWithEmptyObjectName(t *testing.T) {
	fik := getTestFileInfoKey()
	fik.ObjectName = ""

	key, err := fik.Key()

	assert.Equal(t, InvalidKeyAttributes, err.Error())
	assert.Equal(t, "", key)
}

func TestSizeMethod(t *testing.T) {
	fi := FileInfo{
		Key:              getTestFileInfoKey(),
		ObjectGeneration: TestGeneration,
		FileSize:         TestDataFileSize,
	}

	assert.Equal(t, TestDataFileSize, fi.Size())
}

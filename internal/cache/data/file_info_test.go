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

package data

import (
	"fmt"
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
)

func TestFileInfo(t *testing.T) { RunTests(t) }

type fileInfoTestKey struct {
}

type fileInfoTest struct {
}

const TestDataFileSize uint64 = 23
const TestTimeInEpoch int64 = 1654041600
const TestBucketName = "test_bucket"
const TestObjectName = "test/a.txt"
const TestGeneration = 12345678
const ExpectedFileInfoKey = "test_bucket1654041600test/a.txt"

func init() {
	RegisterTestSuite(&fileInfoTestKey{})
	RegisterTestSuite(&fileInfoTest{})
}

func getTestFileInfoKey() FileInfoKey {
	return FileInfoKey{
		BucketName:         TestBucketName,
		ObjectName:         TestObjectName,
		BucketCreationTime: time.Unix(TestTimeInEpoch, 0),
	}
}

func (t *fileInfoTestKey) TestKeyMethod() {
	fik := getTestFileInfoKey()

	key, err := fik.Key()
	AssertEq(nil, err)

	ExpectEq(ExpectedFileInfoKey, key)
}

func (t *fileInfoTestKey) TestKeyMethodWithEmptyBucketName() {
	fik := getTestFileInfoKey()
	fik.BucketName = ""

	key, err := fik.Key()
	AssertEq(InvalidKeyAttributes, err.Error())

	ExpectEq("", key)
}

func (t *fileInfoTestKey) TestKeyMethodWithZeroBucketCreationTime() {
	fik := getTestFileInfoKey()

	key, err := fik.Key()

	ExpectEq(nil, err)
	unixCreationTimeString := fmt.Sprintf("%d", fik.BucketCreationTime.Unix())
	ExpectEq(fik.BucketName+unixCreationTimeString+fik.ObjectName, key)
}

func (t *fileInfoTestKey) TestKeyMethodWithEmptyObjectName() {
	fik := getTestFileInfoKey()
	fik.ObjectName = ""

	key, err := fik.Key()
	AssertEq(InvalidKeyAttributes, err.Error())

	ExpectEq("", key)
}

func (t *fileInfoTest) TestSizeMethod() {
	fi := FileInfo{
		Key:              getTestFileInfoKey(),
		ObjectGeneration: TestGeneration,
		FileSize:         TestDataFileSize,
	}

	ExpectEq(TestDataFileSize, fi.Size())
}

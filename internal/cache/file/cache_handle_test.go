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

package file

import (
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
)

func TestCacheHandle(t *testing.T) { RunTests(t) }

type cacheHandleTest struct {
}

func init() {
	RegisterTestSuite(&cacheHandleTest{})
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

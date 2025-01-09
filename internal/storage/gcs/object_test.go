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

package gcs

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestObject(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ObjectTest struct {
}

func init() { RegisterTestSuite(&ObjectTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ObjectTest) HasContentEncodingGzipPositive() {
	mo := MinObject{}
	mo.ContentEncoding = "gzip"

	AssertTrue(mo.HasContentEncodingGzip())
}

func (t *ObjectTest) HasContentEncodingGzipNegative() {
	encodings := []string{"", "GZIP", "xzip", "zip"}

	for _, encoding := range encodings {
		mo := MinObject{}
		mo.ContentEncoding = encoding

		AssertFalse(mo.HasContentEncodingGzip())
	}
}

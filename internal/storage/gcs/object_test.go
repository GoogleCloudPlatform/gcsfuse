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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHasContentEncodingGzipPositive(t *testing.T) {
	mo := MinObject{}
	mo.ContentEncoding = "gzip"

	assert.True(t, mo.HasContentEncodingGzip())
}

func TestHasContentEncodingGzipNegative(t *testing.T) {
	encodings := []string{"", "GZIP", "xzip", "zip"}

	for _, encoding := range encodings {
		mo := MinObject{}
		mo.ContentEncoding = encoding

		assert.False(t, mo.HasContentEncodingGzip())
	}
}

func TestIsFinalized(t *testing.T) {
	mo := MinObject{Finalized: TimeToNS(time.Date(2025, time.June, 19, 18, 23, 30, 0, time.UTC))}

	assert.False(t, mo.IsUnfinalized())
}

func TestIsNotFinalized(t *testing.T) {
	mo := MinObject{}

	assert.True(t, mo.IsUnfinalized())
}

func TestObjectIsFinalized(t *testing.T) {
	o := Object{Finalized: time.Date(2025, time.June, 19, 18, 23, 30, 0, time.UTC)}

	assert.False(t, o.IsUnfinalized())
}

func TestObjectIsNotFinalized(t *testing.T) {
	o := Object{}

	assert.True(t, o.IsUnfinalized())
}

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
	"math"
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
func TestTimeToNS_Zero(t *testing.T) {
	assert.Equal(t, int64(0), TimeToNS(time.Time{}))
}

func TestTimeToNS_NonZero(t *testing.T) {
	ti := time.Date(2026, time.July, 12, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, ti.UnixNano(), TimeToNS(ti))
}

func TestNSToTime_Zero(t *testing.T) {
	assert.True(t, NSToTime(0).IsZero())
}

func TestNSToTime_NonZero(t *testing.T) {
	ti := time.Date(2026, time.July, 12, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, ti, NSToTime(ti.UnixNano()))
}

func TestMinObject_UpdatedTime(t *testing.T) {
	ti := time.Date(2026, time.July, 12, 10, 0, 0, 0, time.UTC)
	mo := MinObject{Updated: ti.UnixNano()}

	assert.Equal(t, ti, mo.UpdatedTime())
}

func TestMinObject_FinalizedTime(t *testing.T) {
	ti := time.Date(2026, time.July, 12, 10, 0, 0, 0, time.UTC)
	mo := MinObject{Finalized: ti.UnixNano()}

	assert.Equal(t, ti, mo.FinalizedTime())
}

func TestTimeToNS_Epoch(t *testing.T) {
	// time.Unix(0,0) should return math.MinInt64 to avoid colliding with time.Time{} mapping to 0
	ti := time.Unix(0, 0).UTC()
	assert.Equal(t, int64(math.MinInt64), TimeToNS(ti))
}

func TestNSToTime_Epoch(t *testing.T) {
	// math.MinInt64 should be correctly decoded as 1970 epoch
	ti := time.Unix(0, 0).UTC()
	assert.Equal(t, ti, NSToTime(math.MinInt64))
}

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
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
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

func TestUnsafeSizeOf_MinObject(t *testing.T) {
	var emptyMinObj MinObject
	assert.Equal(t, int(unsafe.Sizeof(emptyMinObj)), util.UnsafeSizeOf(&emptyMinObj))

	ptrToM := &emptyMinObj
	assert.Equal(t, int(unsafe.Sizeof(ptrToM)), util.UnsafeSizeOf(&ptrToM))
}

func TestNestedSizeOfGcsMinObject(t *testing.T) {
	const name string = "my-object"
	const contentEncoding string = "gzip/none"
	var generation int64 = 858734898
	var metaGeneration int64 = 858734899
	var crc32 uint32 = 1234
	updated := time.Now()
	customMetadaField1 := "google-symlink"
	customMetadaValue1 := "true"
	customMetadaField2 := "google-xyz-field"
	customMetadaValue2 := "google-symlink"
	customMetadataFields := map[string]string{customMetadaField1: customMetadaValue1, customMetadaField2: customMetadaValue2}
	emptyStringSize := int(reflect.TypeOf("").Size())
	customMetadataFieldsContentSize := emptyStringSize + util.ContentSizeOfString(&customMetadaField1) + emptyStringSize + util.ContentSizeOfString(&customMetadaValue1) + emptyStringSize + util.ContentSizeOfString(&customMetadaField2) + emptyStringSize + util.ContentSizeOfString(&customMetadaValue2)

	m := MinObject{
		Name:            name,
		Size:            100,
		ContentEncoding: contentEncoding,
		Metadata:        customMetadataFields,
		Generation:      generation,
		MetaGeneration:  metaGeneration,
		Updated:         TimeToNS(updated),
		Finalized:       0,
		CRC32C:          &crc32,
	}

	expectedSize := int(unsafe.Sizeof(MinObject{}))
	expectedSize += len(name) + len(contentEncoding) + int(unsafe.Sizeof(crc32))
	expectedSize += customMetadataFieldsContentSize

	assert.Equal(t, expectedSize, NestedSizeOfGcsMinObject(&m))
}

func BenchmarkUnsafeSizeOf_MinObject(b *testing.B) {
	val := MinObject{
		Name: "benchmark_test_min_object",
	}
	b.ResetTimer()
	for range b.N {
		_ = util.UnsafeSizeOf(&val)
	}
}

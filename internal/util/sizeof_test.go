// Copyright 2024 Google LLC
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

package util

import (
	"net/http"
	"testing"
	"time"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

// //////////////////////////////////////////////////////////////////////

var (
	i            int
	ui32         uint32
	intArray     []int
	b            byte
	stringIntMap map[string]int

	sizeOfInt               = int(unsafe.Sizeof(i))
	sizeOfIntPtr            = int(unsafe.Sizeof(&i))
	sizeOfUInt32            = int(unsafe.Sizeof(ui32))
	sizeOfUInt32Ptr         = int(unsafe.Sizeof(&ui32))
	sizeOfByte              = int(unsafe.Sizeof(b))
	sizeOfEmptyIntArray     = int(unsafe.Sizeof(intArray))
	sizeOfEmptyStringIntMap = int(unsafe.Sizeof(stringIntMap))

	sizeOfEmptyMinObject = int(unsafe.Sizeof(gcs.MinObject{}))
	sizeOfEmptyStruct = int(unsafe.Sizeof(struct{}{}))
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestUnsafeSizeOf(t *testing.T) {
	assert.Equal(t, sizeOfInt, UnsafeSizeOf(&i))

	ptrToI := &i
	assert.Equal(t, sizeOfIntPtr, UnsafeSizeOf(&ptrToI))

	assert.Equal(t, sizeOfUInt32, UnsafeSizeOf(&ui32))

	ptrToUi32 := &ui32
	assert.Equal(t, sizeOfUInt32Ptr, UnsafeSizeOf(&ptrToUi32))

	var emptyStructVal struct{}
	assert.Equal(t, sizeOfEmptyStruct, UnsafeSizeOf(&emptyStructVal))

	var structVal1 struct {
		x int
	}
	assert.Equal(t, sizeOfEmptyStruct+sizeOfInt, UnsafeSizeOf(&structVal1))

	var structVal2 struct {
		a          int
		b1, b2, b3 byte
		c          string
	}
	assert.Equal(t, sizeOfEmptyStruct+sizeOfInt+3*sizeOfByte+5 /*for-padding-for-alignment*/ +emptyStringSize, UnsafeSizeOf(&structVal2))

	emptyStr := ""
	assert.Equal(t, emptyStringSize, UnsafeSizeOf(&emptyStr))

	helloStr := "hello"
	assert.Equal(t, emptyStringSize, UnsafeSizeOf(&helloStr))

	intArrayVal := []int{1, 2, 3}
	assert.Equal(t, sizeOfEmptyIntArray, UnsafeSizeOf(&intArrayVal))

	stringArrayVal := []string{"few ", "fewfgwe", "", "fewawef"}
	assert.Equal(t, emptyStringArraySize, UnsafeSizeOf(&stringArrayVal))

	stringIntMapVal := map[string]int{"few ": 432, "fewfgwe": -21, "": 1, "fewawef": 0}
	assert.Equal(t, sizeOfEmptyStringIntMap, UnsafeSizeOf(&stringIntMapVal))

	var emptyMinObj gcs.MinObject
	assert.Equal(t, int(unsafe.Sizeof(emptyMinObj)), UnsafeSizeOf(&emptyMinObj))

	ptrToM := &emptyMinObj
	assert.Equal(t, int(unsafe.Sizeof(ptrToM)), UnsafeSizeOf(&ptrToM))

	var emptyFolder gcs.Folder
	assert.Equal(t, int(unsafe.Sizeof(emptyFolder)), UnsafeSizeOf(&emptyFolder))

	ptrToF := &emptyFolder
	assert.Equal(t, int(unsafe.Sizeof(ptrToF)), UnsafeSizeOf(&ptrToF))

	var nilInt *int = nil
	assert.Equal(t, 0, UnsafeSizeOf(nilInt))
}

func TestContentSizeOfString(t *testing.T) {
	for _, tc := range []struct {
		str                   string
		expected_content_size int
		expected_nested_size  int
	}{
		{
			str:                   "",
			expected_content_size: 0,
		},
		{
			str:                   "hello",
			expected_content_size: 5,
		},
		{
			str:                   "hello-world",
			expected_content_size: 11,
		},
	} {
		assert.Equal(t, tc.expected_content_size, contentSizeOfString(&tc.str))
	}
}

func TestContentSizeOfArrayOfStrings(t *testing.T) {
	for _, tc := range []struct {
		strs                  []string
		expected_content_size int
	}{
		{
			strs:                  []string{},
			expected_content_size: 0,
		},
		{
			strs:                  []string{""},
			expected_content_size: emptyStringSize,
		},
		{
			strs:                  []string{"", ""},
			expected_content_size: 2 * emptyStringSize,
		},
		{
			strs:                  []string{"hello", ""},
			expected_content_size: 2*emptyStringSize + 5,
		},
		{
			strs:                  []string{"hello", "hello-world"},
			expected_content_size: 2*emptyStringSize + 5 + 11,
		},
	} {
		assert.Equal(t, tc.expected_content_size, contentSizeOfArrayOfStrings(&tc.strs))
	}
}

func TestContentSizeOfStringToStringMap(t *testing.T) {
	for _, tc := range []struct {
		m                     map[string]string
		expected_content_size int
	}{
		{
			m:                     map[string]string{},
			expected_content_size: 0,
		},
		{
			m:                     map[string]string{"hello": "to you"},
			expected_content_size: emptyStringSize + 5 + emptyStringSize + 6,
		},
		{
			m:                     map[string]string{"a": ""},
			expected_content_size: emptyStringSize + 1 + emptyStringSize,
		},
		{
			m:                     map[string]string{"": ":"},
			expected_content_size: emptyStringSize + emptyStringSize + 1,
		},
		{
			m:                     map[string]string{"a": "b1", "xyz": "alpha"},
			expected_content_size: emptyStringSize + 1 + emptyStringSize + 2 + emptyStringSize + 3 + emptyStringSize + 5,
		},
	} {
		assert.Equal(t, tc.expected_content_size, contentSizeOfStringToStringMap(&tc.m))
	}
}

func TestContentSizeOfStringToStringArrayMap(t *testing.T) {
	for _, tc := range []struct {
		m                     map[string][]string
		expected_content_size int
	}{
		{
			m:                     map[string][]string{},
			expected_content_size: 0,
		},
		{
			m:                     map[string][]string{"hello": {"to you"}},
			expected_content_size: emptyStringSize + 5 + emptyStringArraySize + emptyStringSize + 6,
		},
		{
			m:                     map[string][]string{"a": {""}},
			expected_content_size: emptyStringSize + 1 + emptyStringArraySize + emptyStringSize,
		},
		{
			m:                     map[string][]string{"": {":"}},
			expected_content_size: emptyStringSize + emptyStringArraySize + emptyStringSize + 1,
		},
		{
			m:                     map[string][]string{"a": {"b1", "b2"}, "xyz": {"alpha", "beta"}},
			expected_content_size: emptyStringSize + 1 + emptyStringArraySize + emptyStringSize + 2 + emptyStringSize + 2 + emptyStringSize + 3 + emptyStringArraySize + emptyStringSize + 5 + emptyStringSize + 4,
		},
	} {
		assert.Equal(t, tc.expected_content_size, contentSizeOfStringToStringArrayMap(&tc.m))
	}
}

func TestContentSizeOfServerResponse(t *testing.T) {
	for _, tc := range []struct {
		sr                    googleapi.ServerResponse
		expected_content_size int
	}{
		{
			sr:                    googleapi.ServerResponse{},
			expected_content_size: 0,
		},
		{
			sr:                    googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"hello": {"to you"}}},
			expected_content_size: emptyStringSize + 5 + emptyStringArraySize + emptyStringSize + 6,
		},
		{
			sr:                    googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"a": {""}}},
			expected_content_size: emptyStringSize + 1 + emptyStringArraySize + emptyStringSize,
		},
		{
			sr:                    googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"": {":"}}},
			expected_content_size: emptyStringSize + emptyStringArraySize + emptyStringSize + 1,
		},
		{
			sr:                    googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"a": {"b1"}, "xyz": {"alpha"}}},
			expected_content_size: emptyStringSize + 1 + emptyStringArraySize + emptyStringSize + 2 + emptyStringSize + 3 + emptyStringArraySize + emptyStringSize + 5,
		},
	} {
		assert.Equal(t, tc.expected_content_size, contentSizeOfServerResponse(&tc.sr))
	}
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
	customMetadataFieldsContentSize := emptyStringSize + contentSizeOfString(&customMetadaField1) + emptyStringSize + contentSizeOfString(&customMetadaValue1) + emptyStringSize + contentSizeOfString(&customMetadaField2) + emptyStringSize + contentSizeOfString(&customMetadaValue2)

	m := gcs.MinObject{
		Name:            name,
		Size:            100,
		ContentEncoding: contentEncoding,
		Metadata:        customMetadataFields,
		Generation:      generation,
		MetaGeneration:  metaGeneration,
		Updated:         updated,
		CRC32C:          &crc32,
	}

	var expectedSize int = sizeOfEmptyMinObject
	expectedSize += len(name) + len(contentEncoding) + sizeOfUInt32
	expectedSize += customMetadataFieldsContentSize

	assert.Equal(t, expectedSize, NestedSizeOfGcsMinObject(&m))
}

func TestNestedSizeOfGcsFolder(t *testing.T) {
	// A nil folder has 0 size.
	assert.Equal(t, 0, NestedSizeOfGcsFolder(nil))

	// An empty folder has size of the struct alone.
	f1 := &gcs.Folder{}
	expectedSizeF1 := UnsafeSizeOf(f1)
	assert.Equal(t, expectedSizeF1, NestedSizeOfGcsFolder(f1))

	// A folder with a name has the struct size + name string length.
	f2 := &gcs.Folder{
		Name: "folder/name/",
	}
	expectedSizeF2 := UnsafeSizeOf(f2) + len("folder/name/")
	assert.Equal(t, expectedSizeF2, NestedSizeOfGcsFolder(f2))
}

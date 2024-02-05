// Copyright 2024 Google Inc. All Rights Reserved.
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
	"crypto/md5"
	"net/http"
	"testing"
	"time"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

func TestSizeof(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SizeofTest struct {
}

func init() { RegisterTestSuite(&SizeofTest{}) }

var (
	i            int
	intArray     []int
	b            byte
	emptyString  string = ""
	stringArray  []string
	stringIntMap map[string]int

	sizeOfInt               int
	sizeOfIntPtr            int
	sizeOfByte              int
	sizeOfEmptyString       int
	sizeOfEmptyIntArray     int
	sizeOfEmptyStringArray  int
	sizeOfEmptyStringIntMap int
	sizeOfEmptyStruct       int
	sizeOfEmptyGcsObject    int
	sizeOfEmptyMinObject    int
)

func init() {
	type emptyStruct struct{}

	sizeOfInt = int(unsafe.Sizeof(i))
	sizeOfIntPtr = int(unsafe.Sizeof(&i))
	sizeOfByte = int(unsafe.Sizeof(b))
	sizeOfEmptyString = int(unsafe.Sizeof(emptyString))
	sizeOfEmptyIntArray = int(unsafe.Sizeof(intArray))
	sizeOfEmptyStringArray = int(unsafe.Sizeof(stringArray))
	sizeOfEmptyStringIntMap = int(unsafe.Sizeof(stringIntMap))

	sizeOfEmptyStruct = int(unsafe.Sizeof(emptyStruct{}))
	AssertEq(0, sizeOfEmptyStruct)

	sizeOfEmptyGcsObject = int(unsafe.Sizeof(gcs.Object{}))
	sizeOfEmptyMinObject = int(unsafe.Sizeof(gcs.MinObject{}))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SizeofTest) TestUnsafeSizeOf() {

	for _, tc := range []struct {
		t             any
		expected_size int
	}{
		{
			t:             i,
			expected_size: sizeOfInt,
		},
		{
			t:             &i,
			expected_size: sizeOfIntPtr,
		},
		{
			t: struct {
			}{},
			expected_size: sizeOfEmptyStruct,
		},
		{
			t: struct {
				x int
			}{},
			expected_size: sizeOfEmptyStruct + sizeOfInt,
		},
		{
			t: struct {
				a          int
				b1, b2, b3 byte
				c          string
			}{},
			expected_size: sizeOfEmptyStruct + sizeOfInt + 3*sizeOfByte + 5 /*for-padding-for-alignment*/ + sizeOfEmptyString,
		},
		{
			t:             "",
			expected_size: sizeOfEmptyString,
		},
		{
			t:             "hello",
			expected_size: sizeOfEmptyString,
		},
		{
			t:             []int{1, 2, 3},
			expected_size: sizeOfEmptyIntArray,
		},
		{
			t:             []string{"few ", "fewfgwe", "", "fewawef"},
			expected_size: sizeOfEmptyStringArray,
		},
		{
			t:             map[string]int{"few ": 432, "fewfgwe": -21, "": 1, "fewawef": 0},
			expected_size: sizeOfEmptyStringIntMap,
		},
	} {
		calculatedSize := UnsafeSizeOf(&tc.t)
		AssertEq(tc.expected_size, calculatedSize)
	}
}

func (t *SizeofTest) TestContentSizeOfString() {
	for _, tc := range []struct {
		str                   string
		expected_content_size int
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
		AssertEq(tc.expected_content_size, contentSizeOfString(&tc.str))
		AssertEq(tc.expected_content_size+sizeOfEmptyString, nestedSizeOfString(&tc.str))
	}
}

func (t *SizeofTest) TestNestedContentSizeOfArrayOfStrings() {
	for _, tc := range []struct {
		strs                         []string
		expected_nested_content_size int
	}{
		{
			strs:                         []string{},
			expected_nested_content_size: 0,
		},
		{
			strs:                         []string{""},
			expected_nested_content_size: sizeOfEmptyString,
		},
		{
			strs:                         []string{"", ""},
			expected_nested_content_size: 2 * sizeOfEmptyString,
		},
		{
			strs:                         []string{"hello", ""},
			expected_nested_content_size: 2*sizeOfEmptyString + 5,
		},
		{
			strs:                         []string{"hello", "hello-world"},
			expected_nested_content_size: 2*sizeOfEmptyString + 5 + 11,
		},
	} {
		AssertEq(tc.expected_nested_content_size, nestedContentSizeOfArrayOfStrings(&tc.strs))
		AssertEq(tc.expected_nested_content_size+sizeOfEmptyStringArray, nestedSizeOfArrayOfStrings(&tc.strs))
	}
}

func (t *SizeofTest) TestNestedContentSizeOfStringToStringMap() {
	for _, tc := range []struct {
		m                            map[string]string
		expected_nested_content_size int
	}{
		{
			m:                            map[string]string{},
			expected_nested_content_size: 0,
		},
		{
			m:                            map[string]string{"hello": "to you"},
			expected_nested_content_size: sizeOfEmptyString + 5 + sizeOfEmptyString + 6,
		},
		{
			m:                            map[string]string{"a": ""},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyString,
		},
		{
			m:                            map[string]string{"": ":"},
			expected_nested_content_size: sizeOfEmptyString + sizeOfEmptyString + 1,
		},
		{
			m:                            map[string]string{"a": "b1", "xyz": "alpha"},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyString + 2 + sizeOfEmptyString + 3 + sizeOfEmptyString + 5,
		},
	} {
		AssertEq(tc.expected_nested_content_size, nestedContentSizeOfStringToStringMap(&tc.m))
	}
}

func (t *SizeofTest) TestNestedContentSizeOfStringToStringArrayMap() {
	for _, tc := range []struct {
		m                            map[string][]string
		expected_nested_content_size int
	}{
		{
			m:                            map[string][]string{},
			expected_nested_content_size: 0,
		},
		{
			m:                            map[string][]string{"hello": {"to you"}},
			expected_nested_content_size: sizeOfEmptyString + 5 + sizeOfEmptyStringArray + sizeOfEmptyString + 6,
		},
		{
			m:                            map[string][]string{"a": {""}},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyStringArray + sizeOfEmptyString,
		},
		{
			m:                            map[string][]string{"": {":"}},
			expected_nested_content_size: sizeOfEmptyString + sizeOfEmptyStringArray + sizeOfEmptyString + 1,
		},
		{
			m:                            map[string][]string{"a": {"b1"}, "xyz": {"alpha"}},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyStringArray + sizeOfEmptyString + 2 + sizeOfEmptyString + 3 + sizeOfEmptyStringArray + sizeOfEmptyString + 5,
		},
	} {
		AssertEq(tc.expected_nested_content_size, nestedContentSizeOfStringToStringArrayMap(&tc.m))
	}
}

func (t *SizeofTest) TestNestedContentSizeOfServerResponse() {
	for _, tc := range []struct {
		sr                           googleapi.ServerResponse
		expected_nested_content_size int
	}{
		{
			sr:                           googleapi.ServerResponse{},
			expected_nested_content_size: 0,
		},
		{
			sr:                           googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"hello": {"to you"}}},
			expected_nested_content_size: sizeOfEmptyString + 5 + sizeOfEmptyStringArray + sizeOfEmptyString + 6,
		},
		{
			sr:                           googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"a": {""}}},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyStringArray + sizeOfEmptyString,
		},
		{
			sr:                           googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"": {":"}}},
			expected_nested_content_size: sizeOfEmptyString + sizeOfEmptyStringArray + sizeOfEmptyString + 1,
		},
		{
			sr:                           googleapi.ServerResponse{HTTPStatusCode: 200, Header: http.Header{"a": {"b1"}, "xyz": {"alpha"}}},
			expected_nested_content_size: sizeOfEmptyString + 1 + sizeOfEmptyStringArray + sizeOfEmptyString + 2 + sizeOfEmptyString + 3 + sizeOfEmptyStringArray + sizeOfEmptyString + 5,
		},
	} {
		AssertEq(tc.expected_nested_content_size, nestedContentSizeOfServerResponse(&tc.sr))
	}
}

func (t *SizeofTest) TestNestedSizeOfGcsObject() {
	const name string = "my-object"
	const contentType string = "plain/bin/gzip"
	const contentLanguage string = "en/fr/jp"
	const cacheControl string = "off/on"
	const contentEncoding string = "gzip/none"
	const owner string = "my-user"
	var md5Value [md5.Size]byte = [md5.Size]byte{0, 2, 42, 2, 4, 54, 3}
	var crc32 uint32 = 758734925
	var mediaLink string = "media-link"
	var generation int64 = 858734898
	var metaGeneration int64 = 858734899
	const storageClass string = "standard"
	deleted := time.Now()
	updated := time.Now()
	const componentCount int64 = 1
	const contentDisposition string = "my-content-disposition"
	const customTime string = "my-custom-time"
	const eventBasedHold bool = true
	customMetadaField1 := "google-symlink"
	customMetadaValue1 := "true"
	customMetadaField2 := "google-xyz-field"
	customMetadaValue2 := "google-symlink"
	customMetadataFields := map[string]string{customMetadaField1: customMetadaValue1, customMetadaField2: customMetadaValue2}
	customMetadataFieldsContentSize := nestedSizeOfString(&customMetadaField1) + nestedSizeOfString(&customMetadaValue1) + nestedSizeOfString(&customMetadaField2) + nestedSizeOfString(&customMetadaValue2)
	customAcls := []*storagev1.ObjectAccessControl{
		{
			Bucket:     "my-bucket",
			Domain:     "my-domain",
			Email:      "my-email@my-domain.com",
			Entity:     "my-entity",
			EntityId:   "my-entity-id",
			Etag:       "my-etag",
			Generation: generation,
			Id:         "object-id",
			Kind:       "object-kind",
			Object:     "my-object",
			ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
				ProjectNumber: "78358753894",
				Team:          "project-team",
				ForceSendFields: []string{
					"field1", "field2", "field3",
				},
			},
			Role:            "my-role",
			SelfLink:        "",
			ServerResponse:  googleapi.ServerResponse{},
			ForceSendFields: []string{},
			NullFields:      []string{},
		},
	}

	o := gcs.Object{
		Name:               name,
		ContentType:        contentType,
		ContentLanguage:    contentLanguage,
		CacheControl:       cacheControl,
		Owner:              owner,
		Size:               100,
		ContentEncoding:    contentEncoding,
		MD5:                &md5Value,
		CRC32C:             &crc32,
		MediaLink:          mediaLink,
		Metadata:           customMetadataFields,
		Generation:         generation,
		MetaGeneration:     metaGeneration,
		StorageClass:       storageClass,
		Deleted:            deleted,
		Updated:            updated,
		ComponentCount:     componentCount,
		ContentDisposition: contentDisposition,
		CustomTime:         customTime,
		EventBasedHold:     eventBasedHold,
		Acl:                customAcls,
	}

	var expectedSize int = sizeOfEmptyGcsObject
	expectedSize += len(contentType) + len(name) + len(contentLanguage) + len(cacheControl) + len(contentEncoding) + len(owner) + len(mediaLink) + len(storageClass) + len(contentDisposition) + len(customTime)
	expectedSize += md5.Size // for MD5 [md5.Size]byte
	expectedSize += 4        // for CRC 32 uint32
	expectedSize += customMetadataFieldsContentSize
	expectedSize += nestedContentSizeOfArrayOfAclPointers(&customAcls)

	AssertEq(expectedSize, NestedSizeOfGcsObject(&o))
}

func (t *SizeofTest) TestNestedSizeOfMinObject() {
	o := gcs.MinObject{}

	AssertEq(sizeOfEmptyMinObject, NestedSizeOfMinObject(&o))
}

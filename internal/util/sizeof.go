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
	"reflect"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

var (
	// pointerSize represents the size of the pointer of any type.
	pointerSize                             int
	emptyStringSize                         int
	emptyStringArraySize                    int
	emptyObjectAccessControlSize            int
	emptyObjectAccessControlProjectTeamSize int
)

func init() {
	var i int
	pointerSize = int(reflect.TypeOf(&i).Size())
	var s string
	emptyStringSize = int(reflect.TypeOf(s).Size())
	var sArray []string
	emptyStringArraySize = int(reflect.TypeOf(sArray).Size())
	var emptyObjectAccessControl storagev1.ObjectAccessControl
	emptyObjectAccessControlSize = int(reflect.TypeOf(emptyObjectAccessControl).Size())
	var emptyObjectAccessControlProjectTeam storagev1.ObjectAccessControlProjectTeam
	emptyObjectAccessControlProjectTeamSize = int(reflect.TypeOf(emptyObjectAccessControlProjectTeam).Size())
}

// Definitions/conventions (not based on a standard, but just made up for convenience).
//   1. Raw/unsafe size: This is the size of empty
//      data structures, i.e. size of memory, for data
//      structures when just initialized, without any
//      content filled into them.
//      This is same as what is returned by unsafe.Sizeof(...)
//
//      Built-in types ([u]int*, bool, time.*) have the same
//      raw/unsafe size as their actual space in memory.
//
//      More complex types like string, array/slice have a fixed
//      /standard raw/unsafe size e.g. string is 16 bytes (8 for
//      holding pointer to its content, 8 bytes for holding length).
//      But it doesn't account for string's content, so any string,
//  	be it empty-string, or "hello" or a million-char string, all
//      have raw/unsafe size of 16.
//
//      Pointers have a raw/unsafe size of 8 bytes on 64-bit platforms.
//
//      Similarly, an array/slice/map etc. have small fixed Raw/unsafe size like 16 or 32 bytes.
//
//      Raw/unsafe size of a struct is the sum of Raw/unsafe sizes of all its constituents.
//
//   2. Content-size: This is the additional size added when some content is
//      added/set to a variable/pointer/struct or to any of its constituents/members.
//      This is calculated recursively.
//
// 		Content size of built-ins such as integers/booleans is 0.
//
//      It is contributed by the content-sizes of the struct/pointer/slice/map members.
//
//      For example, for a string, this is same as the length of the
//       string e.g. 5 for "hello", but 0 for "", and so on.
//
//      For a pointer, this is same as the full-size of the
//       object pointed to by that pointer. E.g. a `*int64` has no content, when it's nil,
//       But has non-zero content when it's not nil.
//
//      In case of structs, this is the sum of the content-sizes of all its constituents.
//      In case of slices, this is the sum of the full-sizes of all its members.
//      In case of maps, this is the sum of the full-sizes of all its keys and values.
//
//   3. Nested-size: The full size of a struct/object
//      in memory, including the full/nested sizes of all its
//      members.
//
//      This is the same as the sum of Raw/unsafe size and the Content-size of the object in question.

// All functions are taking pointers to avoid the cost of
// creating copies of the passed objects.
// Though it's not reflected in the function names, it is assumed
// that they are calculating the sizes of
// the struct/object/built-int pointed to by the passed pointer.

// UnsafeSizeOf returns the unsafe.Sizeof or
// raw-size of the object pointed to by the given pointer.
// It does not account for the pointer's size on memory itself.
// For e.g. if an int is 8 bytes and an empty string is 16 bytes,
// then UnsafeSizeOf(&struct{int, string})
// return 24 (8+16).
func UnsafeSizeOf[T any](ptr *T) int {
	if ptr == nil {
		return 0
	}
	return int(reflect.TypeOf(*ptr).Size())
}

func contentSizeOfString(s *string) int {
	if s == nil {
		return 0
	}
	return len(*s)
}

func contentSizeOfArrayOfStrings(arr *[]string) (size int) {
	if arr == nil {
		return
	}

	for _, str := range *arr {
		size += emptyStringSize + contentSizeOfString(&str)
	}
	return
}

func contentSizeOfStringToStringMap(m *map[string]string) (size int) {
	if m == nil {
		return
	}

	for k, v := range *m {
		size += emptyStringSize + contentSizeOfString(&k)
		size += emptyStringSize + contentSizeOfString(&v)
	}
	return
}

func contentSizeOfStringToStringArrayMap(m *map[string][]string) (size int) {
	if m == nil {
		return
	}

	for k, v := range *m {
		size += emptyStringSize + contentSizeOfString(&k)
		size += emptyStringArraySize + contentSizeOfArrayOfStrings(&v)
	}
	return
}

func contentSizeOfServerResponse(sr *googleapi.ServerResponse) (size int) {
	if sr == nil {
		return
	}

	// Account for integer members - HTTPStatusCode.
	// Nothing to be added here, as explained in the documentation at the top.

	// Account for map members.
	size += contentSizeOfStringToStringArrayMap((*map[string][]string)(&sr.Header))

	return
}

func contentSizeOfObjectAccessControlProjectTeam(oacpt *storagev1.ObjectAccessControlProjectTeam) (size int) {
	if oacpt == nil {
		return
	}

	// Account for string members.
	for _, strPtr := range []*string{
		&oacpt.ProjectNumber, &oacpt.Team,
	} {
		size += contentSizeOfString(strPtr)
	}

	// Account for string-array members.
	for _, strArrayPtr := range []*[]string{
		&oacpt.ForceSendFields, &oacpt.NullFields,
	} {
		size += contentSizeOfArrayOfStrings(strArrayPtr)
	}

	return
}

func contentSizeOfObjectAccessControl(acl *storagev1.ObjectAccessControl) (size int) {
	if acl == nil {
		return
	}

	// Account for string members.
	for _, strPtr := range []*string{
		&acl.Bucket, &acl.Domain, &acl.Email, &acl.Entity,
		&acl.EntityId, &acl.Etag, &acl.Id, &acl.Kind,
		&acl.Object, &acl.Role, &acl.SelfLink} {
		size += contentSizeOfString(strPtr)
	}

	// Account for integer-members - Generation.
	// Nothing to be added here as described in the documentation at the top.

	// Account for pointer-members.
	size += emptyObjectAccessControlProjectTeamSize + contentSizeOfObjectAccessControlProjectTeam(acl.ProjectTeam)

	// Account for other struct members.
	size += contentSizeOfServerResponse(&acl.ServerResponse)

	// Account for string-array members.
	size += contentSizeOfArrayOfStrings(&acl.ForceSendFields)
	size += contentSizeOfArrayOfStrings(&acl.NullFields)

	return
}

func contentSizeOfArrayOfAclPointers(acls *[]*storagev1.ObjectAccessControl) (size int) {
	if acls == nil {
		return
	}

	for _, acl := range *acls {
		// We could use unsafe.Sizeof(&acl) here instead of defining
		// an unnecessary constant pointerSize, but that would
		// have added cost of an extra unsafe.Sizeof on each
		// member.
		size += pointerSize + emptyObjectAccessControlSize + contentSizeOfObjectAccessControl(acl)
	}
	return
}

// NestedSizeOfGcsMinObject returns the full nested memory size
// of the gcs.MinObject pointed by the passed pointer.
// Improvement scope: This can be generalized to a general-struct
// but that needs better understanding of the reflect package
// and other related packages.
func NestedSizeOfGcsMinObject(m *gcs.MinObject) (size int) {
	if m == nil {
		return
	}

	// Get raw size of the structure.
	size = UnsafeSizeOf(m)

	// Account for string members.
	for _, strPtr := range []*string{&m.Name, &m.ContentEncoding} {
		size += contentSizeOfString(strPtr)
	}

	// Account for integer members - Size, Generation, MetaGeneration.
	// Account for time members - Updated.
	// Nothing to be added for any built-in types - already accounted for in unsafeSizeOf(o).

	// Account for map members.
	size += contentSizeOfStringToStringMap(&m.Metadata)

	return
}

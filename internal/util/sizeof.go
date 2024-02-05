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

	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

var (
	// pointerSize represents the size of the pointer of any type.
	pointerSize int
)

func init() {
	var i int
	pointerSize = int(reflect.TypeOf(&i).Size())
}

// Definitions/conventions (not based on a standard, but just made up for convenience).
//   1. Raw/unsafe size: This is the size of empty
//      data structures, i.e. size of memory, for data
//      structures when just initialized, without any
//      content filled into them.
//      This is same as what is returned by unsafe.Sizeof(...)
//
//      Built-in types ([u]int*, bool, time.*) have the same
//      same raw/unsafe size as their actual space in memory.
//
//      More complex types like string, array/slice have a fixed
//      /standard raw/unsafe size e.g. string is 16 bytes (8 for
//      holding pointer to its content, 8 bytes for holding length).
//      But it doesn't account for string's content, so any string,
//  	be it emoty-string, or "hello" or a million-char string, all
//      have Raw/unsafe size of 16.
//
//      Pointers have a raw/unsafe size of 8 bytes on 64-bit platforms.
//
//      Similarly, an array/slice/map etc. have small fixed Raw/unsafe size like 16 or 32 bytes.
//
//      Raw/unsafe size of a struct is the sum of Raw/unsafe sizes of all its constituents.
//
//   2. Content-size (or nested-content-size): This is the additional size added when some content is
//      added/set to a variable/pointer/struct or to any of its constituents/members.
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
//   3. Full-size (or nested-size): The full size of a struct/object
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

func nestedSizeOfString(s *string) int {
	return UnsafeSizeOf(s) + contentSizeOfString(s)
}

func nestedContentSizeOfArrayOfStrings(arr *[]string) (size int) {
	if arr == nil {
		return
	}

	for _, str := range *arr {
		size += nestedSizeOfString(&str)
	}
	return
}

func nestedSizeOfArrayOfStrings(arr *[]string) int {
	return UnsafeSizeOf(arr) + nestedContentSizeOfArrayOfStrings(arr)
}

func nestedContentSizeOfStringToStringMap(m *map[string]string) (size int) {
	if m == nil {
		return
	}

	for k, v := range *m {
		size += nestedSizeOfString(&k)
		size += nestedSizeOfString(&v)
	}
	return
}

func nestedContentSizeOfStringToStringArrayMap(m *map[string][]string) (size int) {
	if m == nil {
		return
	}

	for k, v := range *m {
		size += nestedSizeOfString(&k)
		size += nestedSizeOfArrayOfStrings(&v)
	}
	return
}

func nestedContentSizeOfServerResponse(sr *googleapi.ServerResponse) (size int) {
	if sr == nil {
		return
	}

	// integer members - HTTPStatusCode
	// nothing to be added here

	// map members
	size += nestedContentSizeOfStringToStringArrayMap((*map[string][]string)(&sr.Header))

	return
}

func nestedSizeOfObjectAccessControlProjectTeam(oacpt *storagev1.ObjectAccessControlProjectTeam) (size int) {
	if oacpt == nil {
		return
	}

	size = UnsafeSizeOf(oacpt)

	// string members
	for _, strPtr := range []*string{
		&oacpt.ProjectNumber, &oacpt.Team,
	} {
		size += contentSizeOfString(strPtr)
	}

	// string-array members
	for _, strArrayPtr := range []*[]string{
		&oacpt.ForceSendFields, &oacpt.NullFields,
	} {
		size += nestedContentSizeOfArrayOfStrings(strArrayPtr)
	}

	return
}

func nestedSizeOfObjectAccessControl(acl *storagev1.ObjectAccessControl) (size int) {
	if acl == nil {
		return
	}

	size = UnsafeSizeOf(acl)

	// string members
	for _, strPtr := range []*string{
		&acl.Bucket, &acl.Domain, &acl.Email, &acl.Entity,
		&acl.EntityId, &acl.Etag, &acl.Id, &acl.Kind,
		&acl.Object, &acl.Role, &acl.SelfLink} {
		size += contentSizeOfString(strPtr)
	}

	// integer-members - Generation
	// nothing to be added here

	// pointer-members - ProjectTeam
	size += nestedSizeOfObjectAccessControlProjectTeam(acl.ProjectTeam)

	// other struct members
	size += nestedContentSizeOfServerResponse(&acl.ServerResponse)

	// string-array members
	size += nestedContentSizeOfArrayOfStrings(&acl.ForceSendFields)
	size += nestedContentSizeOfArrayOfStrings(&acl.NullFields)

	return
}

func nestedContentSizeOfArrayOfAclPointers(acls *[]*storagev1.ObjectAccessControl) (size int) {
	if acls == nil {
		return
	}

	for _, acl := range *acls {
		// We could use unsafe.Sizeof(&acl) here instead of defining
		// an unnecessary constant pointerSize, but that would
		// have added cost of an extra unsafe.Sizeof on each
		// member.
		size += pointerSize + nestedSizeOfObjectAccessControl(acl)
	}
	return
}

// NestedSizeOfGcsObject returns the full nested memory size
// of the gcs.Object pointed by the passed pointer.
// Improvement scope: This can be generalized to a general-struct
// but that needs better understanding of the reflect package
// and other related packages.
func NestedSizeOfGcsObject(o *gcs.Object) (size int) {
	if o == nil {
		return
	}

	// get raw size of the structure.
	size = UnsafeSizeOf(o)

	// string members
	for _, strPtr := range []*string{
		&o.Name, &o.ContentType, &o.ContentLanguage, &o.CacheControl,
		&o.Owner, &o.ContentEncoding, &o.MediaLink, &o.StorageClass,
		&o.ContentDisposition, &o.CustomTime} {
		size += contentSizeOfString(strPtr)
	}

	// integer-pointer members
	size += UnsafeSizeOf(o.CRC32C)
	// pointer-to-integer-array members
	size += UnsafeSizeOf(o.MD5)

	// integer members - Size, Generation, MetaGeneration, ComponentCount
	// time members - Deleted, Updated
	// boolean members - EventBasedHold
	// nothing to be added for any built-in types - already accounted for in unsafeSizeOf(o)

	// map members
	size += nestedContentSizeOfStringToStringMap(&o.Metadata)

	// slice members
	size += nestedContentSizeOfArrayOfAclPointers(&o.Acl)

	return
}

// NestedSizeOfMinObject returns the full nested memory size
// of the gcs.MinObject pointed by the passed pointer.
// The logic used here is a subset of the logic used in NestedSizeOfGcsObject.
func NestedSizeOfMinObject(o *gcs.MinObject) (size int) {
	if o == nil {
		return
	}

	// get raw size of the structure.
	size = UnsafeSizeOf(o)

	// string members
	for _, strPtr := range []*string{
		&o.Name, &o.ContentEncoding} {
		size += contentSizeOfString(strPtr)
	}

	// integer members - Size, Generation, MetaGeneration
	// time members - Updated
	// boolean members - EventBasedHold
	// nothing to be added for any built-in types - already accounted for in unsafeSizeOf(o)

	// map members
	size += nestedContentSizeOfStringToStringMap(&o.Metadata)

	return
}

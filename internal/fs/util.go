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

package fs

import (
	"reflect"
)

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

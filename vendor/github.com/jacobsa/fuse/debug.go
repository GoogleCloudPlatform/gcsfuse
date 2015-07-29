// Copyright 2015 Google Inc. All Rights Reserved.
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

package fuse

import (
	"fmt"
	"reflect"
)

func describeRequest(op interface{}) (s string) {
	v := reflect.ValueOf(op).Elem()
	t := v.Type()

	// Find the inode number involved, if possible.
	var inodeDesc string
	if f := v.FieldByName("Inode"); f.IsValid() {
		inodeDesc = fmt.Sprintf("(inode=%v)", f.Interface())
	}

	// Use the type name.
	s = fmt.Sprintf("%s%s", t.Name(), inodeDesc)

	return
}

func describeResponse(op interface{}) (s string) {
	return describeRequest(op)
}

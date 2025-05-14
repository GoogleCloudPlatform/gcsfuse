// Copyright 2025 Google LLC
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
	"github.com/jacobsa/fuse/fuseops"
)

// OpenMode represents the file access mode.
type OpenMode int

// Available file open modes.
const (
	Read OpenMode = iota
	Write
	Append
)

// FileOpenMode analyzes the open flags to determine the file's open mode.
// It returns Read, Write, or Append.
//
// GCSFuse needs to distinguish between the append mode, readonly mode and modes where
// writes are supported.
// read vs write is required to initialize the writeHandle count currently in fileHandle.
// The main difference of r vs (r+, w, w+) is the support for reads which is
// implicitly handled by kernel. Hence combining r+, w, w+ as writes.
// Same goes for a vs a+, hence grouped as append.
func FileOpenMode(op *fuseops.OpenFileOp) OpenMode {
	switch {
	case op.OpenFlags.IsAppend():
		return Append
	case op.OpenFlags.IsReadOnly():
		return Read
	default:
		return Write
	}
}

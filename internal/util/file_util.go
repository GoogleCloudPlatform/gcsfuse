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
// Since there are certain flows with GCSFuse where the append flow is treated differently
// than regular writes, this is included. However, no special handling of a vs a+ mode is
// required, as the differentiating behavior if whether or not reads will be supported is
// implicitly handled by the kernel.
// We need to distinguish the read-only mode as well since we initialize the writeHandleCount
// accordingly.
// All modes where writes are supported , are clubbed into the write mode as we need to
// differentiate it from the read-only mode. No special handling of w vs w+ vs r+ mode is
// required, as the differentiating behavior if whether or not reads will be supported is
// implicitly handled by the kernel.
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

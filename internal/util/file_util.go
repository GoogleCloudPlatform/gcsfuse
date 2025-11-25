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

// Available file access modes, corresponding to O_RDONLY, O_WRONLY, O_RDWR.
const (
	ReadOnly int = iota
	WriteOnly
	ReadWrite
)

// Available file flags.
const (
	O_APPEND int = 1 << iota // O_APPEND
	O_DIRECT                 // O_DIRECT
)

// OpenMode represents the file open mode.
type OpenMode struct {
	// accessMode defines the mutually exclusive access modes for opening a file.
	accessMode int

	// fileFlags defines flags that modify the open/read/write behavior.
	// These can be combined using bitwise OR.
	fileFlags int
}

// NewOpenMode constructs OpenMode given the accessMode and fileFlags.
func NewOpenMode(accessMode int, fileFlags int) OpenMode {
	return OpenMode{
		accessMode: accessMode,
		fileFlags:  fileFlags,
	}
}

func (om OpenMode) AccessMode() int {
	return om.accessMode
}

func (om OpenMode) FileFlags() int {
	return om.fileFlags
}

func (om *OpenMode) SetAccessMode(accessMode int) {
	om.accessMode = accessMode
}

func (om *OpenMode) SetFileFlags(fileFlags int) {
	om.fileFlags = fileFlags
}

// IsAppend checks if the file was opened in append mode.
// A file is considered in append mode (as suited for GCSFuse logic) if it is not
// opened as read-only and the O_APPEND flag is set.
func (om OpenMode) IsAppend() bool {
	return om.accessMode != ReadOnly && om.fileFlags&O_APPEND != 0
}

// IsDirect checks if the O_DIRECT flag is set, indicating that I/O should
// bypass the kernel's page cache.
func (om OpenMode) IsDirect() bool {
	return om.fileFlags&O_DIRECT != 0
}

// OpenFlagAttributes provides an abstraction for the open flags received from
// the FUSE kernel. This interface is necessary because the concrete type for open
// flags in `jacobsa/fuse` (e.g., in `fuseops.OpenFileOp` and `fuseops.CreateFileOp`)
// resides in an internal package and cannot be directly referenced.
//
// This abstraction allows a single function, `FileOpenMode`, to process flags
// from different FUSE operations and also simplifies unit testing.
type OpenFlagAttributes interface {
	IsReadOnly() bool
	IsWriteOnly() bool
	IsReadWrite() bool
	IsAppend() bool
	IsDirect() bool
}

// Function to obtain the mutually exclusive access mode based on the flags passed.
func getAccessMode(flags OpenFlagAttributes) int {
	if flags.IsReadOnly() {
		return ReadOnly
	} else if flags.IsWriteOnly() {
		return WriteOnly
	} else {
		return ReadWrite
	}
}

// Combine behavior-modifying file flags like O_DIRECT and O_APPEND.
func getFileFlags(flags OpenFlagAttributes) int {
	var fileFlags int
	if flags.IsAppend() {
		fileFlags |= O_APPEND
	}
	if flags.IsDirect() {
		fileFlags |= O_DIRECT
	}
	return fileFlags
}

// FileOpenMode analyzes the open flags to determine the file's open mode.
func FileOpenMode(flags OpenFlagAttributes) OpenMode {
	accessMode := getAccessMode(flags)
	fileFlags := getFileFlags(flags)
	return NewOpenMode(accessMode, fileFlags)
}

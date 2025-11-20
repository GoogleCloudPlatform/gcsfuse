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
	// AccessMode defines the mutually exclusive access modes for opening a file.
	AccessMode int

	// FileFlag defines flags that modify the open/read/write behavior.
	// These can be combined using bitwise OR.
	FileFlags int
}

func (om OpenMode) IsAppend() bool {
	return om.FileFlags&O_APPEND != 0
}

func (om OpenMode) IsDirect() bool {
	return om.FileFlags&O_DIRECT != 0
}

// OpenFlagAttributes defines the methods required from the open flags.
type OpenFlagAttributes interface {
	IsReadOnly() bool
	IsWriteOnly() bool
	IsReadWrite() bool
	IsAppend() bool
	IsDirect() bool
}

// Function to obtain the mutually exclusive access mode.
func getAccessMode(flags OpenFlagAttributes) int {
	if flags.IsReadOnly() {
		return ReadOnly
	} else if flags.IsWriteOnly() {
		return WriteOnly
	} else {
		return ReadWrite
	}
}

// Combine file flags e.g. O_DIRECT,O_APPEND.
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
	openMode := OpenMode{}
	// Construct the openMode based on the received flags
	openMode.AccessMode = getAccessMode(flags)
	openMode.FileFlags = getFileFlags(flags)

	return openMode
}

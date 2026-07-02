// Copyright 2026 Google LLC
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

package inode

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestPrintNewSizes(t *testing.T) {
	fmt.Printf("NEW_SIZEOF_FileInode: %d\n", unsafe.Sizeof(FileInode{}))
	fmt.Printf("NEW_SIZEOF_dirInode: %d\n", unsafe.Sizeof(dirInode{}))
	fmt.Printf("NEW_SIZEOF_SymlinkInode: %d\n", unsafe.Sizeof(SymlinkInode{}))
}

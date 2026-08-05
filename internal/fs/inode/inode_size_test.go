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
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

// TestInodeSizes checks the sizes of the FileInode and dirInode structs to prevent memory regressions.
//
// Background:
// Since millions of inodes can be cached in memory during large-scale GCS mounts, keeping their memory
// footprint minimized is critical. In Go, struct fields are laid out sequentially. The compiler
// inserts alignment padding bytes to satisfy CPU alignment constraints (e.g. if a 1-byte bool is followed
// by an 8-byte pointer, the compiler inserts 7 padding bytes).
//
// Optimization Principle:
// To eliminate this padding and achieve the mathematical minimum size:
// 1. Group 8-byte aligned fields (pointers, interfaces, structs, int64, uint64) first.
// 2. Group 4-byte fields (int32, uint32, atomic.Int32) next.
// 3. Group 1-byte fields (booleans, uint8) at the very end.
//
// Expected Sizes:
// - FileInode: 488 bytes (0 padding bytes).
// - dirInode:  344 bytes (4 trailing padding bytes due to 8-byte struct alignment constraint).
//
// If this test fails, a field was added or reordered in a way that introduced compiler padding.
// Please rearrange the fields of the struct in dir.go or file.go according to the principle above.
func TestInodeSizes(t *testing.T) {
	const expectedFileInodeSize = 488
	const expectedDirInodeSize = 344

	fileInodeSize := unsafe.Sizeof(FileInode{})
	dirInodeSize := unsafe.Sizeof(dirInode{})

	assert.LessOrEqual(t, fileInodeSize, uintptr(expectedFileInodeSize), "FileInode size exceeded expected limit (possible regression in field alignment or extra fields)")
	assert.LessOrEqual(t, dirInodeSize, uintptr(expectedDirInodeSize), "dirInode size exceeded expected limit (possible regression in field alignment or extra fields)")
}

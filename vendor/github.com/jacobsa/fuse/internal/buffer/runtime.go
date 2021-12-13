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

package buffer

import "unsafe"

//go:noescape

// Zero the n bytes starting at p.
//
// REQUIRES: the region does not contain any Go pointers.
//go:linkname jacobsa_fuse_memclr runtime.memclrNoHeapPointers
func jacobsa_fuse_memclr(p unsafe.Pointer, n uintptr)

//go:noescape

// Copy from src to dst, allowing overlap.
//go:linkname jacobsa_fuse_memmove runtime.memmove
func jacobsa_fuse_memmove(dst unsafe.Pointer, src unsafe.Pointer, n uintptr)

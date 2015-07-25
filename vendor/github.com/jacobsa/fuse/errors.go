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
	"syscall"

	"github.com/jacobsa/bazilfuse"
)

const (
	// Errors corresponding to kernel error numbers. These may be treated
	// specially by fuseops.Op.Respond methods.
	EEXIST    = bazilfuse.EEXIST
	EINVAL    = bazilfuse.Errno(syscall.EINVAL)
	EIO       = bazilfuse.EIO
	ENOENT    = bazilfuse.ENOENT
	ENOSYS    = bazilfuse.ENOSYS
	ENOTDIR   = bazilfuse.Errno(syscall.ENOTDIR)
	ENOTEMPTY = bazilfuse.Errno(syscall.ENOTEMPTY)
)

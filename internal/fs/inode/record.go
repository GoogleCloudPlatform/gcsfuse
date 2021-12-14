// Copyright 2021 Google Inc. All Rights Reserved.
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
	"strings"

	"github.com/jacobsa/gcloud/gcs"
)

type Type int

var (
	UnknownType     Type = 0
	SymlinkType     Type = 1
	RegularFileType Type = 2
	ExplicitDirType Type = 3
	ImplicitDirType Type = 4
)

// An inode record is an GCS object backing up the inode, or empty indicating
// the inode is an implicit directory.
type Record struct {
	Object *gcs.Object
}

// Return true if the inode record exists.
func (r *Record) Exists() bool {
	return r != nil
}

// Return the inode type of the inode record.
func (r *Record) Type() Type {
	switch {
	case r == nil:
		return UnknownType
	case r.Object == nil:
		return ImplicitDirType
	case strings.HasSuffix(r.Object.Name, "/"):
		return ExplicitDirType
	case IsSymlink(r.Object):
		return SymlinkType
	default:
		return RegularFileType
	}
}

// Copyright 2021 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type Type int

const (
	UnknownType     Type = 0
	SymlinkType     Type = 1
	RegularFileType Type = 2
	ExplicitDirType Type = 3
	ImplicitDirType Type = 4
	NonexistentType Type = 5
)

// Core contains critical information about an inode before its creation.
type Core struct {
	// The full name of the file or directory. Required for all inodes.
	FullName Name

	// The bucket that backs up the inode. Required for all inodes except the
	// base directory that holds all the buckets mounted.
	Bucket *gcsx.SyncerBucket

	// The GCS object in the bucket above that backs up the inode. Can be empty
	// if the inode is the base directory or an implicit directory.
	MinObject *gcs.MinObject

	// The GCS folder in the hierarchical bucket that backs up the inode.
	Folder *gcs.Folder

	// Specifies a local object which is not yet synced to GCS.
	Local bool
}

// Exists returns true iff the back object exists implicitly or explicitly.
func (c *Core) Exists() bool {
	return c != nil
}

func (c *Core) Type() metadata.Type {
	switch {
	case c == nil:
		return metadata.UnknownType
	case c.MinObject == nil && c.Folder == nil && !c.Local:
		return metadata.ImplicitDirType
	case c.FullName.IsDir():
		return metadata.ExplicitDirType
	case IsSymlink(c.MinObject):
		return metadata.SymlinkType
	default:
		return metadata.RegularFileType
	}
}

// SanityCheck returns an error if the object is conflicting with itself, which
// means the metadata of the file system is broken.
func (c Core) SanityCheck() error {
	if c.Folder != nil && c.FullName.objectName != c.Folder.Name {
		return fmt.Errorf("inode name %q mismatches folder name %q", c.FullName, c.Folder.Name)
	}

	if c.MinObject != nil && c.FullName.objectName != c.MinObject.Name {
		return fmt.Errorf("inode name %q mismatches object name %q", c.FullName, c.MinObject.Name)
	}

	if c.MinObject == nil && !c.Local && !c.FullName.IsDir() {
		return fmt.Errorf("object missing for %q", c.FullName)
	}

	return nil
}

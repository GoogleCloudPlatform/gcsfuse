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
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
)

// Core contains critical information about an inode before its creation.
type Core struct {
	// The full name of the file or directory. Required for all inodes.
	FullName Name

	// The bucket that backs up the inode. Required for all inodes except the
	// base directory that holds all the buckets mounted.
	Bucket gcsx.SyncerBucket

	// The GCS object in the bucket above that backs up the inode. Can be empty
	// if the inode is the base directory or an implicit directory.
	Object *gcs.Object

	// True iff the inode is an implicit directory.
	ImplicitDir bool
}

// Exists returns true iff the back object exists implicitly or explicitly.
func (c Core) Exists() bool {
	IsExplicitFileOrDir := c.Object != nil
	IsImplicitDir := c.ImplicitDir
	IsBucketRootDir :=
		c.FullName.LocalName() != "" && c.FullName.IsBucketRoot()
	return IsExplicitFileOrDir || IsImplicitDir || IsBucketRootDir
}

// SanityCheck returns an error if the object is conflicting with itself, which
// means the metadata of the file system is broken.
func (c Core) SanityCheck() error {
	if c.Object != nil {
		if c.ImplicitDir {
			return fmt.Errorf("directory backed by %q is not implicit", c.Object.Name)
		}
		if c.FullName.objectName != c.Object.Name {
			return fmt.Errorf("%q backed by %q inconsistently", c.FullName, c.Object.Name)
		}
	}
	return nil
}

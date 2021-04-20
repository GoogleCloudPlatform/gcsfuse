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
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
)

// BackObject is the object backing up the inode. A back object for a file inode
// must be a GCS object. A back object for a directory inode can be a GCS object
// or an implicit object deduced from other GCS objects.
type BackObject struct {
	// The GCS bucket that owns the back object
	Bucket gcsx.SyncerBucket

	// The full name of the file or directory.
	//
	// Guaranteed to be present only if Exists().
	FullName Name

	// The backing gcs object, if any. If the object is not found or exists only
	// as an implicit directory, or is the root directory of a bucket, this is
	// nil.
	Object *gcs.Object

	// Does the back object exist as a directory implicitly defined by its own
	// descendents? Meaningful only if Object is nil and implicit directories are
	// enabled for the parent inode.
	ImplicitDir bool
}

// Exists returns true iff the back object exists implicitly or explicitly.
func (bo BackObject) Exists() bool {
	IsExplicitFileOrDir := bo.Object != nil
	IsImplicitDir := bo.ImplicitDir
	IsBucketRootDir :=
		bo.FullName.LocalName() != "" && bo.FullName.IsBucketRoot()
	return IsExplicitFileOrDir || IsImplicitDir || IsBucketRootDir
}

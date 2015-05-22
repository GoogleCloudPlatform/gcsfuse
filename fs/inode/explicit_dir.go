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

package inode

import (
	"time"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
)

// An inode representing a directory backed by an object in GCS with a specific
// generation.
type ExplicitDirInode interface {
	DirInode

	// Return the object generation number from which this inode was branched.
	SourceGeneration() int64
}

// Create an explicit dir inode backed by the supplied object. See notes on
// NewDirInode for more.
func NewExplicitDirInode(
	id fuseops.InodeID,
	o *gcs.Object,
	attrs fuseops.InodeAttributes,
	implicitDirs bool,
	typeCacheTTL time.Duration,
	bucket gcs.Bucket,
	clock timeutil.Clock) (d ExplicitDirInode) {
	wrapped := NewDirInode(
		id,
		o.Name,
		attrs,
		implicitDirs,
		typeCacheTTL,
		bucket,
		clock)

	d = &explicitDirInode{
		dirInode:   wrapped.(*dirInode),
		generation: o.Generation,
	}

	return
}

type explicitDirInode struct {
	*dirInode
	generation int64
}

func (d *explicitDirInode) SourceGeneration() (gen int64) {
	gen = d.generation
	return
}

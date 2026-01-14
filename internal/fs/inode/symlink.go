// Copyright 2015 Google LLC
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
	"io"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/net/context"
)

// When this custom metadata key is present in an object record, it is to be
// treated as a symlink. For use in testing only; other users should detect
// this with IsSymlink.
const DeprecatedSymlinkMetadataKey = "gcsfuse_symlink_target"
const SymlinkMetadataKey = "goog-reserved-file-is-symlink"
const MAX_SYMLINK_TARGET_LENGTH = 4095

// IsSymlink Does the supplied object represent a symlink inode?
func IsSymlink(m *gcs.MinObject) bool {
	if m == nil {
		return false
	}

	_, ok1 := m.Metadata[DeprecatedSymlinkMetadataKey]
	_, ok2 := m.Metadata[SymlinkMetadataKey]
	return ok1 || ok2
}

// IsSymlinkWithOldSemantics is required for special handling required for
// existing symlinks in mounted bucket such as in the rename flow.
func IsSymlinkWithOldSemantics(m *gcs.MinObject) bool {
	if m == nil {
		return false
	}

	_, ok1 := m.Metadata[DeprecatedSymlinkMetadataKey]
	_, ok2 := m.Metadata[SymlinkMetadataKey]
	return ok1 && !ok2
}

func resolveSymlinkTarget(ctx context.Context, bucket *gcsx.SyncerBucket, m *gcs.MinObject) (string, error) {
	if m == nil {
		return "", fmt.Errorf("empty object passed. Symlink target cannot be resolved...")
	}

	if !IsSymlinkWithOldSemantics(m) {
		rc, err := bucket.NewReaderWithReadHandle(ctx, &gcs.ReadObjectRequest{
			Name:       m.Name,
			Generation: m.Generation,
		})
		if err != nil {
			return "", fmt.Errorf("NewReaderWithReadHandle: %w", err)
		}
		defer rc.Close()

		content, err := io.ReadAll(rc)
		if err != nil {
			return "", fmt.Errorf("ReadAll: %w", err)
		}
		if len(content) > MAX_SYMLINK_TARGET_LENGTH {
			return "", fmt.Errorf("maximum target length for a symlink reached.")
		}
		return string(content), nil

	} else if target, ok := m.Metadata[DeprecatedSymlinkMetadataKey]; ok {
		return target, nil
	}

	return "", fmt.Errorf("minobject does not have appropriate metadata set. Cannot resolve target...")
}

type SymlinkInode struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	id               fuseops.InodeID
	name             Name
	bucket           *gcsx.SyncerBucket
	sourceGeneration Generation
	attrs            fuseops.InodeAttributes
	target           string
	metadata         map[string]string

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu sync.Mutex

	// GUARDED_BY(mu)
	lc lookupCount
}

var _ Inode = &SymlinkInode{}

// Create a symlink inode for the supplied object record.
//
// REQUIRES: IsSymlink(o)
func NewSymlinkInode(
	ctx context.Context,
	id fuseops.InodeID,
	name Name,
	bucket *gcsx.SyncerBucket,
	m *gcs.MinObject,
	attrs fuseops.InodeAttributes) (s *SymlinkInode) {
	// Resolve the symlink target before returning the inode.
	// Backing MinObject is guranteed exist at this point.
	target, err := resolveSymlinkTarget(ctx, bucket, m)
	if err != nil {
		return nil
	}

	// Create the inode.
	s = &SymlinkInode{
		id:     id,
		name:   name,
		bucket: bucket,
		sourceGeneration: Generation{
			Object:   m.Generation,
			Metadata: m.MetaGeneration,
			Size:     m.Size,
		},
		attrs: fuseops.InodeAttributes{
			Nlink: 1,
			Uid:   attrs.Uid,
			Gid:   attrs.Gid,
			Mode:  attrs.Mode,
			Atime: m.Updated,
			Ctime: m.Updated,
			Mtime: m.Updated,
		},
		target:   target,
		metadata: m.Metadata,
	}

	// Set up lookup counting.
	s.lc.Init(id)

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *SymlinkInode) Lock() {
	s.mu.Lock()
}

func (s *SymlinkInode) Unlock() {
	s.mu.Unlock()
}

func (s *SymlinkInode) ID() fuseops.InodeID {
	return s.id
}

func (s *SymlinkInode) Name() Name {
	return s.name
}

// SourceGeneration returns the object generation from which this inode was branched.
//
// LOCKS_REQUIRED(s)
func (s *SymlinkInode) SourceGeneration() Generation {
	return s.sourceGeneration
}

func (s *SymlinkInode) UpdateSize(size uint64) {
	// The size of a symlink is its target's length, not the backing object's size.
	// However, to keep generation info consistent, we update it.
	s.sourceGeneration.Size = size
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) IncrementLookupCount() {
	s.lc.Inc()
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) DecrementLookupCount(n uint64) (destroy bool) {
	destroy = s.lc.Dec(n)
	return
}

// LOCKS_REQUIRED(s.mu)
func (s *SymlinkInode) Destroy() (err error) {
	// Nothing to do.
	return
}

func (s *SymlinkInode) Attributes(
	ctx context.Context, clobberedCheck bool) (attrs fuseops.InodeAttributes, err error) {
	attrs = s.attrs
	return
}

// Target returns the target of the symlink.
func (s *SymlinkInode) Target() (target string) {
	target = s.target
	return
}

func (s *SymlinkInode) Unlink() {
}

// Bucket returns the bucket that owns this inode.
func (s *SymlinkInode) Bucket() *gcsx.SyncerBucket {
	return s.bucket
}

// Source returns the MinObject from which this inode was created.
func (s *SymlinkInode) Source() *gcs.MinObject {
	return &gcs.MinObject{
		Name:           s.name.GcsObjectName(),
		Generation:     s.sourceGeneration.Object,
		MetaGeneration: s.sourceGeneration.Metadata,
		Size:           s.sourceGeneration.Size,
		Metadata:       s.metadata,
		Updated:        s.attrs.Mtime,
	}
}

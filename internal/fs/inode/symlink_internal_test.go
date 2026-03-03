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
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SymlinkInternalTest struct {
	suite.Suite
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock
}

func TestSymlinkInternalTest(t *testing.T) {
	suite.Run(t, new(SymlinkInternalTest))
}

func (t *SymlinkInternalTest) SetupTest() {
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})
}

func (t *SymlinkInternalTest) createSymlinkInode(name string, target string) *SymlinkInode {
	objName := name
	// Create object in bucket
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(target))
	require.NoError(t.T(), err)

	m := storageutil.ConvertObjToMinObject(o)
	// For standard symlink, we set the metadata key
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[StandardSymlinkMetadataKey] = "true"

	syncerBucket := gcsx.NewSyncerBucket(
		1,
		10, // ChunkTransferTimeoutSecs
		".gcsfuse_tmp/",
		t.bucket,
	)

	return NewSymlinkInode(
		fuseops.InodeID(1),
		NewFileName(NewRootName(""), name),
		&syncerBucket,
		m,
		fuseops.InodeAttributes{},
	)
}

func (t *SymlinkInternalTest) TestOpenReader() {
	target := "target_file"
	s := t.createSymlinkInode("foo", target)

	rc, err := s.openReader(t.ctx)
	require.NoError(t.T(), err)
	defer rc.Close()

	content, err := io.ReadAll(rc)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), target, string(content))
}

func (t *SymlinkInternalTest) TestOpenReader_Clobbered() {
	target := "target_file"
	s := t.createSymlinkInode("foo", target)

	// Clobber the object in GCS (update it, changing generation)
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("new_target"))
	require.NoError(t.T(), err)

	_, err = s.openReader(t.ctx)
	require.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.True(t.T(), errors.As(err, &clobberedErr))
}

func (t *SymlinkInternalTest) TestResolveSymlinkTarget_Standard() {
	target := "target_file"
	s := t.createSymlinkInode("foo", target)

	resolvedTarget, err := s.resolveSymlinkTarget(t.ctx)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), target, resolvedTarget)
}

func (t *SymlinkInternalTest) TestResolveSymlinkTarget_Legacy() {
	target := "target_file"
	objName := "legacy_symlink"
	// Create object in bucket with empty content
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte(""))
	require.NoError(t.T(), err)

	m := storageutil.ConvertObjToMinObject(o)
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[SymlinkMetadataKey] = target

	syncerBucket := gcsx.NewSyncerBucket(
		1,
		10, // ChunkTransferTimeoutSecs
		".gcsfuse_tmp/",
		t.bucket,
	)

	s := NewSymlinkInode(
		fuseops.InodeID(1),
		NewFileName(NewRootName(""), objName),
		&syncerBucket,
		m,
		fuseops.InodeAttributes{},
	)

	resolvedTarget, err := s.resolveSymlinkTarget(t.ctx)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), target, resolvedTarget)
}

func (t *SymlinkInternalTest) TestResolveSymlinkTarget_Clobbered() {
	target := "target_file"
	s := t.createSymlinkInode("foo", target)

	// Clobber the object in GCS (update it, changing generation)
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("new_target"))
	require.NoError(t.T(), err)

	_, err = s.resolveSymlinkTarget(t.ctx)

	require.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.True(t.T(), errors.As(err, &clobberedErr))
}

func (t *SymlinkInternalTest) TestResolveSymlinkTarget_NotSymlink() {
	objName := "not_symlink"
	o, err := storageutil.CreateObject(t.ctx, t.bucket, objName, []byte("content"))
	require.NoError(t.T(), err)

	m := storageutil.ConvertObjToMinObject(o)
	syncerBucket := gcsx.NewSyncerBucket(
		1,
		10, // ChunkTransferTimeoutSecs
		".gcsfuse_tmp/",
		t.bucket,
	)

	s := NewSymlinkInode(
		fuseops.InodeID(1),
		NewFileName(NewRootName(""), objName),
		&syncerBucket,
		m,
		fuseops.InodeAttributes{},
	)

	_, err = s.resolveSymlinkTarget(t.ctx)

	require.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "symlink target could not be resolved")
}

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

package inode_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSymlinkTest(t *testing.T) *gcsx.SyncerBucket {
	bucket := gcsx.NewSyncerBucket(
		/*appendThreshold=*/ 1,
		/*chunkRetryDeadlineSecs=*/ 120,
		/*chunkTransferTimeoutSecs=*/ 10,
		".gcsfuse_tmp/",
		fake.NewFakeBucket(timeutil.RealClock(), "some-bucket", gcs.BucketType{}),
	)
	return &bucket
}

func TestIsSymlinkWhenMetadataKeyIsPresent(t *testing.T) {
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	m := gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}

	assert.True(t, inode.IsSymlink(&m))
}

func TestIsSymlinkWhenMetadataKeyIsNotPresent(t *testing.T) {
	m := gcs.MinObject{
		Name: "test",
	}

	assert.False(t, inode.IsSymlink(&m))
}

func TestIsSymlinkWhenStandardMetadataKeyIsPresent(t *testing.T) {
	metadata := map[string]string{
		inode.StandardSymlinkMetadataKey: "true",
	}
	m := gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}

	assert.True(t, inode.IsSymlink(&m))
}

func TestIsSymlinkWhenStandardMetadataKeyIsFalse(t *testing.T) {
	metadata := map[string]string{
		inode.StandardSymlinkMetadataKey: "false",
	}
	m := gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}

	assert.False(t, inode.IsSymlink(&m))
}

func TestIsSymlinkForNilObject(t *testing.T) {
	assert.False(t, inode.IsSymlink(nil))
}

func TestAttributes(t *testing.T) {
	bucket := setupSymlinkTest(t)
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	m := &gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}
	attrs := fuseops.InodeAttributes{
		Uid:  1001,
		Gid:  1002,
		Mode: 0777 | os.ModeSymlink,
	}
	name := inode.NewFileName(inode.NewRootName("some-bucket"), m.Name)
	s, err := inode.NewSymlinkInode(context.Background(), fuseops.InodeID(42), name, bucket, m, attrs)
	require.NoError(t, err)

	tests := []struct {
		name           string
		clobberedCheck bool
	}{
		{"WithClobberedCheckFalse", false},
		{"WithClobberedCheckTrue", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call Attributes
			extracted, err := s.Attributes(context.TODO(), tt.clobberedCheck)

			// Check expected values
			require.NoError(t, err)
			assert.Equal(t, uint32(1), extracted.Nlink)
			assert.Equal(t, attrs.Uid, extracted.Uid)
			assert.Equal(t, attrs.Gid, extracted.Gid)
			assert.Equal(t, attrs.Mode, extracted.Mode)
		})
	}
}

func TestUpdateSize(t *testing.T) {
	bucket := setupSymlinkTest(t)
	m := &gcs.MinObject{
		Name:           "test",
		Generation:     1,
		MetaGeneration: 2,
		Size:           100,
		Metadata:       map[string]string{inode.SymlinkMetadataKey: "target"},
	}
	attrs := fuseops.InodeAttributes{}
	name := inode.NewFileName(inode.NewRootName("some-bucket"), m.Name)
	s, err := inode.NewSymlinkInode(context.Background(), fuseops.InodeID(42), name, bucket, m, attrs)
	require.NoError(t, err)

	s.UpdateSize(200)

	assert.Equal(t, uint64(200), s.SourceGeneration().Size)
}

func TestSource(t *testing.T) {
	bucket := setupSymlinkTest(t)
	obj, err := storageutil.CreateObject(
		context.Background(),
		bucket,
		"test", // The name of the object in GCS
		[]byte("target_path"),
	)
	require.NoError(t, err)
	m := storageutil.ConvertObjToMinObject(obj)
	m.Metadata = map[string]string{inode.StandardSymlinkMetadataKey: "true"}
	m.Updated = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Explicitly set Updated time for consistent testing.
	attrs := fuseops.InodeAttributes{}
	name := inode.NewFileName(inode.NewRootName("some-bucket"), m.Name)
	s, err := inode.NewSymlinkInode(context.Background(), fuseops.InodeID(42), name, bucket, m, attrs)
	require.NoError(t, err)

	source := s.Source()

	assert.Equal(t, m.Name, source.Name)
	assert.Equal(t, m.Generation, source.Generation)
	assert.Equal(t, m.MetaGeneration, source.MetaGeneration)
	assert.Equal(t, m.Size, source.Size)
	assert.Equal(t, m.Metadata, source.Metadata)
	assert.Equal(t, 0, m.Updated.Compare(source.Updated))
}

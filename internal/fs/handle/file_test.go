// Copyright 2025 Google LLC
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

package handle

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

// createDirInode helps create the parent directory inode for the file inode
// which will be used for testing methods defined on the fileHandle.
func createDirInode(
	bucket *gcsx.SyncerBucket,
	clock *timeutil.SimulatedClock,
	dirName string) inode.DirInode {
	return inode.NewDirInode(
		1,
		inode.NewDirName(inode.NewRootName(""), dirName),
		fuseops.InodeAttributes{
			Uid:  0,
			Gid:  0,
			Mode: 0712,
		},
		false,
		false,
		true,
		0,
		bucket,
		clock,
		clock,
		4,
		false,
	)
}

// createFileInode is a helper to create a FileInode for testing.
func createFileInode(
	t *testing.T,
	bucket *gcsx.SyncerBucket,
	clock *timeutil.SimulatedClock,
	config *cfg.Config,
	parent inode.DirInode,
	objectName string,
	content []byte) *inode.FileInode {

	obj := &gcs.MinObject{
		Name:           objectName,
		Size:           uint64(len(content)),
		Generation:     1,
		MetaGeneration: 1,
		Updated:        clock.Now(),
	}

	// Create object in the fake bucket to simulate existing GCS object
	_, err := bucket.CreateObject(context.Background(), &gcs.CreateObjectRequest{
		Name:     objectName,
		Contents: io.NopCloser(bytes.NewReader(content)),
	})
	if err != nil {
		t.Fatalf("Failed to create object in fake bucket: %v", err)
	}

	return inode.NewFileInode(
		fuseops.InodeID(2),
		inode.NewFileName(parent.Name(), obj.Name),
		obj,
		fuseops.InodeAttributes{},
		bucket,
		false,
		contentcache.New("", clock),
		clock,
		false,
		config,
		semaphore.NewWeighted(100),
	)
}

// TODO: Add unit test for fh.Read()

func TestFileHandleWrite(t *testing.T) {
	var clock timeutil.SimulatedClock
	clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := gcsx.NewSyncerBucket(
		1, 10, ".gcsfuse_tmp/", fake.NewFakeBucket(&clock, "some_bucket", gcs.BucketType{}))
	parent := createDirInode(&bucket, &clock, "parentRoot")
	config := &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: false}}
	in := createFileInode(t, &bucket, &clock, config, parent, "test_obj", nil)
	fh := NewFileHandle(in, nil, false, nil, util.Write, &cfg.ReadConfig{})
	ctx := context.Background()
	data := []byte("hello")

	err := fh.Write(ctx, data, 0)

	assert.Nil(t, err)
	// Validate that write is successful at inode.
	buf := make([]byte, len(data))
	n, err := in.Read(ctx, buf, 0)
	buf = buf[:n]
	// Ignore EOF.
	if err == io.EOF {
		err = nil
	}
	assert.Nil(t, err)
	assert.Equal(t, data, buf)
}

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
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type fileTest struct {
	suite.Suite
	ctx    context.Context
	clock  timeutil.SimulatedClock
	bucket gcsx.SyncerBucket
}

func TestRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(fileTest))
}

func (t *fileTest) SetupTest() {
	t.ctx = context.TODO()
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	t.bucket = gcsx.NewSyncerBucket(
		1, 10, ".gcsfuse_tmp/", fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{}))
}

func (t *fileTest) TearDownTest() {
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *fileTest) TestFileHandleWrite() {
	parent := createDirInode(&t.bucket, &t.clock, "parentRoot")
	config := &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: false}}
	in := createFileInode(t.T(), &t.bucket, &t.clock, config, parent, "test_obj", nil)
	fh := NewFileHandle(in, nil, false, nil, util.Write, &cfg.ReadConfig{})
	data := []byte("hello")

	_, err := fh.Write(t.ctx, data, 0)

	assert.Nil(t.T(), err)
	// Validate that write is successful at inode.
	buf := make([]byte, len(data))
	n, err := in.Read(t.ctx, buf, 0)
	buf = buf[:n]
	// Ignore EOF.
	if err == io.EOF {
		err = nil
	}
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), data, buf)
}

func (t *fileTest) TestFileHandleRead() {
	type testCase struct {
		name              string
		enableReadManager bool
		expectedData      []byte
		expectErr         bool
	}
	testCases := []testCase{
		{
			name:              "use reader",
			enableReadManager: false,
			expectedData:      []byte("hello from reader"),
			expectErr:         false,
		},
		{
			name:              "use readManager",
			enableReadManager: true,
			expectedData:      []byte("hello from readManager"),
			expectErr:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			parent := createDirInode(&t.bucket, &t.clock, "parentRoot")
			objectName := "test_obj_" + tc.name
			in := createFileInode(t.T(), &t.bucket, &t.clock, nil, parent, objectName, tc.expectedData)
			fh := NewFileHandle(in, nil, false, common.NewNoopMetrics(), util.Read, &cfg.ReadConfig{})
			buf := make([]byte, len(tc.expectedData))
			fh.inode.Lock()

			output, n, err := fh.Read(t.ctx, buf, 0, 200, tc.enableReadManager)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), len(tc.expectedData), n)
			assert.Equal(t.T(), tc.expectedData, output)
		})
	}
}

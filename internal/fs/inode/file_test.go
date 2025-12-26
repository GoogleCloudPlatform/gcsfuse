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
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const uid = 123
const gid = 456

const fileInodeID = 17
const fileName = "foo/bar"
const fileMode os.FileMode = 0641
const Delta = 30 * time.Minute
const LocalFile = "Local"
const EmptyGCSFile = "EmptyGCS"

var AppendMode = util.NewOpenMode(util.WriteOnly, util.O_APPEND)
var WriteMode = util.NewOpenMode(util.WriteOnly, 0)

type FileTest struct {
	suite.Suite
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock

	initialContents string
	backingObj      *gcs.MinObject

	in *FileInode
}

func TestFileTestSuite(t *testing.T) {
	suite.Run(t, new(FileTest))
}

func (t *FileTest) SetupSubTest() {
	t.SetupTest()
}

func (t *FileTest) SetupTest() {
	// Enabling invariant check for all tests.
	syncutil.EnableInvariantChecking()
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})

	// Set up the backing object.
	var err error

	t.initialContents = "taco"
	object, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		fileName,
		[]byte(t.initialContents))
	t.backingObj = storageutil.ConvertObjToMinObject(object)

	assert.Nil(t.T(), err)

	// Create the inode.
	t.createInode()
}

func (t *FileTest) TearDownTest() {
	t.in.Unlock()
}

func (t *FileTest) createInodeWithEmptyObject() {
	object, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		fileName,
		[]byte{})
	t.backingObj = storageutil.ConvertObjToMinObject(object)

	assert.Nil(t.T(), err)

	// Create the inode.
	t.createInode()
}

func (t *FileTest) createInode() {
	t.createInodeWithLocalParam(fileName, false)
}
func (t *FileTest) createInodeWithLocalParam(fileName string, local bool) {
	name := NewFileName(
		NewRootName(""),
		fileName,
	)
	syncerBucket := gcsx.NewSyncerBucket(
		1, // Append threshold
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		t.bucket)

	if local {
		t.backingObj = nil
	}
	t.in = NewFileInode(
		fileInodeID,
		name,
		t.backingObj,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: fileMode,
		},
		&syncerBucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		local,
		&cfg.Config{},
		semaphore.NewWeighted(math.MaxInt64),
		nil)

	t.in.Lock()
}

func (t *FileTest) createBufferedWriteHandler(shouldInitialize bool, openMode util.OpenMode) {
	// Initialize BWH for local inode created above.
	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx, openMode)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), shouldInitialize, initialized)
	if shouldInitialize {
		assert.NotNil(t.T(), t.in.bwh)
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileTest) TestID() {
	assert.Equal(t.T(), fileInodeID, int(t.in.ID()))
}

func (t *FileTest) TestName() {
	assert.Equal(t.T(), fileName, t.in.Name().GcsObjectName())
}

func (t *FileTest) TestAreBufferedWritesSupported() {
	finalizedTime := time.Date(2025, time.June, 18, 23, 30, 0, 0, time.UTC)
	unFinalizedTime := time.Time{}
	nonNilContents := "taco"
	testCases := []struct {
		name       string
		content    string
		openMode   util.OpenMode
		bucketType gcs.BucketType
		finalized  time.Time
		supported  bool
	}{
		{
			name:       "AppendToFinalizedObjOnZB",
			content:    nonNilContents,
			bucketType: gcs.BucketType{Zonal: true},
			finalized:  finalizedTime,
			openMode:   AppendMode,
			supported:  false,
		},
		{
			name:       "AppendToUnfinalizedObjOnZB",
			content:    nonNilContents,
			bucketType: gcs.BucketType{Zonal: true},
			finalized:  unFinalizedTime,
			openMode:   AppendMode,
			supported:  true,
		},
		{
			name:       "AppendToObjOnNonZB",
			content:    nonNilContents,
			bucketType: gcs.BucketType{},
			finalized:  finalizedTime,
			openMode:   AppendMode,
			supported:  false,
		},
		{
			name:       "WriteToObjOnNonZB",
			content:    nonNilContents,
			bucketType: gcs.BucketType{},
			finalized:  finalizedTime,
			openMode:   WriteMode,
			supported:  false,
		},
		{
			name:       "WriteToEmptyObj",
			content:    "",
			bucketType: gcs.BucketType{},
			finalized:  finalizedTime,
			openMode:   WriteMode,
			supported:  true,
		},
	}
	for _, tc := range testCases {
		t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", tc.bucketType)
		// Set up the backing object.
		var err error
		t.initialContents = tc.content
		object, err := storageutil.CreateObject(
			t.ctx,
			t.bucket,
			fileName,
			[]byte(t.initialContents))
		assert.Nil(t.T(), err)
		object.Finalized = tc.finalized
		t.backingObj = storageutil.ConvertObjToMinObject(object)
		t.createInode()
		t.in.config.Write.EnableRapidAppends = true

		isSupported := t.in.areBufferedWritesSupported(tc.openMode, object)

		assert.Equal(t.T(), tc.supported, isSupported)
	}
}

func (t *FileTest) TestInitialSourceGeneration() {
	sg := t.in.SourceGeneration()
	assert.Equal(t.T(), t.backingObj.Generation, sg.Object)
	assert.Equal(t.T(), t.backingObj.MetaGeneration, sg.Metadata)
	assert.Equal(t.T(), t.backingObj.Size, sg.Size)
}

func (t *FileTest) TestSourceGenerationSizeAfterWriteDoesNotChange() {
	gcsSynced, err := t.in.Write(context.Background(), []byte(setup.GenerateRandomString(5)), 0, WriteMode)
	require.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)

	sg := t.in.SourceGeneration()

	assert.Equal(t.T(), t.backingObj.Generation, sg.Object)
	assert.Equal(t.T(), t.backingObj.MetaGeneration, sg.Metadata)
	assert.Equal(t.T(), t.backingObj.Size, sg.Size)
}

func (t *FileTest) TestSourceGenerationIsAuthoritativeReturnsTrue() {
	assert.True(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileTest) TestSourceGenerationIsAuthoritativeReturnsFalseAfterWrite() {
	gcsSynced, err := t.in.Write(t.ctx, []byte("taco"), 0, WriteMode)
	assert.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)

	assert.False(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileTest) TestSyncPendingBufferedWritesReturnsNilAndNoOpForNonStreamingWrites() {
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.initialContents, string(contents))
	gcsSynced, err := t.in.Write(t.ctx, []byte("bar"), 0, WriteMode)
	assert.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)

	gcsSynced, err = t.in.SyncPendingBufferedWrites()

	require.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	contents, err = storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.initialContents, string(contents))
}

func (t *FileTest) TestAttributes_Clobbered_WithClobberCheckTrue() {
	// Simulate a clobbered file by creating a new object with the same name,
	// which will have a new generation.
	_, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("new clobbering content"))
	require.NoError(t.T(), err)

	attrs, err := t.in.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	// Since clobberCheck is true and the generation has changed,
	// Nlink should be 0.
	assert.Equal(t.T(), uint32(0), attrs.Nlink)
}

func (t *FileTest) TestAttributes_Clobbered_WithClobberCheckFalse() {
	// Simulate a clobbered file by creating a new object with the same name,
	// which will have a new generation.
	_, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("new clobbering content"))
	require.NoError(t.T(), err)

	attrs, err := t.in.Attributes(t.ctx, false)

	require.NoError(t.T(), err)
	// Since clobberCheck is false, Nlink should be 1.
	assert.Equal(t.T(), uint32(1), attrs.Nlink)
}

func (t *FileTest) TestInitialAttributes() {
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), uint64(len(t.initialContents)), attrs.Size)
	assert.Equal(t.T(), uint32(1), attrs.Nlink)
	assert.Equal(t.T(), uint32(uid), attrs.Uid)
	assert.Equal(t.T(), uint32(gid), attrs.Gid)
	assert.Equal(t.T(), fileMode, attrs.Mode)
	assert.Equal(t.T(), attrs.Atime, t.backingObj.Updated)
	assert.Equal(t.T(), attrs.Ctime, t.backingObj.Updated)
	assert.Equal(t.T(), attrs.Mtime, t.backingObj.Updated)
}

func (t *FileTest) TestInitialAttributes_MtimeFromObjectMetadata_Gcsfuse() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	mtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0)
	t.backingObj.Metadata["gcsfuse_mtime"] = mtime.Format(time.RFC3339Nano)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), attrs.Mtime, mtime)
}

func (t *FileTest) TestInitialAttributes_MtimeFromObjectMetadata_Gsutil() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	mtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0).Round(time.Second)
	t.backingObj.Metadata["goog-reserved-file-mtime"] = strconv.FormatInt(mtime.Unix(), 10)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), attrs.Mtime.UTC(), mtime)
}

func (t *FileTest) TestInitialAttributes_MtimeFromObjectMetadata_GcsfuseOutranksGsutil() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	gsutilMtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0).Round(time.Second)
	t.backingObj.Metadata["goog-reserved-file-mtime"] = strconv.FormatInt(gsutilMtime.Unix(), 10)

	canonicalMtime := time.Now().Add(456*time.Second).UTC().AddDate(0, 0, 0)
	t.backingObj.Metadata["gcsfuse_mtime"] = canonicalMtime.Format(time.RFC3339Nano)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), attrs.Mtime, canonicalMtime)
}

func (t *FileTest) TestRead() {
	assert.Equal(t.T(), "taco", t.initialContents)

	// Make several reads, checking the expected contents.
	testCases := []struct {
		offset   int64
		size     int
		expected string
	}{
		{0, 1, "t"},
		{0, 2, "ta"},
		{0, 3, "tac"},
		{0, 4, "taco"},
		{0, 5, "taco"},

		{1, 1, "a"},
		{1, 2, "ac"},
		{1, 3, "aco"},
		{1, 4, "aco"},

		{3, 1, "o"},
		{3, 2, "o"},

		// Empty ranges
		{0, 0, ""},
		{3, 0, ""},
		{4, 0, ""},
		{4, 1, ""},
		{5, 0, ""},
		{5, 1, ""},
		{5, 2, ""},
	}

	for _, tc := range testCases {
		desc := fmt.Sprintf("offset: %d, size: %d", tc.offset, tc.size)
		data := make([]byte, tc.size)
		n, err := t.in.Read(t.ctx, data, tc.offset)
		data = data[:n]

		// Ignore EOF.
		if err == io.EOF {
			err = nil
		}

		assert.Nil(t.T(), err, "%s", desc)
		assert.Equal(t.T(), tc.expected, string(data), "%s", desc)
	}
}

func (t *FileTest) TestWrite() {
	var err error

	assert.Equal(t.T(), "taco", t.initialContents)

	// Overwite a byte.
	gcsSynced, err := t.in.Write(t.ctx, []byte("p"), 0, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)

	// Add some data at the end.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()

	gcsSynced, err = t.in.Write(t.ctx, []byte("burrito"), 4, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)

	t.clock.AdvanceTime(time.Second)

	// Read back the content.
	var buf [1024]byte
	n, err := t.in.Read(t.ctx, buf[:], 0)

	if err == io.EOF {
		err = nil
	}

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "pacoburrito", string(buf[:n]))

	// Check attributes.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), uint64(len("pacoburrito")), attrs.Size)
	assert.Equal(t.T(), attrs.Mtime, writeTime)
}

func (t *FileTest) TestTruncate() {
	var attrs fuseops.InodeAttributes
	var err error

	assert.Equal(t.T(), "taco", t.initialContents)

	// Truncate downward.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	gcsSynced, err := t.in.Truncate(t.ctx, 2)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)

	t.clock.AdvanceTime(time.Second)

	// Read the contents.
	var buf [1024]byte
	n, err := t.in.Read(t.ctx, buf[:], 0)

	if err == io.EOF {
		err = nil
	}

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "ta", string(buf[:n]))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), uint64(len("ta")), attrs.Size)
	assert.Equal(t.T(), attrs.Mtime, truncateTime)
}

func (t *FileTest) TestTruncateNegative() {
	assert.Equal(t.T(), "taco", t.initialContents)

	// Truncate neagtive.
	gcsSynced, err := t.in.Truncate(t.ctx, -1)

	require.Error(t.T(), err)
	assert.False(t.T(), gcsSynced)
}

func (t *FileTest) TestWriteThenSync() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error

			assert.Equal(t.T(), "taco", t.initialContents)

			// Overwrite a byte.
			t.clock.AdvanceTime(time.Second)
			writeTime := t.clock.Now()

			gcsSynced, err := t.in.Write(t.ctx, []byte("p"), 0, WriteMode)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)

			t.clock.AdvanceTime(time.Second)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// The generation should have advanced.
			assert.Less(t.T(), t.backingObj.Generation, t.in.SourceGeneration().Object)

			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)

			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(len("paco")), m.Size)
			assert.Equal(t.T(),
				writeTime.UTC().Format(time.RFC3339Nano),
				m.Metadata["gcsfuse_mtime"])

			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())

			require.NoError(t.T(), err)
			assert.Equal(t.T(), "paco", string(contents))

			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)

			assert.Equal(t.T(), uint64(len("paco")), attrs.Size)
			assert.Equal(t.T(), attrs.Mtime, writeTime.UTC())
		})
	}
}

func (t *FileTest) TestWriteToLocalFileThenSync() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error
			// Create a local file inode.
			t.createInodeWithLocalParam("test", true)
			// Create a temp file for the local inode created above.
			err = t.in.CreateEmptyTempFile(t.ctx)
			assert.Nil(t.T(), err)
			// Write some content to temp file.
			t.clock.AdvanceTime(time.Second)
			writeTime := t.clock.Now()
			gcsSynced, err := t.in.Write(t.ctx, []byte("tacos"), 0, WriteMode)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)
			t.clock.AdvanceTime(time.Second)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// Verify that fileInode is no more local
			assert.False(t.T(), t.in.IsLocal())
			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)
			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(len("tacos")), m.Size)
			assert.Equal(t.T(),
				writeTime.UTC().Format(time.RFC3339Nano),
				m.Metadata["gcsfuse_mtime"])
			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is same as the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			require.NoError(t.T(), err)
			assert.Equal(t.T(), "tacos", string(contents))
			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(len("tacos")), attrs.Size)
			assert.Equal(t.T(), attrs.Mtime, writeTime.UTC())
		})
	}
}

func (t *FileTest) TestSyncEmptyLocalFile() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error
			// Create a local file inode.
			t.createInodeWithLocalParam("test", true)
			creationTime := t.clock.Now()
			// Create a temp file for the local inode created above.
			err = t.in.CreateEmptyTempFile(t.ctx)
			assert.Nil(t.T(), err)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// Verify that fileInode is no more local
			assert.False(t.T(), t.in.IsLocal())
			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)
			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(0), m.Size)
			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate the mtime.
			mtimeInBucket, ok := m.Metadata["gcsfuse_mtime"]
			assert.True(t.T(), ok)
			mtime, _ := time.Parse(time.RFC3339Nano, mtimeInBucket)
			assert.WithinDuration(t.T(), mtime, creationTime, Delta)
			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			require.NoError(t.T(), err)
			assert.Equal(t.T(), "", string(contents))
			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)
		})
	}
}

func (t *FileTest) TestAppendThenSync() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error

			assert.Equal(t.T(), "taco", t.initialContents)

			// Append some data.
			t.clock.AdvanceTime(time.Second)
			writeTime := t.clock.Now()

			gcsSynced, err := t.in.Write(t.ctx, []byte("burrito"), int64(len("taco")), AppendMode)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)

			t.clock.AdvanceTime(time.Second)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// The generation should have advanced.
			assert.Less(t.T(), t.backingObj.Generation, t.in.SourceGeneration().Object)

			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)

			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(len("tacoburrito")), m.Size)
			assert.Equal(t.T(),
				writeTime.UTC().Format(time.RFC3339Nano),
				m.Metadata["gcsfuse_mtime"])
			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())

			require.NoError(t.T(), err)
			assert.Equal(t.T(), "tacoburrito", string(contents))

			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)

			assert.Equal(t.T(), uint64(len("tacoburrito")), attrs.Size)
			assert.Equal(t.T(), attrs.Mtime, writeTime.UTC())
		})
	}
}

func (t *FileTest) TestAppendToUnfinalizedObjInZB() {
	// Set up the Zonal Bucket
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{Zonal: true})
	// Set up the backing unfinalized object.
	var err error
	t.initialContents = "lychee"
	object, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		fileName,
		[]byte(t.initialContents))
	assert.Nil(t.T(), err)
	object.Finalized = time.Time{}
	t.backingObj = storageutil.ConvertObjToMinObject(object)
	t.createInode()
	t.in.config = &cfg.Config{Write: *getWriteConfigWithEnabledRapidAppends()}
	assert.Nil(t.T(), t.in.content)
	t.createBufferedWriteHandler(true, AppendMode)
	assert.NotNil(t.T(), t.in.bwh)

	gcsSynced, err := t.in.Write(t.ctx, []byte("juice"), int64(len(t.initialContents)), AppendMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)

	gcsSynced, err = t.in.Sync(t.ctx)
	require.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)

	// Read the object contents through back-door.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "lycheejuice", string(contents))
}

func (t *FileTest) TestTruncateDownwardThenSync() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error

			// Truncate downward.
			t.clock.AdvanceTime(time.Second)
			truncateTime := t.clock.Now()

			gcsSynced, err := t.in.Truncate(t.ctx, 2)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)

			t.clock.AdvanceTime(time.Second)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// The generation should have advanced.
			assert.Less(t.T(), t.backingObj.Generation, t.in.SourceGeneration().Object)

			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)

			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(2), m.Size)
			assert.Equal(t.T(),
				truncateTime.UTC().Format(time.RFC3339Nano),
				m.Metadata["gcsfuse_mtime"])

			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)

			assert.Equal(t.T(), uint64(2), attrs.Size)
			assert.Equal(t.T(), attrs.Mtime, truncateTime.UTC())
		})
	}
}

func (t *FileTest) TestTruncateUpwardThenFlush() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var attrs fuseops.InodeAttributes
			var err error

			assert.Equal(t.T(), 4, len(t.initialContents))

			// Truncate upward.
			t.clock.AdvanceTime(time.Second)
			truncateTime := t.clock.Now()

			gcsSynced, err := t.in.Truncate(t.ctx, 6)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)

			t.clock.AdvanceTime(time.Second)

			if tc.callSync {
				gcsSynced, err := t.in.Sync(t.ctx)
				require.NoError(t.T(), err)
				assert.True(t.T(), gcsSynced)
			} else {
				err = t.in.Flush(t.ctx)
				assert.Nil(t.T(), err)
			}

			// The generation should have advanced.
			assert.Less(t.T(), t.backingObj.Generation, t.in.SourceGeneration().Object)

			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

			// Stat the current object in the bucket.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)

			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(),
				truncateTime.UTC().Format(time.RFC3339Nano),
				m.Metadata["gcsfuse_mtime"])
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), t.in.SourceGeneration().Size, m.Size)
			assert.Equal(t.T(), uint64(6), m.Size)

			// Check attributes.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)

			assert.Equal(t.T(), uint64(6), attrs.Size)
			assert.Equal(t.T(), attrs.Mtime, truncateTime.UTC())
		})
	}
}

func (t *FileTest) TestTruncateUpwardForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)
	// Fetch the attributes and check if the file is empty.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(0), attrs.Size)

	gcsSynced, err := t.in.Truncate(t.ctx, 6)

	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	// The inode should return the new size.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(6), attrs.Size)
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	require.Error(t.T(), err)
	assert.Equal(t.T(), "gcs.NotFoundError: object test not found", err.Error())
}

func (t *FileTest) TestTruncateDownwardForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)
	// Write some data to the local file.
	gcsSynced, err := t.in.Write(t.ctx, []byte("burrito"), 0, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	// Validate the new data is written correctly.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(7), attrs.Size)

	gcsSynced, err = t.in.Truncate(t.ctx, 2)

	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	// The inode should return the new size.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(2), attrs.Size)
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	require.Error(t.T(), err)
	assert.Equal(t.T(), "gcs.NotFoundError: object test not found", err.Error())
}

func (t *FileTest) TestTruncateUpwardForLocalFileWhenStreamingWritesAreEnabled() {
	tbl := []struct {
		name         string
		performWrite bool
	}{
		{
			name:         "WithWrite",
			performWrite: true,
		},
		{
			name:         "WithOutWrite",
			performWrite: false,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			// Create a local file inode.
			t.createInodeWithLocalParam("test", true)
			t.in.config = &cfg.Config{Write: *getWriteConfig()}
			// Fetch the attributes and check if the file is empty.
			attrs, err := t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)

			if tc.performWrite {
				t.createBufferedWriteHandler(true, WriteMode)
				gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)
				assert.Nil(t.T(), err)
				assert.False(t.T(), gcsSynced)
				assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)
				// Fetch the attributes and check if the file size reflects the write.
				attrs, err := t.in.Attributes(t.ctx, true)
				require.NoError(t.T(), err)
				assert.Equal(t.T(), uint64(2), attrs.Size)
			}
			t.createBufferedWriteHandler(!tc.performWrite, WriteMode)

			gcsSynced, err := t.in.Truncate(t.ctx, 10)

			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)
			// The inode should return the new size.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(10), attrs.Size)
			// Data shouldn't be updated to GCS.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			_, _, err = t.bucket.StatObject(t.ctx, statReq)
			require.Error(t.T(), err)
			assert.Equal(t.T(), "gcs.NotFoundError: object test not found", err.Error())
		})
	}
}

func (t *FileTest) TestTruncateUpwardForEmptyGCSFileWhenStreamingWritesAreEnabled() {
	tbl := []struct {
		name         string
		performWrite bool
	}{
		{
			name:         "WithWrite",
			performWrite: true,
		},
		{
			name:         "WithOutWrite",
			performWrite: false,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			t.createInodeWithEmptyObject()
			t.in.config = &cfg.Config{Write: *getWriteConfig()}

			// Fetch the attributes and check if the file is empty.
			attrs, err := t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)

			if tc.performWrite {
				t.createBufferedWriteHandler(true, WriteMode)
				gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)
				assert.Nil(t.T(), err)
				assert.False(t.T(), gcsSynced)
				assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)
				// Fetch the attributes and check if the file size reflects the write.
				attrs, err := t.in.Attributes(t.ctx, true)
				require.NoError(t.T(), err)
				assert.Equal(t.T(), uint64(2), attrs.Size)
			}
			t.createBufferedWriteHandler(!tc.performWrite, WriteMode)

			gcsSynced, err := t.in.Truncate(t.ctx, 10)

			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)
			// The inode should return the new size.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(10), attrs.Size)
			// Data shouldn't be updated to GCS.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			minObject, _, err := t.bucket.StatObject(t.ctx, statReq)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), minObject.Size)
		})
	}
}

func (t *FileTest) TestTruncateDownwardWhenStreamingWritesAreEnabled() {
	tbl := []struct {
		name         string
		fileType     string
		truncateSize int64
	}{
		{
			name:         "LocalFileTruncateToNonZero",
			fileType:     LocalFile,
			truncateSize: 2,
		},
		{
			name:         "LocalFileTruncateToZero",
			fileType:     LocalFile,
			truncateSize: 0,
		},
		{
			name:         "EmptyGCSFileTruncateToNonZero",
			fileType:     EmptyGCSFile,
			truncateSize: 2,
		},
		{
			name:         "EmptyGCSFileTruncateToZero",
			fileType:     EmptyGCSFile,
			truncateSize: 0,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			if tc.fileType == LocalFile {
				t.createInodeWithLocalParam("test", true)
			}
			if tc.fileType == EmptyGCSFile {
				t.createInodeWithEmptyObject()
			}
			t.in.config = &cfg.Config{Write: *getWriteConfig()}
			// Fetch the attributes and check if the file is empty.
			attrs, err := t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)

			t.createBufferedWriteHandler(true, WriteMode)
			gcsSynced, err := t.in.Write(t.ctx, []byte("hihello"), 0, WriteMode)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)
			assert.Equal(t.T(), int64(7), t.in.bwh.WriteFileInfo().TotalSize)
			// Fetch the attributes and check if the file size reflects the write.
			attrs, err = t.in.Attributes(t.ctx, true)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(7), attrs.Size)
			gcsSynced, err = t.in.Truncate(t.ctx, tc.truncateSize)

			require.NoError(t.T(), err)
			assert.True(t.T(), gcsSynced)
			t.createBufferedWriteHandler(false, WriteMode)
		})
	}
}

func (t *FileTest) TestSyncFlush_Clobbered() {
	testcases := []struct {
		name     string
		callSync bool
	}{
		{
			name:     "sync",
			callSync: true,
		},
		{
			name:     "flush",
			callSync: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func() {
			var err error

			// Truncate downward.
			gcsSynced, err := t.in.Truncate(t.ctx, 2)
			assert.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)

			// Clobber the backing object.
			newObj, err := storageutil.CreateObject(
				t.ctx,
				t.bucket,
				t.in.Name().GcsObjectName(),
				[]byte("burrito"))

			assert.Nil(t.T(), err)

			if tc.callSync {
				var gcsSynced bool
				// Sync. The call should not succeed, and we expect a FileClobberedError.
				gcsSynced, err = t.in.Sync(t.ctx)
				require.Error(t.T(), err)
				assert.False(t.T(), gcsSynced)
			} else {
				// Flush. The call should not succeed, and we expect a FileClobberedError.
				err = t.in.Flush(t.ctx)
			}

			// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
			assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
			// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
			assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

			// Check if the error is a FileClobberedError
			var fcErr *gcsfuse_errors.FileClobberedError
			assert.True(t.T(), errors.As(err, &fcErr), "expected FileClobberedError but got %v", err)
			assert.Equal(t.T(), t.backingObj.Generation, t.in.SourceGeneration().Object)
			assert.Equal(t.T(), t.backingObj.MetaGeneration, t.in.SourceGeneration().Metadata)
			assert.Equal(t.T(), t.backingObj.Size, t.in.SourceGeneration().Size)

			// The object in the bucket should not have been changed.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)

			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), newObj.Generation, m.Generation)
			assert.Equal(t.T(), newObj.Size, m.Size)
		})
	}
}

func (t *FileTest) TestOpenReader_ThrowsFileClobberedError() {
	// Modify the file locally.
	gcsSynced, err := t.in.Truncate(t.ctx, 2)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	// Clobber the backing object.
	_, err = storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("burrito"))
	assert.Nil(t.T(), err)

	_, err = t.in.openReader(t.ctx)

	// assert error is not nil.
	var fcErr *gcsfuse_errors.FileClobberedError
	assert.True(t.T(), errors.As(err, &fcErr), "expected FileClobberedError but got %v", err)
}

func (t *FileTest) TestSetMtime_ContentNotFaultedIn() {
	var err error
	var attrs fuseops.InodeAttributes

	// Set mtime.
	mtime := time.Now().UTC().Add(123*time.Second).AddDate(0, 0, 0)

	err = t.in.SetMtime(t.ctx, mtime)
	assert.Nil(t.T(), err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)

	// The inode should have added the mtime to the backing object's metadata.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(),
		mtime.UTC().Format(time.RFC3339Nano),
		m.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) TestSetMtime_ContentClean() {
	var err error
	var attrs fuseops.InodeAttributes

	// Cause the content to be faulted in.
	_, err = t.in.Read(t.ctx, make([]byte, 1), 0)
	assert.Nil(t.T(), err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123*time.Second).AddDate(0, 0, 0)

	err = t.in.SetMtime(t.ctx, mtime)
	assert.Nil(t.T(), err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)

	// The inode should have added the mtime to the backing object's metadata.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(),
		mtime.UTC().Format(time.RFC3339Nano),
		m.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) TestSetMtime_ContentDirty() {
	var err error
	var attrs fuseops.InodeAttributes

	// Dirty the content.
	gcsSynced, err := t.in.Write(t.ctx, []byte("a"), 0, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)

	err = t.in.SetMtime(t.ctx, mtime)
	assert.Nil(t.T(), err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)

	// Sync.
	gcsSynced, err = t.in.Sync(t.ctx)
	require.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)

	// Now the object in the bucket should have the appropriate mtime.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)

	// Validate MinObject in inode and MRDWrapper points to different copy of MinObject.
	assert.NotSame(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())
	// Validate MinObject in MRDWrapper is equal to the MinObject in inode.
	assert.Equal(t.T(), &t.in.src, t.in.MRDWrapper.GetMinObject())

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(),
		mtime.UTC().Format(time.RFC3339Nano),
		m.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) TestSetMtime_SourceObjectGenerationChanged() {
	var err error

	// Clobber the backing object.
	newObj, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("burrito"))

	assert.Nil(t.T(), err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)
	assert.Nil(t.T(), err)

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(), newObj.Generation, m.Generation)
	assert.Equal(t.T(), 0, len(m.Metadata))
}

func (t *FileTest) TestSetMtime_SourceObjectMetaGenerationChanged() {
	var err error

	// Update the backing object.
	lang := "fr"
	newObj, err := t.bucket.UpdateObject(
		t.ctx,
		&gcs.UpdateObjectRequest{
			Name:            t.in.Name().GcsObjectName(),
			ContentLanguage: &lang,
		})

	assert.Nil(t.T(), err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)
	assert.Nil(t.T(), err)

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(), newObj.Generation, m.Generation)
	assert.Equal(t.T(), newObj.MetaGeneration, m.MetaGeneration)
}

func (t *FileTest) TestSetMtimeForUnlinkedFileIsNoOp() {
	t.in.unlinked = true
	beforeUpdateAttr, err := t.in.Attributes(t.ctx, true)
	require.Nil(t.T(), err)
	mtime := beforeUpdateAttr.Mtime.UTC().Add(123 * time.Second)

	// Set mtime.
	err = t.in.SetMtime(t.ctx, mtime)

	require.Nil(t.T(), err)
	afterUpdateAttr, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.NotEqual(t.T(), mtime, afterUpdateAttr.Mtime)
	assert.Equal(t.T(), beforeUpdateAttr.Mtime, afterUpdateAttr.Mtime)
}

func (t *FileTest) TestTestSetMtimeForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes

	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	createTime := t.in.mtimeClock.Now()
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)
	// Validate the attributes on an empty file.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.WithinDuration(t.T(), attrs.Mtime, createTime, Delta)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)

	assert.Nil(t.T(), err)
	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)
	assert.Equal(t.T(), attrs.Ctime, mtime)
	assert.Equal(t.T(), attrs.Atime, mtime)
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	require.Error(t.T(), err)
	assert.Equal(t.T(), "gcs.NotFoundError: object test not found", err.Error())
}

func (t *FileTest) TestSetMtimeForLocalFileWhenStreamingWritesAreEnabled() {
	var err error
	var attrs fuseops.InodeAttributes

	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	t.createBufferedWriteHandler(true, WriteMode)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)

	assert.Nil(t.T(), err)
	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)
	assert.Equal(t.T(), attrs.Ctime, mtime)
	assert.Equal(t.T(), attrs.Atime, mtime)
}

func (t *FileTest) TestContentEncodingGzip() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := "gzip"
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	assert.Equal(t.T(), contentEncoding, t.in.Source().ContentEncoding)
	assert.True(t.T(), t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) TestContentEncodingNone() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := ""
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	assert.Equal(t.T(), contentEncoding, t.in.Source().ContentEncoding)
	assert.False(t.T(), t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) TestContentEncodingOther() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := "other"
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	assert.Equal(t.T(), contentEncoding, t.in.Source().ContentEncoding)
	assert.False(t.T(), t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) TestTestCheckInvariantsShouldNotThrowExceptionForLocalFiles() {
	t.createInodeWithLocalParam("test", true)

	assert.NotNil(t.T(), t.in)
}

func (t *FileTest) TestCreateEmptyTempFile() {
	err := t.in.CreateEmptyTempFile(t.ctx)

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.content)
	// Validate that file size is 0.
	sr, err := t.in.content.Stat()
	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(0), sr.Size)
}

func (t *FileTest) TestCreateEmptyTempFileWhenBWHIsNotNil() {
	testCases := []struct {
		name    string
		isLocal bool
	}{
		{
			name:    "ShouldNotCreateForEmptyGCSFile",
			isLocal: false,
		},
		{
			name:    "ShouldNotCreateForLocalFile",
			isLocal: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			if tc.isLocal {
				t.createInodeWithLocalParam("test", true)
			} else {
				t.createInodeWithEmptyObject()
			}
			t.in.config = &cfg.Config{Write: *getWriteConfig()}
			t.createBufferedWriteHandler(true, WriteMode)

			err := t.in.CreateEmptyTempFile(t.ctx)

			assert.Nil(t.T(), err)
			assert.Nil(t.T(), t.in.content)
		})
	}
}

func (t *FileTest) TestInitBufferedWriteHandlerIfEligibleShouldNotCreateBWHNonEmptySyncedFile() {
	// Enabling buffered writes.
	t.in.config = &cfg.Config{Write: *getWriteConfig()}

	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx, WriteMode)

	assert.NoError(t.T(), err)
	assert.Nil(t.T(), t.in.bwh)
	assert.False(t.T(), initialized)
}

func (t *FileTest) TestUnlinkLocalFile() {
	var err error
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)

	// Unlink.
	t.in.Unlink()

	// Verify that fileInode is now unlinked
	assert.True(t.T(), t.in.IsUnlinked())
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, _, err = t.bucket.StatObject(t.ctx, statReq)
	require.Error(t.T(), err)
	assert.Equal(t.T(), "gcs.NotFoundError: object test not found", err.Error())
}

func (t *FileTest) TestReadFileWhenStreamingWritesAreEnabled() {
	tbl := []struct {
		name         string
		fileType     string
		performWrite bool
	}{
		{
			name:         "LocalFileWithWrite",
			fileType:     LocalFile,
			performWrite: true,
		},
		{
			name:         "LocalFileWithOutWrite",
			fileType:     LocalFile,
			performWrite: false,
		},
		{
			name:         "EmptyGCSFileWithWrite",
			fileType:     EmptyGCSFile,
			performWrite: true,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			if tc.fileType == LocalFile {
				// Create a local file inode.
				t.createInodeWithLocalParam("test", true)
				t.in.config = &cfg.Config{Write: *getWriteConfig()}
				t.createBufferedWriteHandler(true, WriteMode)
			}

			if tc.fileType == EmptyGCSFile {
				t.createInodeWithEmptyObject()
				t.in.config = &cfg.Config{Write: *getWriteConfig()}
			}

			if tc.performWrite {
				t.createBufferedWriteHandler(tc.fileType != LocalFile, WriteMode)
				gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)
				assert.Nil(t.T(), err)
				assert.False(t.T(), gcsSynced)
				assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)
			}
			data := make([]byte, len("hi"))
			// Flush is required before reading an object for which BWH is open.
			assert.NoError(t.T(), t.in.Flush(context.Background()))

			n, err := t.in.Read(t.ctx, data, 0)

			if tc.performWrite {
				assert.Equal(t.T(), len(data), n)
				require.NoError(t.T(), err)
			} else {
				assert.Equal(t.T(), 0, n)
				require.Error(t.T(), err)
				assert.ErrorIs(t.T(), err, io.EOF)
			}
		})
	}
}

func (t *FileTest) TestReadEmptyGCSFileWhenStreamingWritesAreNotInProgress() {
	t.createInodeWithEmptyObject()
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	data := make([]byte, 10)

	n, err := t.in.Read(t.ctx, data, 0)

	assert.Equal(t.T(), 0, n)
	require.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "EOF")
}

func (t *FileTest) TestInitBufferedWriteHandlerWithInvalidConfigWhenStreamingWritesAreEnabled() {
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	t.in.config = &cfg.Config{Write: cfg.WriteConfig{EnableStreamingWrites: true}}

	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx, WriteMode)

	assert.True(t.T(), strings.Contains(err.Error(), "invalid configuration"))
	assert.False(t.T(), initialized)
	assert.Nil(t.T(), t.in.bwh)
}

func (t *FileTest) TestWriteToLocalFileWhenStreamingWritesAreEnabled() {
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	t.createBufferedWriteHandler(true, WriteMode)

	gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)

	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.NotNil(t.T(), t.in.bwh)
	writeFileInfo := t.in.bwh.WriteFileInfo()
	assert.Equal(t.T(), int64(2), writeFileInfo.TotalSize)
}

func (t *FileTest) TestMultipleWritesToLocalFileWhenStreamingWritesAreEnabled() {
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	createTime := t.in.mtimeClock.Now()
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	t.createBufferedWriteHandler(true, WriteMode)

	gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.NotNil(t.T(), t.in.bwh)
	assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)

	gcsSynced, err = t.in.Write(t.ctx, []byte("hello"), 2, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.Equal(t.T(), int64(7), t.in.bwh.WriteFileInfo().TotalSize)
	// The inode should agree about the new mtime.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(7), attrs.Size)
	assert.WithinDuration(t.T(), attrs.Mtime, createTime, Delta)
}

func (t *FileTest) TestWriteToEmptyGCSFileWhenStreamingWritesAreEnabled() {
	t.createInodeWithEmptyObject()
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	createTime := t.in.mtimeClock.Now()
	t.createBufferedWriteHandler(true, WriteMode)

	gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)

	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.NotNil(t.T(), t.in.bwh)
	writeFileInfo := t.in.bwh.WriteFileInfo()
	assert.Equal(t.T(), int64(2), writeFileInfo.TotalSize)
	// The inode should agree about the new mtime.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(2), attrs.Size)
	assert.WithinDuration(t.T(), attrs.Mtime, createTime, Delta)
}

func (t *FileTest) TestSetMtimeOnEmptyGCSFileWhenStreamingWritesAreEnabled() {
	t.createInodeWithEmptyObject()
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	assert.Nil(t.T(), t.in.bwh)

	// This test checks if the mtime is updated to GCS. Since test framework
	// doesn't support t.run, calling the test method here directly.
	t.TestSetMtime_ContentNotFaultedIn()
	// bufferedWritesHandler shouldn't get initialized.
	assert.Nil(t.T(), t.in.bwh)
}

func (t *FileTest) TestSetMtimeOnEmptyGCSFileAfterWritesWhenStreamingWritesAreEnabled() {
	t.createInodeWithEmptyObject()
	t.in.config = &cfg.Config{Write: *getWriteConfig()}
	t.createBufferedWriteHandler(true, WriteMode)
	// Initiate write call.
	gcsSynced, err := t.in.Write(t.ctx, []byte("hi"), 0, WriteMode)
	assert.Nil(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.NotNil(t.T(), t.in.bwh)
	writeFileInfo := t.in.bwh.WriteFileInfo()
	assert.Equal(t.T(), int64(2), writeFileInfo.TotalSize)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)

	assert.Nil(t.T(), err)
	// The inode should agree about the new mtime.
	attrs, err := t.in.Attributes(t.ctx, true)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), attrs.Mtime, mtime)
	assert.Equal(t.T(), attrs.Ctime, mtime)
	assert.Equal(t.T(), attrs.Atime, mtime)
}

func (t *FileTest) TestRegisterFileHandle() {
	tbl := []struct {
		name        string
		readonly    bool
		currentVal  int32
		expectedVal int32
	}{
		{
			name:        "ReadOnlyHandle",
			readonly:    true,
			currentVal:  0,
			expectedVal: 0,
		},
		{
			name:        "ZeroCurrentValueForWriteHandle",
			readonly:    false,
			currentVal:  0,
			expectedVal: 1,
		},
		{
			name:        "NonZeroCurrentValueForWriteHandle",
			readonly:    false,
			currentVal:  5,
			expectedVal: 6,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			t.in.writeHandleCount = tc.currentVal

			t.in.RegisterFileHandle(tc.readonly)

			assert.Equal(t.T(), tc.expectedVal, t.in.writeHandleCount)
		})
	}
}

func getWriteConfig() *cfg.WriteConfig {
	return &cfg.WriteConfig{
		MaxBlocksPerFile:      10,
		BlockSizeMb:           1,
		EnableStreamingWrites: true,
	}
}

func getWriteConfigWithEnabledRapidAppends() *cfg.WriteConfig {
	return &cfg.WriteConfig{
		MaxBlocksPerFile:      10,
		BlockSizeMb:           1,
		EnableStreamingWrites: true,
		EnableRapidAppends:    true,
	}
}

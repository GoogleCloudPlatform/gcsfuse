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

package gcsx

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate for FullObjectCreatorTests
////////////////////////////////////////////////////////////////////////

type fullObjectCreatorHelper struct {
	t           *testing.T
	assert      *assert.Assertions
	require     *require.Assertions
	ctx         context.Context
	bucket      *storage.TestifyMockBucket
	creator     objectCreator
	srcObject   gcs.Object
	srcContents string
	mtime       time.Time
}

func newFullObjectCreatorHelper(t *testing.T) *fullObjectCreatorHelper {
	h := &fullObjectCreatorHelper{
		t:       t,
		assert:  assert.New(t),
		require: require.New(t),
		ctx:     context.Background(),
	}
	h.bucket = new(storage.TestifyMockBucket)
	h.creator = &fullObjectCreator{
		bucket: h.bucket,
	}
	return h
}

func (h *fullObjectCreatorHelper) call() (*gcs.Object, error) {
	return h.creator.Create(
		h.ctx,
		h.srcObject.Name,
		&h.srcObject,
		&h.mtime,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		strings.NewReader(h.srcContents))
}

func (h *fullObjectCreatorHelper) validateEmptyProperties(req *gcs.CreateObjectRequest) {
	h.require.NotNil(req)
	if h.assert.NotNil(req.GenerationPrecondition) {
		h.assert.Equal(int64(0), *req.GenerationPrecondition)
	}
	// All the properties should be empty/nil.
	h.assert.Nil(req.MetaGenerationPrecondition)
	h.assert.Equal("", req.CacheControl)
	h.assert.Equal("", req.ContentDisposition)
	h.assert.Equal("", req.ContentEncoding)
	h.assert.Equal("", req.ContentType)
	h.assert.Equal("", req.CustomTime)
	h.assert.Equal(false, req.EventBasedHold)
	h.assert.Equal("", req.StorageClass)
	// Validate the object contents.
	b, err := io.ReadAll(req.Contents)
	h.require.NoError(err)
	h.assert.Equal(h.srcContents, string(b))
}

////////////////////////////////////////////////////////////////////////
// Tests for FullObjectCreator
////////////////////////////////////////////////////////////////////////

func TestFullObjectCreator_CallsCreateObject(t *testing.T) {
	h := newFullObjectCreatorHelper(t)
	h.srcContents = "taco"

	// CreateObject
	var req *gcs.CreateObjectRequest
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.CreateObjectRequest)
		})

	// Call
	_, _ = h.call()

	h.require.NotNil(req)
	if h.assert.NotNil(req.GenerationPrecondition) {
		h.assert.Equal(int64(0), *req.GenerationPrecondition)
	}

	b, err := io.ReadAll(req.Contents)
	h.require.NoError(err)
	h.assert.Equal(h.srcContents, string(b))
}

func TestFullObjectCreator_CreateObjectFails(t *testing.T) {
	h := newFullObjectCreatorHelper(t)

	// CreateObject
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("taco"))

	// Call
	_, err := h.call()

	h.assert.ErrorContains(err, "CreateObject")
	h.assert.ErrorContains(err, "taco")
}

func TestFullObjectCreator_CreateObjectReturnsPreconditionError(t *testing.T) {
	h := newFullObjectCreatorHelper(t)

	// CreateObject
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), &gcs.PreconditionError{Err: errors.New("taco")})

	// Call
	_, err := h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
	h.assert.ErrorContains(err, "CreateObject")
	h.assert.ErrorContains(err, "taco")
}

func TestFullObjectCreator_CallsCreateObjectsWithObjectProperties(t *testing.T) {
	h := newFullObjectCreatorHelper(t)
	h.srcObject.Name = "foo"
	h.srcObject.Generation = 17
	h.srcObject.MetaGeneration = 23
	h.srcObject.CacheControl = "testCacheControl"
	h.srcObject.ContentDisposition = "inline"
	h.srcObject.ContentEncoding = "gzip"
	h.srcObject.ContentType = "text/plain"
	h.srcObject.CustomTime = "2022-04-02T00:30:00Z"
	h.srcObject.EventBasedHold = true
	h.srcObject.StorageClass = "STANDARD"
	h.srcObject.Metadata = map[string]string{
		"test_key": "test_value",
	}
	h.mtime = time.Now().Add(123 * time.Second).UTC()

	var req *gcs.CreateObjectRequest
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.CreateObjectRequest)
		})

	// Call
	_, _ = h.call()

	h.require.NotNil(req)
	h.assert.Equal(h.srcObject.Name, req.Name)
	h.assert.Equal(h.srcObject.CacheControl, req.CacheControl)
	h.assert.Equal(h.srcObject.ContentDisposition, req.ContentDisposition)
	h.assert.Equal(h.srcObject.ContentEncoding, req.ContentEncoding)
	h.assert.Equal(h.srcObject.ContentType, req.ContentType)
	h.assert.Equal(h.srcObject.CustomTime, req.CustomTime)
	h.assert.Equal(h.srcObject.EventBasedHold, req.EventBasedHold)

	h.assert.Equal(2, len(req.Metadata))
	h.assert.Equal(h.mtime.Format(time.RFC3339Nano), req.Metadata["gcsfuse_mtime"])
	h.assert.Equal("test_value", req.Metadata["test_key"])
}

func TestFullObjectCreator_CallsCreateObjectWhenSrcObjectIsNil(t *testing.T) {
	h := newFullObjectCreatorHelper(t)
	h.srcContents = "taco"
	// CreateObject
	var req *gcs.CreateObjectRequest
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.CreateObjectRequest)
		})

	// Call
	_, _ = h.creator.Create(
		h.ctx,
		h.srcObject.Name,
		nil,
		&h.mtime,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		strings.NewReader(h.srcContents))

	h.validateEmptyProperties(req)
	h.assert.Equal(h.mtime.Format(time.RFC3339Nano), req.Metadata["gcsfuse_mtime"])
}

func TestFullObjectCreator_CallsCreateObjectWhenSrcObjectAndMtimeAreNil(t *testing.T) {
	h := newFullObjectCreatorHelper(t)
	h.srcContents = "taco"
	// CreateObject
	var req *gcs.CreateObjectRequest
	h.bucket.On("CreateObject", mock.Anything, mock.Anything).
		Return((*gcs.Object)(nil), errors.New("")).
		Run(func(args mock.Arguments) {
			req = args.Get(1).(*gcs.CreateObjectRequest)
		})

	// Call
	_, _ = h.creator.Create(
		h.ctx,
		h.srcObject.Name,
		nil,
		nil,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		strings.NewReader(h.srcContents))

	h.validateEmptyProperties(req)
	_, ok := req.Metadata["gcsfuse_mtime"]
	h.assert.False(ok)
}

////////////////////////////////////////////////////////////////////////
// fakeObjectCreator
////////////////////////////////////////////////////////////////////////

// An objectCreator that records the arguments it is called with, returning
// canned results.
type fakeObjectCreator struct {
	t      *testing.T
	assert *assert.Assertions
	called bool

	// Supplied arguments
	srcObject *gcs.Object
	mtime     time.Time
	contents  []byte

	// Canned results
	o   *gcs.Object
	err error
}

func (oc *fakeObjectCreator) Create(
	ctx context.Context,
	fileName string,
	srcObject *gcs.Object,
	mtime *time.Time,
	chunkRetryDeadlineSecs int64,
	chunkTransferTimeoutSecs int64,
	r io.Reader) (o *gcs.Object, err error) {
	// Have we been called more than once?
	oc.assert.False(oc.called)
	oc.called = true

	// Record args.
	oc.srcObject = srcObject
	if mtime != nil {
		oc.mtime = *mtime
	}
	oc.contents, err = io.ReadAll(r)
	oc.assert.NoError(err)

	// Return results.
	o, err = oc.o, oc.err
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate for SyncerTests
////////////////////////////////////////////////////////////////////////

const srcObjectContents = "taco"
const appendThreshold = int64(len(srcObjectContents))
const chunkRetryDeadlineSecs = 120
const chunkTransferTimeoutSecs = 10

type syncerHelper struct {
	t             *testing.T
	assert        *assert.Assertions
	require       *require.Assertions
	ctx           context.Context
	fullCreator   fakeObjectCreator
	appendCreator fakeObjectCreator
	bucket        gcs.Bucket
	syncer        Syncer
	clock         timeutil.SimulatedClock
	srcObject     *gcs.Object
	content       TempFile
}

func newSyncerHelper(t *testing.T) *syncerHelper {
	h := &syncerHelper{
		t:       t,
		assert:  assert.New(t),
		require: require.New(t),
		ctx:     context.Background(),
		fullCreator: fakeObjectCreator{
			t:      t,
			assert: assert.New(t),
		},
		appendCreator: fakeObjectCreator{
			t:      t,
			assert: assert.New(t),
		},
	}

	// Set up dependencies.
	h.bucket = fake.NewFakeBucket(&h.clock, "some_bucket", gcs.BucketType{})
	h.syncer = newSyncer(
		appendThreshold,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		&h.fullCreator,
		&h.appendCreator)

	h.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up a source object.
	var err error
	h.srcObject, err = h.bucket.CreateObject(
		h.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader(srcObjectContents),
		})
	h.require.NoError(err)
	h.srcObject.Finalized = time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)

	// Wrap a TempFile around it.
	h.content, err = NewTempFile(
		dummyReadCloser{strings.NewReader(srcObjectContents)},
		t.TempDir(),
		&h.clock)
	h.require.NoError(err)

	// Return errors from the fakes by default.
	h.fullCreator.err = errors.New("Fake error")
	h.appendCreator.err = errors.New("Fake error")

	return h
}

func (h *syncerHelper) tearDown() {
	if h.content != nil {
		h.content.Destroy()
	}
}

func (h *syncerHelper) call() (*gcs.Object, error) {
	return h.syncer.SyncObject(h.ctx, h.srcObject.Name, h.srcObject, h.content)
}

type dummyReadCloser struct {
	io.Reader
}

func (rc dummyReadCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////
// Tests for Syncer
////////////////////////////////////////////////////////////////////////

func TestSyncer_SyncObjectShouldInvokeFullObjectCreatorWhenSrcObjectIsNil(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	// It doesn't make sense to validate returned object or error since fake
	// is not handling them.
	_, _ = h.syncer.SyncObject(h.ctx, h.srcObject.Name, nil, h.content)

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_UnfinalizedObjectBypassesDirtyThresholdCheck(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.fullCreator.o = &gcs.Object{}
	h.fullCreator.err = nil
	// Simulate an unfinalized object (Finalized time is zero).
	h.srcObject.Finalized = time.Time{}
	h.srcObject.Size = 0
	// Set up the content to have a dirty threshold of 5.
	// We populate NewTempFile with 10 bytes of data so tf.dirtyThreshold is initialized to 10.
	h.content, err = NewTempFile(
		dummyReadCloser{strings.NewReader("1234567890")},
		t.TempDir(),
		&h.clock)
	h.require.NoError(err)
	// Write at offset 5 to make dirtyThreshold = min(10, 5) = 5.
	_, err = h.content.(io.WriterAt).WriteAt([]byte("hi"), 5)
	h.require.NoError(err)

	o, err := h.call()

	// It should bypass the weird dirty threshold error and succeed.
	h.require.NoError(err)
	h.assert.Equal(h.fullCreator.o, o)
	h.assert.True(h.fullCreator.called)
}

func TestSyncer_UnfinalizedObjectDoesNotReturnEarlyOnTruncateToZero(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.fullCreator.o = &gcs.Object{}
	h.fullCreator.err = nil
	// Simulate an unfinalized object (Finalized time is zero).
	h.srcObject.Finalized = time.Time{}
	h.srcObject.Size = 0
	// Set up the content with 10 bytes of initial data.
	h.content, err = NewTempFile(
		dummyReadCloser{strings.NewReader("1234567890")},
		t.TempDir(),
		&h.clock)
	h.require.NoError(err)
	// Truncate to 0. This sets size to 0 and dirtyThreshold to 0.
	err = h.content.Truncate(0)
	h.require.NoError(err)

	// Even though sr.Size == 0 (srcSize) and sr.DirtyThreshold == 0 (srcSize),
	// since the object is unfinalized, it should NOT return early but call fullCreator.
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Equal(h.fullCreator.o, o)
	h.assert.True(h.fullCreator.called)
}

func TestSyncer_UnfinalizedObjectReturnsEarlyIfUnmodifiedAndNonZeroSize(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	// Simulate an unfinalized object (Finalized time is zero).
	h.srcObject.Finalized = time.Time{}
	// But it has a correct metadata size (e.g. 4 bytes, matching local content).
	h.srcObject.Size = uint64(len(srcObjectContents))

	// Since the local temp file is unmodified (sr.Mtime is nil), it should return
	// early even though the object is unfinalized.
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Nil(o)
	// Neither creator should be called.
	h.assert.False(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_UnfinalizedObjectDoesNotReturnEarlyOnTruncateToStaleMetadataSize(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.appendCreator.o = &gcs.Object{}
	h.appendCreator.err = nil
	// Simulate an unfinalized object (Finalized time is zero).
	h.srcObject.Finalized = time.Time{}
	// GCS metadata reports stale size 4 (but actual size was 10).
	h.srcObject.Size = 4
	// Set up the content with 10 bytes of initial data.
	h.content, err = NewTempFile(
		dummyReadCloser{strings.NewReader("1234567890")},
		t.TempDir(),
		&h.clock)
	h.require.NoError(err)
	// Truncate to 4. This sets size to 4 and dirtyThreshold to 4.
	// Since we truncated, tf.mtime becomes non-nil.
	err = h.content.Truncate(4)
	h.require.NoError(err)

	// Even though local size matches stale GCS size (4 == 4) and dirtyThreshold (4 == 4)
	// matches GCS size, since tf.mtime is non-nil and the object is unfinalized,
	// it should bypass early return and call appendCreator.
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Equal(h.appendCreator.o, o)
	h.assert.True(h.appendCreator.called)
	h.assert.False(h.fullCreator.called)
}

func TestSyncer_NotDirty(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	// Call
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Nil(o)

	// Neither creater should have been called.
	h.assert.False(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_SmallerThanSource(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	// Truncate downward.
	err := h.content.Truncate(int64(len(srcObjectContents) - 1))
	h.require.NoError(err)

	// The full creator should be called.
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_SameSizeAsSource(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	// Dirty a byte without changing the length.
	_, err := h.content.(io.WriterAt).WriteAt(
		[]byte("a"),
		int64(len(srcObjectContents)-1))

	h.require.NoError(err)

	// The full creator should be called.
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_LargerThanSource_ThresholdInSource(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error

	// Extend the length of the content.
	err = h.content.Truncate(int64(len(srcObjectContents) + 100))
	h.require.NoError(err)

	// But dirty a byte within the initial content.
	_, err = h.content.(io.WriterAt).WriteAt(
		[]byte("a"),
		int64(len(srcObjectContents)-1))

	h.require.NoError(err)

	// The full creator should be called.
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_SourceTooShortForAppend(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error

	// Recreate the syncer with a higher append threshold.
	h.syncer = newSyncer(
		int64(len(srcObjectContents)+1),
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		&h.fullCreator,
		&h.appendCreator)

	// Extend the length of the content.
	err = h.content.Truncate(int64(len(srcObjectContents) + 1))
	h.require.NoError(err)

	// The full creator should be called.
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_SourceComponentCountTooHigh(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error

	// Simulate a large component count.
	h.srcObject.ComponentCount = gcs.MaxComponentCount

	// Extend the length of the content.
	err = h.content.Truncate(int64(len(srcObjectContents) + 1))
	h.require.NoError(err)

	// The full creator should be called.
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.False(h.appendCreator.called)
}

func TestSyncer_LargerThanSource_ThresholdAtEndOfSource(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error

	// Extend the length of the content.
	err = h.content.Truncate(int64(len(srcObjectContents) + 1))
	h.require.NoError(err)

	// The append creator should be called.
	_, _ = h.call()

	h.assert.False(h.fullCreator.called)
	h.assert.True(h.appendCreator.called)
}

func TestSyncer_CallsFullCreator(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.require.Less(2, int(h.srcObject.Size))

	// Ready the content.
	err = h.content.Truncate(2)
	h.require.NoError(err)

	mtime := time.Now().Add(123 * time.Second)
	h.content.SetMtime(mtime)

	// Call
	_, _ = h.call()

	h.assert.True(h.fullCreator.called)
	h.assert.Equal(h.srcObject, h.fullCreator.srcObject)
	h.assert.True(mtime.Equal(h.fullCreator.mtime))
	h.assert.Equal(srcObjectContents[:2], string(h.fullCreator.contents))
}

func TestSyncer_FullCreatorFails(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.fullCreator.err = errors.New("taco")

	// Truncate downward.
	err = h.content.Truncate(2)
	h.require.NoError(err)

	// Call
	_, err = h.call()

	h.assert.ErrorContains(err, "create")
	h.assert.ErrorContains(err, "taco")
}

func TestSyncer_FullCreatorReturnsPreconditionError(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.fullCreator.err = &gcs.PreconditionError{}

	// Truncate downward.
	err = h.content.Truncate(2)
	h.require.NoError(err)

	// Call
	_, err = h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
}

func TestSyncer_FullCreatorSucceeds(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.fullCreator.o = &gcs.Object{}
	h.fullCreator.err = nil

	// Truncate downward.
	err = h.content.Truncate(2)
	h.require.NoError(err)

	// Call
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Equal(h.fullCreator.o, o)
}

func TestSyncer_CallsAppendCreator(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error

	// Append some data.
	_, err = h.content.(io.WriterAt).WriteAt([]byte("burrito"), int64(h.srcObject.Size))
	h.require.NoError(err)

	// Set up an expected mtime.
	mtime := time.Now().Add(123 * time.Second)
	h.content.SetMtime(mtime)

	// Call
	_, _ = h.call()

	h.assert.True(h.appendCreator.called)
	h.assert.Equal(h.srcObject, h.appendCreator.srcObject)
	h.assert.True(mtime.Equal(h.appendCreator.mtime))
	h.assert.Equal("burrito", string(h.appendCreator.contents))
}

func TestSyncer_AppendCreatorFails(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.appendCreator.err = errors.New("taco")

	// Append some data.
	_, err = h.content.(io.WriterAt).WriteAt([]byte("burrito"), int64(h.srcObject.Size))
	h.require.NoError(err)

	// Call
	_, err = h.call()

	h.assert.ErrorContains(err, "create")
	h.assert.ErrorContains(err, "taco")
}

func TestSyncer_AppendCreatorReturnsPreconditionError(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.appendCreator.err = &gcs.PreconditionError{}

	// Append some data.
	_, err = h.content.(io.WriterAt).WriteAt([]byte("burrito"), int64(h.srcObject.Size))
	h.require.NoError(err)

	// Call
	_, err = h.call()

	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)
}

func TestSyncer_AppendCreatorSucceeds(t *testing.T) {
	h := newSyncerHelper(t)
	defer h.tearDown()

	var err error
	h.appendCreator.o = &gcs.Object{}
	h.appendCreator.err = nil

	// Append some data.
	_, err = h.content.(io.WriterAt).WriteAt([]byte("burrito"), int64(h.srcObject.Size))
	h.require.NoError(err)

	// Call
	o, err := h.call()

	h.require.NoError(err)
	h.assert.Equal(h.appendCreator.o, o)
}

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

package gcsx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create random content of the given length, which must be a multiple of 4.
func randBytes(n int) (b []byte) {
	if n%4 != 0 {
		panic(fmt.Sprintf("Invalid n: %d", n))
	}

	b = make([]byte, n)
	for i := 0; i < n; i += 4 {
		w := rand.Uint32()
		b[i] = byte(w >> 24)
		b[i+1] = byte(w >> 16)
		b[i+2] = byte(w >> 8)
		b[i+3] = byte(w >> 0)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type integrationTestHelper struct {
	t       *testing.T
	assert  *assert.Assertions
	require *require.Assertions
	ctx     context.Context
	bucket  gcs.Bucket
	clock   timeutil.SimulatedClock
	syncer  gcsx.Syncer
	tf      gcsx.TempFile
}

func newIntegrationTestHelper(t *testing.T) *integrationTestHelper {
	h := &integrationTestHelper{
		t:       t,
		assert:  assert.New(t),
		require: require.New(t),
		ctx:     context.Background(),
	}
	h.bucket = fake.NewFakeBucket(&h.clock, "some_bucket", gcs.BucketType{})

	// Set up a fixed, non-zero time.
	h.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// Set up the syncer.
	const appendThreshold = 0
	const chunkRetryDeadlineSecs = 120
	const chunkTransferTimeoutSecs = 10
	const tmpObjectPrefix = ".gcsfuse_tmp/"

	h.syncer = gcsx.NewSyncer(
		appendThreshold,
		chunkRetryDeadlineSecs,
		chunkTransferTimeoutSecs,
		tmpObjectPrefix,
		h.bucket)

	return h
}

func (h *integrationTestHelper) tearDown() {
	if h.tf != nil {
		h.tf.Destroy()
	}
}

func (h *integrationTestHelper) create(o *gcs.Object) {
	if h.tf != nil {
		h.tf.Destroy()
	}
	if o.Finalized.IsZero() {
		o.Finalized = h.clock.Now()
	}

	// Set up a reader.
	rc, err := h.bucket.NewReaderWithReadHandle(
		h.ctx,
		&gcs.ReadObjectRequest{
			Name:       o.Name,
			Generation: o.Generation,
		})
	h.require.NoError(err)

	// Use it to create the temp file.
	h.tf, err = gcsx.NewTempFile(rc, h.t.TempDir(), &h.clock)
	h.require.NoError(err)

	// Close it.
	err = rc.Close()
	h.require.NoError(err)
}

// Helper to write to h.tf avoiding linter issues with embedded interfaces.
func (h *integrationTestHelper) writeAt(p []byte, off int64) (int, error) {
	return h.tf.(io.WriterAt).WriteAt(p, off)
}

// Return the object generation, or -1 if non-existent. Panic on error.
func (h *integrationTestHelper) objectGeneration(name string) (gen int64) {
	// Stat.
	req := &gcs.StatObjectRequest{Name: name}
	m, _, err := h.bucket.StatObject(h.ctx, req)

	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		gen = -1
		return
	}

	h.require.NoError(err)
	gen = m.Generation
	return
}

func (h *integrationTestHelper) sync(src *gcs.Object) (*gcs.Object, error) {
	o, err := h.syncer.SyncObject(h.ctx, src.Name, src, h.tf)
	if err == nil && o != nil {
		h.tf = nil
	}
	return o, err
}

func (h *integrationTestHelper) verifyBucketContents(expectedNames []string) {
	objects, runs, err := storageutil.ListAll(h.ctx, h.bucket, &gcs.ListObjectsRequest{})
	h.require.NoError(err)
	h.assert.Equal(0, len(runs))

	var names []string
	for _, o := range objects {
		names = append(names, o.Name)
	}
	h.assert.ElementsMatch(expectedNames, names)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestIntegration_ReadThenSync(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Read the contents.
	buf := make([]byte, 1024)
	n, err := h.tf.ReadAt(buf, 0)

	h.require.True(err == nil || errors.Is(err, io.EOF))
	h.assert.Equal(len("taco"), n)
	h.assert.Equal("taco", string(buf[:n]))

	// Sync doesn't need to do anything.
	newObj, err := h.sync(o)

	h.require.NoError(err)
	h.assert.Nil(newObj)
}

func TestIntegration_SyncEmptyLocalFile(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create a temp file and write some contents to it.
	tf, err := gcsx.NewTempFile(io.NopCloser(strings.NewReader("")), h.t.TempDir(), &h.clock)
	h.require.NoError(err)
	defer tf.Destroy()

	// Sync should update the object in GCS.
	newObj, err := h.syncer.SyncObject(h.ctx, "test", nil, tf)

	h.require.NoError(err)
	h.assert.Equal(h.objectGeneration("test"), newObj.Generation)
	_, ok := newObj.Metadata["gcsfuse_mtime"]
	h.assert.False(ok)

	// Read via the bucket.
	contents, err := storageutil.ReadObject(h.ctx, h.bucket, "test")
	h.require.NoError(err)
	h.assert.Equal("", string(contents))

	// Verify bucket contents.
	h.verifyBucketContents([]string{"test"})
}

func TestIntegration_SyncNonEmptyLocalFile(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create a temp file and write some contents to it.
	tf, err := gcsx.NewTempFile(io.NopCloser(strings.NewReader("")), h.t.TempDir(), &h.clock)
	h.require.NoError(err)
	defer tf.Destroy()
	h.clock.AdvanceTime(time.Second)
	writeTime := h.clock.Now()
	n, err := tf.(io.WriterAt).WriteAt([]byte("tacobell"), 0)
	h.require.NoError(err)
	h.assert.Equal(8, n)
	h.clock.AdvanceTime(time.Second)

	// Sync should update the object in GCS.
	newObj, err := h.syncer.SyncObject(h.ctx, "test", nil, tf)

	h.require.NoError(err)
	h.assert.Equal(h.objectGeneration("test"), newObj.Generation)
	h.assert.Equal(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	// Read via the bucket.
	contents, err := storageutil.ReadObject(h.ctx, h.bucket, "test")
	h.require.NoError(err)
	h.assert.Equal("tacobell", string(contents))

	// Verify bucket contents.
	h.verifyBucketContents([]string{"test"})
}

func TestIntegration_WriteThenSync(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Overwrite.
	h.clock.AdvanceTime(time.Second)
	writeTime := h.clock.Now()
	n, err := h.writeAt([]byte("burrito"), 0)
	h.clock.AdvanceTime(time.Second)

	h.require.NoError(err)
	h.assert.Equal(len("burrito"), n)

	// Sync should save out the new generation.
	newObj, err := h.sync(o)
	h.require.NoError(err)

	h.assert.NotEqual(o.Generation, newObj.Generation)
	h.assert.Equal(h.objectGeneration("foo"), newObj.Generation)
	h.assert.Equal(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	// Read via the bucket.
	contents, err := storageutil.ReadObject(h.ctx, h.bucket, "foo")
	h.require.NoError(err)
	h.assert.Equal("burrito", string(contents))

	// Verify bucket contents.
	h.verifyBucketContents([]string{"foo"})
}

func TestIntegration_AppendThenSync(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Append some data.
	h.clock.AdvanceTime(time.Second)
	writeTime := h.clock.Now()
	n, err := h.writeAt([]byte("burrito"), 4)
	h.clock.AdvanceTime(time.Second)

	h.require.NoError(err)
	h.assert.Equal(len("burrito"), n)

	// Sync should save out the new generation.
	newObj, err := h.sync(o)
	h.require.NoError(err)

	h.assert.NotEqual(o.Generation, newObj.Generation)
	h.assert.Equal(h.objectGeneration("foo"), newObj.Generation)
	h.assert.Equal(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	// Read via the bucket.
	contents, err := storageutil.ReadObject(h.ctx, h.bucket, "foo")
	h.require.NoError(err)
	h.assert.Equal("tacoburrito", string(contents))

	// Verify bucket contents.
	h.verifyBucketContents([]string{"foo"})
}

func TestIntegration_TruncateThenSync(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Truncate.
	h.clock.AdvanceTime(time.Second)
	truncateTime := h.clock.Now()
	err = h.tf.Truncate(2)
	h.clock.AdvanceTime(time.Second)

	h.require.NoError(err)

	// Sync should save out the new generation.
	newObj, err := h.sync(o)
	h.require.NoError(err)

	h.assert.NotEqual(o.Generation, newObj.Generation)
	h.assert.Equal(h.objectGeneration("foo"), newObj.Generation)
	h.assert.Equal(
		truncateTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	contents, err := storageutil.ReadObject(h.ctx, h.bucket, "foo")
	h.require.NoError(err)
	h.assert.Equal("ta", string(contents))
}

func TestIntegration_Stat_InitialState(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Stat.
	sr, err := h.tf.Stat()
	h.require.NoError(err)

	h.assert.Equal(int64(o.Size), sr.Size)
	h.assert.Equal(int64(o.Size), sr.DirtyThreshold)
	h.assert.Nil(sr.Mtime)
}

func TestIntegration_Stat_Dirty(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Dirty.
	h.clock.AdvanceTime(time.Second)
	truncateTime := h.clock.Now()

	err = h.tf.Truncate(2)
	h.require.NoError(err)

	h.clock.AdvanceTime(time.Second)

	// Stat.
	sr, err := h.tf.Stat()
	h.require.NoError(err)

	h.assert.Equal(int64(2), sr.Size)
	h.assert.Equal(int64(2), sr.DirtyThreshold)
	if h.assert.NotNil(sr.Mtime) {
		h.assert.True(sr.Mtime.Equal(truncateTime))
	}
}

func TestIntegration_BackingObjectHasBeenDeleted(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Fault in the contents.
	_, err = h.tf.ReadAt([]byte{}, 0)
	h.require.NoError(err)

	// Delete the backing object.
	err = h.bucket.DeleteObject(h.ctx, &gcs.DeleteObjectRequest{Name: o.Name})
	h.require.NoError(err)

	// Reading and modications should still work.
	_, err = h.tf.ReadAt([]byte{}, 0)
	h.require.NoError(err)

	_, err = h.writeAt([]byte("a"), 0)
	h.require.NoError(err)

	truncateTime := h.clock.Now()
	err = h.tf.Truncate(1)
	h.require.NoError(err)
	h.clock.AdvanceTime(time.Second)

	// Stat should see the current state.
	sr, err := h.tf.Stat()
	h.require.NoError(err)

	h.assert.Equal(int64(1), sr.Size)
	h.assert.Equal(int64(0), sr.DirtyThreshold)
	if h.assert.NotNil(sr.Mtime) {
		h.assert.True(sr.Mtime.Equal(truncateTime))
	}

	// Sync should fail with a precondition error.
	_, err = h.sync(o)
	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)

	// Nothing should have been created.
	_, err = storageutil.ReadObject(h.ctx, h.bucket, o.Name)
	var notFoundErr *gcs.NotFoundError
	h.assert.ErrorAs(err, &notFoundErr)
}

func TestIntegration_BackingObjectHasBeenOverwritten(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// Create.
	o, err := storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("taco"))
	h.require.NoError(err)

	h.create(o)

	// Fault in the contents.
	_, err = h.tf.ReadAt([]byte{}, 0)
	h.require.NoError(err)

	// Overwrite the backing object.
	_, err = storageutil.CreateObject(h.ctx, h.bucket, "foo", []byte("burrito"))
	h.require.NoError(err)

	// Reading and modications should still work.
	_, err = h.tf.ReadAt([]byte{}, 0)
	h.require.NoError(err)

	_, err = h.writeAt([]byte("a"), 0)
	h.require.NoError(err)

	truncateTime := h.clock.Now()
	err = h.tf.Truncate(3)
	h.require.NoError(err)
	h.clock.AdvanceTime(time.Second)

	// Stat should see the current state.
	sr, err := h.tf.Stat()
	h.require.NoError(err)

	h.assert.Equal(int64(3), sr.Size)
	h.assert.Equal(int64(0), sr.DirtyThreshold)
	if h.assert.NotNil(sr.Mtime) {
		h.assert.True(sr.Mtime.Equal(truncateTime))
	}

	// Sync should fail with a precondition error.
	_, err = h.sync(o)
	var preconditionErr *gcs.PreconditionError
	h.assert.ErrorAs(err, &preconditionErr)

	// The newer version should still be present.
	contents, err := storageutil.ReadObject(h.ctx, h.bucket, o.Name)
	h.require.NoError(err)
	h.assert.Equal("burrito", string(contents))
}

func TestIntegration_MultipleInteractions(t *testing.T) {
	h := newIntegrationTestHelper(t)
	defer h.tearDown()

	// We will run through the script below for multiple interesting object
	// sizes.
	sizes := []int{
		0,
		1,
		1 << 19,
		1 << 20,
		1 << 21,
	}

	// Generate random contents for the maximum size.
	var maxSize int
	for _, size := range sizes {
		if size > maxSize {
			maxSize = size
		}
	}

	randData := randBytes(maxSize)

	// Transition the mutable object in and out of the dirty state. Make sure
	// everything stays consistent.
	for i, size := range sizes {
		desc := fmt.Sprintf("test case %d (size %d)", i, size)
		name := fmt.Sprintf("obj_%d", i)
		buf := make([]byte, size)

		// Create the backing object with random initial contents.
		expectedContents := make([]byte, size)
		copy(expectedContents, randData)

		o, err := storageutil.CreateObject(
			h.ctx,
			h.bucket,
			name,
			expectedContents)
		h.require.NoError(err, desc)

		// Create a temp file around it.
		h.create(o)

		// Read the contents of the temp file.
		_, err = h.tf.ReadAt(buf, 0)
		h.require.True(err == nil || errors.Is(err, io.EOF), desc)
		h.assert.Equal(expectedContents, buf, desc)

		// Modify some bytes.
		if size > 0 {
			expectedContents[0] = 17
			expectedContents[size/2] = 19
			expectedContents[size-1] = 23

			_, err = h.writeAt([]byte{17}, 0)
			h.require.NoError(err, desc)

			_, err = h.writeAt([]byte{19}, int64(size/2))
			h.require.NoError(err, desc)

			_, err = h.writeAt([]byte{23}, int64(size-1))
			h.require.NoError(err, desc)
		}

		// Compare contents again.
		_, err = h.tf.ReadAt(buf, 0)
		h.require.True(err == nil || errors.Is(err, io.EOF), desc)
		h.assert.Equal(expectedContents, buf, desc)

		// Sync and recreate if necessary.
		newObj, err := h.sync(o)
		h.require.NoError(err, desc)

		if newObj != nil {
			h.create(newObj)
		}

		// Check the new backing object's contents.
		objContents, err := storageutil.ReadObject(h.ctx, h.bucket, name)
		h.require.NoError(err, desc)
		h.assert.Equal(expectedContents, objContents, desc)

		// Compare contents again.
		_, err = h.tf.ReadAt(buf, 0)
		h.require.True(err == nil || errors.Is(err, io.EOF), desc)
		h.assert.Equal(expectedContents, buf, desc)

		// Dirty again.
		if size > 0 {
			expectedContents[0] = 29

			_, err = h.writeAt([]byte{29}, 0)
			h.require.NoError(err, desc)
		}

		// Compare contents again.
		_, err = h.tf.ReadAt(buf, 0)
		h.require.True(err == nil || errors.Is(err, io.EOF), desc)
		h.assert.Equal(expectedContents, buf, desc)
	}
}

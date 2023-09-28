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

package gcsx_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

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

type IntegrationTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock
	syncer gcsx.Syncer

	tf gcsx.TempFile
}

var _ SetUpInterface = &IntegrationTest{}
var _ TearDownInterface = &IntegrationTest{}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket")

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	// Set up the syncer.
	const appendThreshold = 0
	const tmpObjectPrefix = ".gcsfuse_tmp/"

	t.syncer = gcsx.NewSyncer(
		appendThreshold,
		tmpObjectPrefix,
		t.bucket)
}

func (t *IntegrationTest) TearDown() {
	if t.tf != nil {
		t.tf.Destroy()
	}
}

func (t *IntegrationTest) create(o *gcs.Object) {
	// Set up a reader.
	rc, err := t.bucket.NewReader(
		t.ctx,
		&gcs.ReadObjectRequest{
			Name:       o.Name,
			Generation: o.Generation,
		})

	AssertEq(nil, err)

	// Use it to create the temp file.
	t.tf, err = gcsx.NewTempFile(rc, "", &t.clock)
	AssertEq(nil, err)

	// Close it.
	err = rc.Close()
	AssertEq(nil, err)
}

// Return the object generation, or -1 if non-existent. Panic on error.
func (t *IntegrationTest) objectGeneration(name string) (gen int64) {
	// Stat.
	req := &gcs.StatObjectRequest{Name: name}
	o, err := t.bucket.StatObject(t.ctx, req)

	var notFoundErr *gcs.NotFoundError
	if errors.As(err, &notFoundErr) {
		gen = -1
		return
	}

	if err != nil {
		panic(err)
	}

	gen = o.Generation
	return
}

func (t *IntegrationTest) sync(src *gcs.Object) (o *gcs.Object, err error) {
	o, err = t.syncer.SyncObject(t.ctx, src.Name, src, t.tf)
	if err == nil && o != nil {
		t.tf = nil
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *IntegrationTest) ReadThenSync() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Read the contents.
	buf := make([]byte, 1024)
	n, err := t.tf.ReadAt(buf, 0)

	AssertThat(err, AnyOf(io.EOF, nil))
	ExpectEq(len("taco"), n)
	ExpectEq("taco", string(buf[:n]))

	// Sync doesn't need to do anything.
	newObj, err := t.sync(o)

	AssertEq(nil, err)
	ExpectEq(nil, newObj)
}

func (t *IntegrationTest) SyncEmptyLocalFile() {
	// Create a temp file and write some contents to it.
	tf, err := gcsx.NewTempFile(io.NopCloser(strings.NewReader("")), "", &t.clock)
	AssertEq(nil, err)

	// Sync should update the object in GCS.
	newObj, err := t.syncer.SyncObject(t.ctx, "test", nil, tf)

	AssertEq(nil, err)
	ExpectEq(t.objectGeneration("test"), newObj.Generation)
	_, ok := newObj.Metadata["gcsfuse_mtime"]
	AssertFalse(ok)
	// Read via the bucket.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "test")
	AssertEq(nil, err)
	ExpectEq("", string(contents))
	// There should be no junk left over in the bucket besides the object of
	// interest.
	objects, runs, err := storageutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})
	AssertEq(nil, err)
	AssertEq(1, len(objects))
	AssertEq(0, len(runs))
	ExpectEq("test", objects[0].Name)
}

func (t *IntegrationTest) SyncNonEmptyLocalFile() {
	// Create a temp file and write some contents to it.
	tf, err := gcsx.NewTempFile(io.NopCloser(strings.NewReader("")), "", &t.clock)
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()
	n, err := tf.WriteAt([]byte("tacobell"), 0)
	AssertEq(nil, err)
	AssertEq(8, n)
	t.clock.AdvanceTime(time.Second)

	// Sync should update the object in GCS.
	newObj, err := t.syncer.SyncObject(t.ctx, "test", nil, tf)

	AssertEq(nil, err)
	ExpectEq(t.objectGeneration("test"), newObj.Generation)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])
	// Read via the bucket.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "test")
	AssertEq(nil, err)
	ExpectEq("tacobell", string(contents))
	// There should be no junk left over in the bucket besides the object of
	// interest.
	objects, runs, err := storageutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})
	AssertEq(nil, err)
	AssertEq(1, len(objects))
	AssertEq(0, len(runs))
	ExpectEq("test", objects[0].Name)
}

func (t *IntegrationTest) WriteThenSync() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Overwrite the first byte.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()
	n, err := t.tf.WriteAt([]byte("p"), 0)
	t.clock.AdvanceTime(time.Second)

	AssertEq(nil, err)
	ExpectEq(1, n)

	// Sync should save out the new generation.
	newObj, err := t.sync(o)
	AssertEq(nil, err)

	ExpectNe(o.Generation, newObj.Generation)
	ExpectEq(t.objectGeneration("foo"), newObj.Generation)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	// Read via the bucket.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("paco", string(contents))

	// There should be no junk left over in the bucket besides the object of
	// interest.
	objects, runs, err := storageutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	AssertEq(1, len(objects))
	AssertEq(0, len(runs))

	ExpectEq("foo", objects[0].Name)
}

func (t *IntegrationTest) AppendThenSync() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Append some data.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()
	n, err := t.tf.WriteAt([]byte("burrito"), 4)
	t.clock.AdvanceTime(time.Second)

	AssertEq(nil, err)
	ExpectEq(len("burrito"), n)

	// Sync should save out the new generation.
	newObj, err := t.sync(o)
	AssertEq(nil, err)

	ExpectNe(o.Generation, newObj.Generation)
	ExpectEq(t.objectGeneration("foo"), newObj.Generation)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	// Read via the bucket.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))

	// There should be no junk left over in the bucket besides the object of
	// interest.
	objects, runs, err := storageutil.ListAll(
		t.ctx,
		t.bucket,
		&gcs.ListObjectsRequest{})

	AssertEq(nil, err)
	AssertEq(1, len(objects))
	AssertEq(0, len(runs))

	ExpectEq("foo", objects[0].Name)
}

func (t *IntegrationTest) TruncateThenSync() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Truncate.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()
	err = t.tf.Truncate(2)
	t.clock.AdvanceTime(time.Second)

	AssertEq(nil, err)

	// Sync should save out the new generation.
	newObj, err := t.sync(o)
	AssertEq(nil, err)

	ExpectNe(o.Generation, newObj.Generation)
	ExpectEq(t.objectGeneration("foo"), newObj.Generation)
	ExpectEq(
		truncateTime.UTC().Format(time.RFC3339Nano),
		newObj.Metadata["gcsfuse_mtime"])

	contents, err := storageutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("ta", string(contents))
}

func (t *IntegrationTest) Stat_InitialState() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Stat.
	sr, err := t.tf.Stat()
	AssertEq(nil, err)

	ExpectEq(o.Size, sr.Size)
	ExpectEq(o.Size, sr.DirtyThreshold)
	ExpectEq(nil, sr.Mtime)
}

func (t *IntegrationTest) Stat_Dirty() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Dirty.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.tf.Truncate(2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Stat.
	sr, err := t.tf.Stat()
	AssertEq(nil, err)

	ExpectEq(2, sr.Size)
	ExpectEq(2, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(truncateTime)))
}

func (t *IntegrationTest) BackingObjectHasBeenDeleted() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Fault in the contents.
	_, err = t.tf.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	// Delete the backing object.
	err = t.bucket.DeleteObject(t.ctx, &gcs.DeleteObjectRequest{Name: o.Name})
	AssertEq(nil, err)

	// Reading and modications should still work.
	_, err = t.tf.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	_, err = t.tf.WriteAt([]byte("a"), 0)
	AssertEq(nil, err)

	truncateTime := t.clock.Now()
	err = t.tf.Truncate(1)
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	// Stat should see the current state.
	sr, err := t.tf.Stat()
	AssertEq(nil, err)

	ExpectEq(1, sr.Size)
	ExpectEq(0, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(truncateTime)))

	// Sync should fail with a precondition error.
	_, err = t.sync(o)
	var preconditionErr *gcs.PreconditionError
	ExpectTrue(errors.As(err, &preconditionErr))

	// Nothing should have been created.
	_, err = storageutil.ReadObject(t.ctx, t.bucket, o.Name)
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *IntegrationTest) BackingObjectHasBeenOverwritten() {
	// Create.
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	t.create(o)

	// Fault in the contents.
	_, err = t.tf.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	// Overwrite the backing object.
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("burrito"))
	AssertEq(nil, err)

	// Reading and modications should still work.
	_, err = t.tf.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	_, err = t.tf.WriteAt([]byte("a"), 0)
	AssertEq(nil, err)

	truncateTime := t.clock.Now()
	err = t.tf.Truncate(3)
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	// Stat should see the current state.
	sr, err := t.tf.Stat()
	AssertEq(nil, err)

	ExpectEq(3, sr.Size)
	ExpectEq(0, sr.DirtyThreshold)
	ExpectThat(sr.Mtime, Pointee(timeutil.TimeEq(truncateTime)))

	// Sync should fail with a precondition error.
	_, err = t.sync(o)
	var preconditionErr *gcs.PreconditionError
	ExpectTrue(errors.As(err, &preconditionErr))

	// The newer version should still be present.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, o.Name)
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}

func (t *IntegrationTest) MultipleInteractions() {
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
			t.ctx,
			t.bucket,
			name,
			expectedContents)

		AssertEq(nil, err)

		// Create a temp file around it.
		t.create(o)

		// Read the contents of the temp file.
		_, err = t.tf.ReadAt(buf, 0)

		AssertThat(err, AnyOf(nil, io.EOF))
		if !bytes.Equal(buf, expectedContents) {
			AddFailure("Contents mismatch for %s", desc)
			AbortTest()
		}

		// Modify some bytes.
		if size > 0 {
			expectedContents[0] = 17
			expectedContents[size/2] = 19
			expectedContents[size-1] = 23

			_, err = t.tf.WriteAt([]byte{17}, 0)
			AssertEq(nil, err)

			_, err = t.tf.WriteAt([]byte{19}, int64(size/2))
			AssertEq(nil, err)

			_, err = t.tf.WriteAt([]byte{23}, int64(size-1))
			AssertEq(nil, err)
		}

		// Compare contents again.
		_, err = t.tf.ReadAt(buf, 0)

		AssertThat(err, AnyOf(nil, io.EOF))
		if !bytes.Equal(buf, expectedContents) {
			AddFailure("Contents mismatch for %s", desc)
			AbortTest()
		}

		// Sync and recreate if necessary.
		newObj, err := t.sync(o)
		AssertEq(nil, err)

		if newObj != nil {
			t.create(newObj)
		}

		// Check the new backing object's contents.
		objContents, err := storageutil.ReadObject(t.ctx, t.bucket, name)
		AssertEq(nil, err)
		if !bytes.Equal(objContents, expectedContents) {
			AddFailure("Contents mismatch for %s", desc)
			AbortTest()
		}

		// Compare contents again.
		_, err = t.tf.ReadAt(buf, 0)

		AssertThat(err, AnyOf(nil, io.EOF))
		if !bytes.Equal(buf, expectedContents) {
			AddFailure("Contents mismatch for %s", desc)
			AbortTest()
		}

		// Dirty again.
		if size > 0 {
			expectedContents[0] = 29

			_, err = t.tf.WriteAt([]byte{29}, 0)
			AssertEq(nil, err)
		}

		// Compare contents again.
		_, err = t.tf.ReadAt(buf, 0)

		AssertThat(err, AnyOf(nil, io.EOF))
		if !bytes.Equal(buf, expectedContents) {
			AddFailure("Contents mismatch for %s", desc)
			AbortTest()
		}
	}
}

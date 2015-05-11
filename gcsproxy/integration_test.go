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

package gcsproxy_test

import (
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileLeaserLimit = 1 << 10

type IntegrationTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	leaser lease.FileLeaser
	clock  timeutil.SimulatedClock

	mo *checkingMutableObject
}

var _ SetUpInterface = &IntegrationTest{}
var _ TearDownInterface = &IntegrationTest{}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.leaser = lease.NewFileLeaser("", fileLeaserLimit)

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *IntegrationTest) TearDown() {
	if t.mo != nil {
		t.mo.Destroy()
	}
}

func (t *IntegrationTest) create(o *gcs.Object) {
	// Ensure invariants are checked.
	t.mo = &checkingMutableObject{
		ctx: t.ctx,
		wrapped: gcsproxy.NewMutableObject(
			o,
			t.bucket,
			t.leaser,
			&t.clock),
	}
}

// Return the object generation, or -1 if non-existent. Panic on error.
func (t *IntegrationTest) objectGeneration(name string) (gen int64) {
	// Stat.
	req := &gcs.StatObjectRequest{Name: name}
	o, err := t.bucket.StatObject(t.ctx, req)

	if _, ok := err.(*gcs.NotFoundError); ok {
		gen = -1
		return
	}

	if err != nil {
		panic(err)
	}

	// Check the result.
	if o.Generation > math.MaxInt64 {
		panic(fmt.Sprintf("Out of range: %v", o.Generation))
	}

	gen = o.Generation
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *IntegrationTest) ReadThenSync() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Read the contents.
	buf := make([]byte, 1024)
	n, err := t.mo.ReadAt(buf, 0)

	AssertThat(err, AnyOf(io.EOF, nil))
	ExpectEq(len("taco"), n)
	ExpectEq("taco", string(buf[:n]))

	// Sync doesn't need to do anything.
	err = t.mo.Sync()
	ExpectEq(nil, err)

	ExpectEq(o.Generation, t.mo.SourceGeneration())
	ExpectEq(o.Generation, t.objectGeneration("foo"))
}

func (t *IntegrationTest) WriteThenSync() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Overwrite the first byte.
	n, err := t.mo.WriteAt([]byte("p"), 0)

	AssertEq(nil, err)
	ExpectEq(1, n)

	// Sync should save out the new generation.
	err = t.mo.Sync()
	ExpectEq(nil, err)

	ExpectNe(o.Generation, t.mo.SourceGeneration())
	ExpectEq(t.objectGeneration("foo"), t.mo.SourceGeneration())

	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("paco", contents)
}

func (t *IntegrationTest) TruncateThenSync() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Truncate.
	err = t.mo.Truncate(2)
	AssertEq(nil, err)

	// Sync should save out the new generation.
	err = t.mo.Sync()
	ExpectEq(nil, err)

	ExpectNe(o.Generation, t.mo.SourceGeneration())
	ExpectEq(t.objectGeneration("foo"), t.mo.SourceGeneration())

	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, "foo")
	AssertEq(nil, err)
	ExpectEq("ta", contents)
}

func (t *IntegrationTest) Stat_InitialState() {
	// Create.
	createTime := t.clock.Now()
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	t.create(o)

	// Stat.
	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(o.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(createTime))
	ExpectFalse(sr.Clobbered)
}

func (t *IntegrationTest) Stat_Synced() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Dirty.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.mo.Sync()
	AssertEq(nil, err)

	// Stat.
	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(2, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectFalse(sr.Clobbered)
}

func (t *IntegrationTest) Stat_Dirty() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Dirty.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Stat.
	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(2, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectFalse(sr.Clobbered)
}

func (t *IntegrationTest) WithinLeaserLimit() {
	AssertLt(len("taco"), fileLeaserLimit)

	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Extend to be up against the leaser limit, then write out to GCS, which
	// should downgrade to a read proxy.
	err = t.mo.Truncate(fileLeaserLimit)
	AssertEq(nil, err)

	err = t.mo.Sync()
	AssertEq(nil, err)

	// The backing object should be present and contain the correct contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, o.Name)
	AssertEq(nil, err)
	ExpectEq(fileLeaserLimit, len(contents))

	// Delete the backing object.
	err = t.bucket.DeleteObject(t.ctx, o.Name)
	AssertEq(nil, err)

	// We should still be able to read the contents, because the read lease
	// should still be valid.
	buf := make([]byte, 4)
	n, err := t.mo.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("taco", string(buf[0:n]))
}

func (t *IntegrationTest) LargerThanLeaserLimit() {
	AssertLt(len("taco"), fileLeaserLimit)

	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Extend to be past the leaser limit, then write out to GCS, which should
	// downgrade to a read proxy.
	err = t.mo.Truncate(fileLeaserLimit + 1)
	AssertEq(nil, err)

	err = t.mo.Sync()
	AssertEq(nil, err)

	// The backing object should be present and contain the correct contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, o.Name)
	AssertEq(nil, err)
	ExpectEq(fileLeaserLimit+1, len(contents))

	// Delete the backing object.
	err = t.bucket.DeleteObject(t.ctx, o.Name)
	AssertEq(nil, err)

	// The contents should be lost, because the leaser should have revoked the
	// read lease.
	_, err = t.mo.ReadAt([]byte{}, 0)
	ExpectThat(err, Error(HasSubstr("not found")))
}

func (t *IntegrationTest) BackingObjectHasBeenDeleted_BeforeReading() {
	// Create an object to obtain a record, then delete it.
	createTime := t.clock.Now()
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	err = t.bucket.DeleteObject(t.ctx, o.Name)
	AssertEq(nil, err)

	// Create a mutable object around it.
	t.create(o)

	// Synchronously-available things should work.
	ExpectEq(o.Name, t.mo.Name())
	ExpectEq(o.Generation, t.mo.SourceGeneration())

	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(o.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(createTime))
	ExpectTrue(sr.Clobbered)

	// Sync doesn't need to do anything.
	err = t.mo.Sync()
	ExpectEq(nil, err)

	// Anything that needs to fault in the contents should fail.
	_, err = t.mo.ReadAt([]byte{}, 0)
	ExpectThat(err, Error(HasSubstr("not found")))

	err = t.mo.Truncate(10)
	ExpectThat(err, Error(HasSubstr("not found")))

	_, err = t.mo.WriteAt([]byte{}, 0)
	ExpectThat(err, Error(HasSubstr("not found")))
}

func (t *IntegrationTest) BackingObjectHasBeenDeleted_AfterReading() {
	// Create.
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)

	t.create(o)

	// Fault in the contents.
	_, err = t.mo.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	// Delete the backing object.
	err = t.bucket.DeleteObject(t.ctx, o.Name)
	AssertEq(nil, err)

	// Reading and modications should still work.
	ExpectEq(o.Name, t.mo.Name())
	ExpectEq(o.Generation, t.mo.SourceGeneration())

	_, err = t.mo.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	_, err = t.mo.WriteAt([]byte("a"), 0)
	AssertEq(nil, err)

	truncateTime := t.clock.Now()
	err = t.mo.Truncate(1)
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	// Stat should see the current state, and see that the object has been
	// clobbered.
	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(1, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectTrue(sr.Clobbered)

	// Sync should fail with a precondition error.
	err = t.mo.Sync()
	ExpectThat(err, HasSameTypeAs(&gcs.PreconditionError{}))

	// Nothing should have been created.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, o.Name)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *IntegrationTest) BackingObjectHasBeenOverwritten_BeforeReading() {
	// Create an object, then create the mutable object wrapper around it.
	createTime := t.clock.Now()
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", "taco")
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	t.create(o)

	// Overwrite the GCS object.
	_, err = gcsutil.CreateObject(t.ctx, t.bucket, "foo", "burrito")
	AssertEq(nil, err)

	// Synchronously-available things should work.
	ExpectEq(o.Name, t.mo.Name())
	ExpectEq(o.Generation, t.mo.SourceGeneration())

	sr, err := t.mo.Stat(true)
	AssertEq(nil, err)
	ExpectEq(o.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(createTime))
	ExpectTrue(sr.Clobbered)

	// Sync doesn't need to do anything.
	err = t.mo.Sync()
	ExpectEq(nil, err)

	// Anything that needs to fault in the contents should fail.
	_, err = t.mo.ReadAt([]byte{}, 0)
	ExpectThat(err, Error(HasSubstr("not found")))

	err = t.mo.Truncate(10)
	ExpectThat(err, Error(HasSubstr("not found")))

	_, err = t.mo.WriteAt([]byte{}, 0)
	ExpectThat(err, Error(HasSubstr("not found")))
}

func (t *IntegrationTest) BackingObjectHasBeenOverwritten_AfterReading() {
	AssertTrue(false, "TODO")
}

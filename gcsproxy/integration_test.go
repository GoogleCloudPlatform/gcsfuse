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

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

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
	AssertTrue(false, "TODO")
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

func (t *IntegrationTest) Name() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) ReadThenSync() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) WriteThenSync() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) TruncateThenSync() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) Stat_Clean() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) Stat_Dirty() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) SmallerThanLeaserLimit() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) LargerThanLeaserLimit() {
	AssertTrue(false, "TODO")
}

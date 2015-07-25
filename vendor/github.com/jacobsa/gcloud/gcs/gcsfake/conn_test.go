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

package gcsfake_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestConn(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ConnTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock
	conn  gcs.Conn
}

func init() { RegisterTestSuite(&ConnTest{}) }

var _ SetUpInterface = &ConnTest{}

func (t *ConnTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.conn = gcsfake.NewConn(&t.clock)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConnTest) BucketContentsAreStable() {
	const bucketName = "foo"
	const objName = "bar"

	var err error

	// Open the bucket.
	bucket, err := t.conn.OpenBucket(t.ctx, bucketName)
	AssertEq(nil, err)

	// Add an object to a bucket.
	_, err = gcsutil.CreateObject(
		t.ctx,
		bucket,
		objName,
		"taco")

	AssertEq(nil, err)

	// Grab the bucket again. It should still be there.
	contents, err := gcsutil.ReadObject(
		t.ctx,
		bucket,
		objName)

	AssertEq(nil, err)
	ExpectEq("taco", string(contents))
}

func (t *ConnTest) BucketsAreSegregatedByName() {
	const objName = "baz"
	var contents []byte
	var err error

	b0, err := t.conn.OpenBucket(t.ctx, "foo")
	AssertEq(nil, err)

	b1, err := t.conn.OpenBucket(t.ctx, "bar")
	AssertEq(nil, err)

	// Add an object with the same name but different contents to each of two
	// buckets.
	_, err = gcsutil.CreateObject(t.ctx, b0, objName, "taco")
	AssertEq(nil, err)

	_, err = gcsutil.CreateObject(t.ctx, b1, objName, "burrito")
	AssertEq(nil, err)

	// Each should have stored it independently.
	contents, err = gcsutil.ReadObject(t.ctx, b0, objName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	contents, err = gcsutil.ReadObject(t.ctx, b1, objName)
	AssertEq(nil, err)
	ExpectEq("burrito", string(contents))
}

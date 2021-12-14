// Copyright 2021 Google Inc. All Rights Reserved.
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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestCore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type CoreTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock
}

var _ SetUpInterface = &CoreTest{}
var _ TearDownInterface = &CoreTest{}

func init() { RegisterTestSuite(&CoreTest{}) }

func (t *CoreTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.bucket = gcsx.NewSyncerBucket(
		1, ".gcsfuse_tmp/", gcsfake.NewFakeBucket(&t.clock, "some_bucket"))
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *CoreTest) TearDown() {}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CoreTest) File() {
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	AssertEq(nil, err)

	name := inode.NewFileName(inode.NewRootName(t.bucket.Name()), o.Name)
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      o,
		ImplicitDir: false,
	}
	ExpectTrue(c.Exists())
}

func (t *CoreTest) ExplicitDir() {
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "bar/", []byte(""))
	AssertEq(nil, err)

	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), o.Name)
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      o,
		ImplicitDir: false,
	}
	ExpectTrue(c.Exists())
}

func (t *CoreTest) ImplicitDir() {
	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), "bar/")
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      nil,
		ImplicitDir: true,
	}
	ExpectTrue(c.Exists())
}

func (t *CoreTest) BucketRootDir() {
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    inode.NewRootName(t.bucket.Name()),
		Object:      nil,
		ImplicitDir: false,
	}
	ExpectTrue(c.Exists())
}

func (t *CoreTest) Nonexistent() {
	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), "bar/")
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      nil,
		ImplicitDir: false,
	}
	ExpectFalse(c.Exists())
}

func (t *CoreTest) SanityCheck() {
	o, err := gcsutil.CreateObject(t.ctx, t.bucket, "bar/", []byte(""))
	AssertEq(nil, err)

	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), o.Name)
	c := inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      o,
		ImplicitDir: true,
	}
	ExpectNe(nil, c.SanityCheck())

	c = inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      nil,
		ImplicitDir: true,
	}
	ExpectEq(nil, c.SanityCheck())

	c = inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      nil,
		ImplicitDir: false,
	}
	ExpectEq(nil, c.SanityCheck())

	o.Name = "foo/"
	c = inode.Core{
		Bucket:      t.bucket,
		FullName:    name,
		Object:      o,
		ImplicitDir: false,
	}
	ExpectNe(nil, c.SanityCheck())
}

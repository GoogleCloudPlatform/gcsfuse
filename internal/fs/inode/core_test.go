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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
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
		1, ".gcsfuse_tmp/", fake.NewFakeBucket(&t.clock, "some_bucket"))
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *CoreTest) TearDown() {}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CoreTest) File() {
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "foo", []byte("taco"))
	m := storageutil.ConvertObjToMinObject(o)
	AssertEq(nil, err)

	name := inode.NewFileName(inode.NewRootName(t.bucket.Name()), m.Name)
	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  name,
		MinObject: m,
	}
	ExpectTrue(c.Exists())
	ExpectEq(metadata.RegularFileType, c.Type())
}

func (t *CoreTest) LocalFile() {
	name := inode.NewFileName(inode.NewRootName(t.bucket.Name()), "test")
	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  name,
		MinObject: nil,
		Local:     true,
	}
	ExpectTrue(c.Exists())
	ExpectEq(metadata.RegularFileType, c.Type())
}

func (t *CoreTest) ExplicitDir() {
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "bar/", []byte(""))
	m := storageutil.ConvertObjToMinObject(o)
	AssertEq(nil, err)

	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), m.Name)
	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  name,
		MinObject: m,
	}
	ExpectTrue(c.Exists())
	ExpectEq(metadata.ExplicitDirType, c.Type())
}

func (t *CoreTest) ImplicitDir() {
	name := inode.NewDirName(inode.NewRootName(t.bucket.Name()), "bar/")
	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  name,
		MinObject: nil,
	}
	ExpectTrue(c.Exists())
	ExpectEq(metadata.ImplicitDirType, c.Type())
}

func (t *CoreTest) BucketRootDir() {
	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewRootName(t.bucket.Name()),
		MinObject: nil,
	}
	ExpectTrue(c.Exists())
	ExpectEq(metadata.ImplicitDirType, c.Type())
}

func (t *CoreTest) Nonexistent() {
	var c *inode.Core
	ExpectFalse(c.Exists())
	ExpectEq(metadata.UnknownType, c.Type())
}

func (t *CoreTest) SanityCheck() {
	root := inode.NewRootName(t.bucket.Name())
	o, err := storageutil.CreateObject(t.ctx, t.bucket, "bar", []byte(""))
	m := storageutil.ConvertObjToMinObject(o)
	AssertEq(nil, err)

	c := &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewDirName(root, "bar"),
		MinObject: nil,
	}
	ExpectEq(nil, c.SanityCheck()) // implicit dir is okay

	c = &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewFileName(root, m.Name),
		MinObject: m,
	}
	ExpectEq(nil, c.SanityCheck()) // name match

	c = &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewFileName(root, "foo"),
		MinObject: m,
	}
	ExpectNe(nil, c.SanityCheck()) // name mismatch

	c = &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewFileName(root, "foo"),
		MinObject: nil,
		Local:     true,
	}
	ExpectEq(nil, c.SanityCheck()) // object is nil for local fileInode.

	c = &inode.Core{
		Bucket:    &t.bucket,
		FullName:  inode.NewFileName(root, "foo"),
		MinObject: nil,
		Local:     false,
	}
	ExpectNe(nil, c.SanityCheck()) // Missing object for non-local fileInode.
}

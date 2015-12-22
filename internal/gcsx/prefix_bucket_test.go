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
	"io/ioutil"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestPrefixBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type PrefixBucketTest struct {
	ctx     context.Context
	prefix  string
	wrapped gcs.Bucket
	bucket  gcs.Bucket
}

var _ SetUpInterface = &PrefixBucketTest{}

func init() { RegisterTestSuite(&PrefixBucketTest{}) }

func (t *PrefixBucketTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx
	t.prefix = "foo_"
	t.wrapped = gcsfake.NewFakeBucket(timeutil.RealClock(), "some_bucket")

	t.bucket, err = gcsx.NewPrefixBucket(t.prefix, t.wrapped)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *PrefixBucketTest) Name() {
	ExpectEq(t.wrapped.Name(), t.bucket.Name())
}

func (t *PrefixBucketTest) NewReader() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = gcsutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	AssertEq(nil, err)

	// Read it through the prefix bucket.
	rc, err := t.bucket.NewReader(
		t.ctx,
		&gcs.ReadObjectRequest{
			Name: suffix,
		})

	AssertEq(nil, err)
	defer rc.Close()

	actual, err := ioutil.ReadAll(rc)
	AssertEq(nil, err)
	ExpectEq(contents, string(actual))
}

func (t *PrefixBucketTest) CreateObject() {
	var err error
	suffix := "taco"
	contents := "foobar"

	// Create the object.
	o, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:            suffix,
			ContentLanguage: "en-GB",
			Contents:        strings.NewReader(contents),
		})

	AssertEq(nil, err)
	ExpectEq(suffix, o.Name)
	ExpectEq("en-GB", o.ContentLanguage)

	// Read it through the back door.
	actual, err := gcsutil.ReadObject(t.ctx, t.wrapped, t.prefix+suffix)
	AssertEq(nil, err)
	ExpectEq(contents, string(actual))
}

func (t *PrefixBucketTest) CopyObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = gcsutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	AssertEq(nil, err)

	// Copy it to a new name.
	newSuffix := "burrito"
	o, err := t.bucket.CopyObject(
		t.ctx,
		&gcs.CopyObjectRequest{
			SrcName: suffix,
			DstName: newSuffix,
		})

	AssertEq(nil, err)
	ExpectEq(newSuffix, o.Name)

	// Read it through the back door.
	actual, err := gcsutil.ReadObject(t.ctx, t.wrapped, t.prefix+newSuffix)
	AssertEq(nil, err)
	ExpectEq(contents, string(actual))
}

func (t *PrefixBucketTest) ComposeObjects() {
	var err error

	suffix0 := "taco"
	contents0 := "foo"

	suffix1 := "burrito"
	contents1 := "bar"

	// Create two objects through the back door.
	err = gcsutil.CreateObjects(
		t.ctx,
		t.wrapped,
		map[string][]byte{
			(t.prefix + suffix0): []byte(contents0),
			(t.prefix + suffix1): []byte(contents1),
		})

	AssertEq(nil, err)

	// Compose them.
	newSuffix := "enchilada"
	o, err := t.bucket.ComposeObjects(
		t.ctx,
		&gcs.ComposeObjectsRequest{
			DstName: newSuffix,
			Sources: []gcs.ComposeSource{
				{Name: suffix0},
				{Name: suffix1},
			},
		})

	AssertEq(nil, err)
	ExpectEq(newSuffix, o.Name)

	// Read it through the back door.
	actual, err := gcsutil.ReadObject(t.ctx, t.wrapped, t.prefix+newSuffix)
	AssertEq(nil, err)
	ExpectEq(contents0+contents1, string(actual))
}

func (t *PrefixBucketTest) StatObject() {
	var err error
	suffix := "taco"
	name := t.prefix + suffix
	contents := "foobar"

	// Create an object through the back door.
	_, err = gcsutil.CreateObject(t.ctx, t.wrapped, name, []byte(contents))
	AssertEq(nil, err)

	// Stat it.
	o, err := t.bucket.StatObject(
		t.ctx,
		&gcs.StatObjectRequest{
			Name: suffix,
		})

	AssertEq(nil, err)
	ExpectEq(suffix, o.Name)
	ExpectEq(len(contents), o.Size)
}

func (t *PrefixBucketTest) ListObjects() {
	AddFailure("TODO")
}

func (t *PrefixBucketTest) UpdateObject() {
	AddFailure("TODO")
}

func (t *PrefixBucketTest) DeleteObject() {
	AddFailure("TODO")
}

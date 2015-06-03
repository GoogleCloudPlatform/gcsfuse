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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/mutable"
	"github.com/googlecloudplatform/gcsfuse/mutable/mock"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestObjectSyncer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ObjectSyncerTest struct {
	ctx context.Context

	srcObject gcs.Object
	content   mock_mutable.MockContent
	bucket    mock_gcs.MockBucket

	simulatedContents []byte
}

var _ SetUpInterface = &ObjectSyncerTest{}

func init() { RegisterTestSuite(&ObjectSyncerTest{}) }

func (t *ObjectSyncerTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the source object.
	t.srcObject.Generation = 1234
	t.srcObject.Name = "foo"
	t.srcObject.Size = 17

	// Set up dependencies.
	t.content = mock_mutable.NewMockContent(
		ti.MockController,
		"content")

	t.bucket = mock_gcs.NewMockBucket(
		ti.MockController,
		"bucket")

	// By default, show the content as dirty.
	sr := mutable.StatResult{
		DirtyThreshold: int64(t.srcObject.Size - 1),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillRepeatedly(Return(sr, nil))

	// Set up fake contents.
	t.simulatedContents = []byte("taco")
	ExpectCall(t.content, "ReadAt")(Any(), Any(), Any()).
		WillRepeatedly(Invoke(t.serveReadAt))

	// And for the released read/write lease.
	leaser := lease.NewFileLeaser("", math.MaxInt32, math.MaxInt32)
	rwl, err := leaser.NewFile()
	AssertEq(nil, err)

	_, err = rwl.Write(t.simulatedContents)
	AssertEq(nil, err)

	ExpectCall(t.content, "Release")().
		WillRepeatedly(Return(rwl))
}

func (t *ObjectSyncerTest) call() (rl lease.ReadLease, o *gcs.Object, err error) {
	rl, o, err = gcsproxy.Sync(
		t.ctx,
		&t.srcObject,
		t.content,
		t.bucket)

	return
}

func (t *ObjectSyncerTest) serveReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, err error) {
	// Handle out of range reads.
	if offset > int64(len(t.simulatedContents)) {
		err = io.EOF
		return
	}

	// Copy into the buffer.
	n = copy(p, t.simulatedContents[int(offset):])
	if n < len(p) {
		err = io.EOF
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ObjectSyncerTest) StatFails() {
	// Stat
	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(mutable.StatResult{}, errors.New("taco")))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("Stat")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectSyncerTest) StatReturnsWackyDirtyThreshold() {
	// Stat
	sr := mutable.StatResult{
		DirtyThreshold: int64(t.srcObject.Size + 1),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("Stat")))
	ExpectThat(err, Error(HasSubstr("DirtyThreshold")))
	ExpectThat(err, Error(HasSubstr(fmt.Sprint(t.srcObject.Size))))
	ExpectThat(err, Error(HasSubstr(fmt.Sprint(t.srcObject.Size+1))))
}

func (t *ObjectSyncerTest) StatSaysNotDirty() {
	// Stat
	sr := mutable.StatResult{
		Size:           int64(t.srcObject.Size),
		DirtyThreshold: int64(t.srcObject.Size),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// Call
	rl, o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(nil, rl)
	ExpectEq(nil, o)
}

func (t *ObjectSyncerTest) StatSaysDirty_SameSizeAsSource() {
	// Stat
	sr := mutable.StatResult{
		Size:           int64(t.srcObject.Size),
		DirtyThreshold: int64(t.srcObject.Size - 1),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *ObjectSyncerTest) StatSaysDirty_SmallerThanSource() {
	// Stat
	sr := mutable.StatResult{
		Size:           int64(t.srcObject.Size - 1),
		DirtyThreshold: int64(t.srcObject.Size - 1),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *ObjectSyncerTest) StatSaysDirty_LargerThanSource() {
	// Stat
	sr := mutable.StatResult{
		Size:           int64(t.srcObject.Size + 1),
		DirtyThreshold: int64(t.srcObject.Size),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *ObjectSyncerTest) CallsBucket() {
	// CreateObject
	var req *gcs.CreateObjectRequest
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &req), Return(nil, errors.New(""))))

	// Call
	t.call()

	AssertNe(nil, req)
	ExpectEq(t.srcObject.Name, req.Name)
	ExpectThat(req.GenerationPrecondition, Pointee(Equals(t.srcObject.Generation)))

	b, err := ioutil.ReadAll(req.Contents)
	AssertEq(nil, err)
	ExpectEq(string(t.simulatedContents), string(b))
}

func (t *ObjectSyncerTest) BucketFails() {
	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectSyncerTest) BucketReturnsPreconditionError() {
	// CreateObject
	expected := &gcs.PreconditionError{}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, expected))

	// Call
	_, _, err := t.call()

	ExpectEq(expected, err)
}

func (t *ObjectSyncerTest) BucketSucceeds() {
	// CreateObject
	expected := &gcs.Object{}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Call
	rl, o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(expected, o)

	_, err = rl.Seek(0, 0)
	AssertEq(nil, err)

	b, err := ioutil.ReadAll(rl)
	AssertEq(nil, err)
	ExpectEq(string(t.simulatedContents), string(b))
}

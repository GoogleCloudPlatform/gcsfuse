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
	"io/ioutil"
	"math"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/gcsproxy/mock"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestSync(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SyncTest struct {
	ctx context.Context

	srcObject gcs.Object
	content   mock_gcsproxy.MockMutableContent
	bucket    mock_gcs.MockBucket

	simulatedContents []byte
}

var _ SetUpInterface = &SyncTest{}

func init() { RegisterTestSuite(&SyncTest{}) }

func (t *SyncTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the source object.
	t.srcObject.Generation = 1234
	t.srcObject.Name = "foo"
	t.srcObject.Size = 17

	// Set up dependencies.
	t.content = mock_gcsproxy.NewMockMutableContent(
		ti.MockController,
		"content")

	t.bucket = mock_gcs.NewMockBucket(
		ti.MockController,
		"bucket")

	// By default, show the content as dirty.
	sr := gcsproxy.StatResult{
		DirtyThreshold: int64(t.srcObject.Size - 1),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillRepeatedly(Return(sr, nil))

	// Set up fake contents.
	t.simulatedContents = []byte("taco")

	leaser := lease.NewFileLeaser("", math.MaxInt32, math.MaxInt32)
	rwl, err := leaser.NewFile()
	AssertEq(nil, err)

	_, err = rwl.Write(t.simulatedContents)
	AssertEq(nil, err)

	ExpectCall(t.content, "Release")().
		WillRepeatedly(Return(rwl))
}

func (t *SyncTest) call() (rp lease.ReadProxy, o *gcs.Object, err error) {
	rp, o, err = gcsproxy.Sync(
		t.ctx,
		&t.srcObject,
		t.content,
		t.bucket)

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SyncTest) StatFails() {
	// Stat
	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(gcsproxy.StatResult{}, errors.New("taco")))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("Stat")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *SyncTest) StatReturnsWackyDirtyThreshold() {
	// Stat
	sr := gcsproxy.StatResult{
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

func (t *SyncTest) StatSaysNotDirty() {
	// Stat
	sr := gcsproxy.StatResult{
		DirtyThreshold: int64(t.srcObject.Size),
	}

	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(sr, nil))

	// Call
	rp, o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(nil, rp)
	ExpectEq(nil, o)
}

func (t *SyncTest) CallsBucket() {
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

func (t *SyncTest) BucketFails() {
	// CreateObject
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *SyncTest) BucketReturnsPreconditionError() {
	// CreateObject
	expected := &gcs.PreconditionError{}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, expected))

	// Call
	_, _, err := t.call()

	ExpectEq(expected, err)
}

func (t *SyncTest) BucketSucceeds() {
	// CreateObject
	expected := &gcs.Object{}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(Return(expected, nil))

	// Call
	rp, o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(expected, o)

	buf := make([]byte, 1024)
	n, err := rp.ReadAt(t.ctx, buf, 0)

	AssertEq(nil, err)
	ExpectEq(string(t.simulatedContents), string(buf[:n]))
}

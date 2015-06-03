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

package gcsproxy

import (
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

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

func TestStattingObjectSyncer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type StattingObjectSyncerTest struct {
	ctx context.Context

	srcObject gcs.Object
	content   mock_mutable.MockContent
	bucket    mock_gcs.MockBucket

	syncer ObjectSyncer

	simulatedContents []byte
}

var _ SetUpInterface = &StattingObjectSyncerTest{}

func init() { RegisterTestSuite(&StattingObjectSyncerTest{}) }

func (t *StattingObjectSyncerTest) SetUp(ti *TestInfo) {
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

	// Set up the syncer.
	t.syncer = NewObjectSyncer(t.bucket)

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

func (t *StattingObjectSyncerTest) call() (
	rl lease.ReadLease, o *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}

func (t *StattingObjectSyncerTest) serveReadAt(
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

func (t *StattingObjectSyncerTest) StatFails() {
	// Stat
	ExpectCall(t.content, "Stat")(Any()).
		WillOnce(Return(mutable.StatResult{}, errors.New("taco")))

	// Call
	_, _, err := t.call()

	ExpectThat(err, Error(HasSubstr("Stat")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *StattingObjectSyncerTest) StatReturnsWackyDirtyThreshold() {
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

func (t *StattingObjectSyncerTest) StatSaysNotDirty() {
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

func (t *StattingObjectSyncerTest) SmallerThanSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SameSizeAsSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) LargerThanSource_ThresholdInSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) LargerThanSource_ThresholdAtEndOfSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncFullFails() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncFullSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncAppendFails() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncAppendSucceeds() {
	AssertTrue(false, "TODO")
}

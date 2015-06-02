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

	"github.com/googlecloudplatform/gcsfuse/gcsproxy/mock"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestSync(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SyncTest struct {
	ctx context.Context

	srcObject  gcs.Object
	newContent mock_gcsproxy.MockMutableContent
	bucket     mock_gcs.MockBucket
}

var _ SetUpInterface = &SyncTest{}

func init() { RegisterTestSuite(&SyncTest{}) }

func (t *SyncTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Set up the source object.
	t.srcObject.Generation = 1234
	t.srcObject.Name = "foo"

	// Set up dependencies.
	t.newContent = mock_gcsproxy.NewMockMutableContent(
		ti.MockController,
		"newContent")

	t.bucket = mock_gcs.NewMockBucket(
		ti.MockController,
		"bucket")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SyncTest) StatFails() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) StatReturnsWackyDirtyThreshold() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) StatSaysNotDirty() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) CallsUpgrade() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) UpgradeFails() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) CallsBucket() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) BucketFails() {
	AssertTrue(false, "TODO")
}

func (t *SyncTest) BucketSucceeds() {
	AssertTrue(false, "TODO")
}

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

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileLeaserLimit = 1 << 10

type IntegrationTest struct {
	bucket gcs.Bucket
	leaser lease.FileLeaser
	clock  timeutil.SimulatedClock
}

var _ SetUpInterface = &IntegrationTest{}
var _ TearDownInterface = &IntegrationTest{}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(ti *TestInfo) {
	panic("TODO")

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *IntegrationTest) TearDown() {
	// TODO(jacobsa): Call Destroy to make sure nothing weird happens. Make sure
	// checkingMutableObject checks around Destroy, too.
	panic("TODO")
}

func (t *IntegrationTest) createMutableObject(
	o *gcs.Object) (mo *checkingMutableObject)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *IntegrationTest) NonExistentBackingObjectName() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) BackingObjectHasBeenClobbered_BeforeReading() {
	AssertTrue(false, "TODO")
}

func (t *IntegrationTest) BackingObjectHasBeenClobbered_AfterReading() {
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

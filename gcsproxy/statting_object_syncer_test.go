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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/gcloud/gcs"
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
}

var _ SetUpInterface = &StattingObjectSyncerTest{}

func init() { RegisterTestSuite(&StattingObjectSyncerTest{}) }

func (t *StattingObjectSyncerTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) call() (
	rl lease.ReadLease, o *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StattingObjectSyncerTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

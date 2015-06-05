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

package main

import (
	"flag"
	"log"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
)

func TestRun(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type RunTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock
	conn  gcs.Conn
}

var _ SetUpInterface = &RunTest{}

func init() { RegisterTestSuite(&RunTest{}) }

func (t *RunTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.conn = gcsfake.NewConn(&t.clock)
}

func (t *RunTest) start(args []string) (join <-chan struct{}) {
	joinChan := make(chan struct{})
	join = joinChan

	go func() {
		err := run(
			args,
			new(flag.FlagSet),
			t.conn,
			t.handleSIGINT)

		ExpectEq(nil, err)
		close(joinChan)
	}()

	return
}

func (t *RunTest) handleSIGINT(mountPoint string) {
	log.Println("Received SIGINT; exiting after this test completes.")
	StopRunningTests()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RunTest) BasicUsage() {
	AssertTrue(false, "TODO")
}

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
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
)

func TestMount(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MountTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock
	conn  gcs.Conn

	// A temporary directory that is cleaned up at the end of the test run.
	dir string
}

var _ SetUpInterface = &MountTest{}
var _ TearDownInterface = &MountTest{}

func init() { RegisterTestSuite(&MountTest{}) }

func (t *MountTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.conn = gcsfake.NewConn(&t.clock)

	// Set up the temporary directory.
	t.dir, err = ioutil.TempDir("", "run_test")
	AssertEq(nil, err)
}

func (t *MountTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

func (t *MountTest) start(args []string) (join <-chan struct{}) {
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

func (t *MountTest) handleSIGINT(mountPoint string) {
	log.Println("Received SIGINT; exiting after this test completes.")
	StopRunningTests()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MountTest) BasicUsage() {
	var err error
	const fileName = "foo"

	// Grab a bucket.
	bucket := t.conn.GetBucket("some_bucket")

	// Mount that bucket.
	join := t.start([]string{
		bucket.Name(),
		t.dir,
	})

	// Create a file.
	err = ioutil.WriteFile(path.Join(t.dir, fileName), []byte("taco"), 0400)
	AssertEq(nil, err)

	// Read the object from the bucket.
	contents, err := gcsutil.ReadObject(t.ctx, bucket, fileName)
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Read the file.
	contents, err = ioutil.ReadFile(path.Join(t.dir, fileName))
	AssertEq(nil, err)
	ExpectEq("taco", string(contents))

	// Unmount and join.
	err = fuse.Unmount(t.dir)
	AssertEq(nil, err)
	<-join
}

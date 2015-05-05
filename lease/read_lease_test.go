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

package lease_test

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/fuse/fsutil"
	. "github.com/jacobsa/ogletest"
)

func TestReadLease(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func doNothingForRevoke() {
}

func panicForUpgrade(f *os.File) *lease.WriteLease {
	panic("panicForUpgrade should not be called")
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileContents = "taco"

type ReadLeaseTest struct {
	f *os.File
}

var _ SetUpInterface = &ReadLeaseTest{}

func init() { RegisterTestSuite(&ReadLeaseTest{}) }

func (t *ReadLeaseTest) SetUp(ti *TestInfo) {
	var err error

	// Set up a temporary file.
	t.f, err = fsutil.AnonymousFile("")
	AssertEq(nil, err)

	// Write the initial contents to it.
	_, err = t.f.Write([]byte(fileContents))
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReadLeaseTest) ReadWhileAvailable() {
	var err error

	// Create the lease.
	rl := lease.NewReadLease(t.f, doNothingForRevoke, panicForUpgrade)

	// Read from it.
	buf := make([]byte, 2)
	_, err = rl.ReadAt(buf, 1)

	AssertEq(nil, err)
	ExpectEq(fileContents[1:3], string(buf))
}

func (t *ReadLeaseTest) Revoke() {
	AssertTrue(false, "TODO")
}

func (t *ReadLeaseTest) Upgrade() {
	AssertTrue(false, "TODO")
}

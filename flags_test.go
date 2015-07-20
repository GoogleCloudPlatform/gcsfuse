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
	"os"
	"testing"
	"time"

	. "github.com/jacobsa/ogletest"
	"github.com/jgeewax/cli"
)

func TestFlags(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FlagsTest struct {
}

func init() { RegisterTestSuite(&FlagsTest{}) }

func parseArgs(args []string) (flags *flagStorage) {
	// Create a CLI app, and abuse it to snoop on the flags.
	app := newApp()
	app.Action = func(appCtx *cli.Context) {
		flags = populateFlags(appCtx)
	}

	// Simulate argv.
	fullArgs := append([]string{"some_app"}, args...)

	err := app.Run(fullArgs)
	AssertEq(nil, err)

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FlagsTest) Defaults() {
	f := parseArgs([]string{})

	// File system
	ExpectEq(0, len(f.MountOptions), "Options: %v", f.MountOptions)
	ExpectEq(os.FileMode(0755), f.DirMode)
	ExpectEq(os.FileMode(0644), f.FileMode)
	ExpectEq(-1, f.Uid)
	ExpectEq(-1, f.Gid)
	ExpectFalse(f.ImplicitDirs)

	// GCS
	ExpectEq("", f.KeyFile)
	ExpectEq(-1, f.EgressBandwidthLimitBytesPerSecond)
	ExpectEq(5, f.OpRateLimitHz)

	// Tuning
	ExpectEq(time.Minute, f.StatCacheTTL)
	ExpectEq(time.Minute, f.TypeCacheTTL)
	ExpectEq(1<<24, f.GCSChunkSize)
	ExpectEq("", f.TempDir)
	ExpectEq(1<<31, f.TempDirLimit)

	// Debugging
	ExpectFalse(f.DebugFuse)
	ExpectFalse(f.DebugGCS)
	ExpectFalse(f.DebugHTTP)
	ExpectFalse(f.DebugInvariants)
}

func (t *FlagsTest) Bools() {
	AssertTrue(false, "TODO")
}

func (t *FlagsTest) Integers() {
	AssertTrue(false, "TODO")
}

func (t *FlagsTest) Strings() {
	AssertTrue(false, "TODO")
}

func (t *FlagsTest) Durations() {
	AssertTrue(false, "TODO")
}

func (t *FlagsTest) Maps() {
	AssertTrue(false, "TODO")
}

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
	"fmt"
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
	names := []string{
		"implicit-dirs",
		"debug_fuse",
		"debug_gcs",
		"debug_http",
		"debug_invariants",
	}

	var args []string
	var f *flagStorage

	// --foo form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s", n))
	}

	f = parseArgs(args)
	ExpectTrue(f.ImplicitDirs)
	ExpectTrue(f.DebugFuse)
	ExpectTrue(f.DebugGCS)
	ExpectTrue(f.DebugHTTP)
	ExpectTrue(f.DebugInvariants)

	// --foo=false form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=false", n))
	}

	f = parseArgs(args)
	ExpectFalse(f.ImplicitDirs)
	ExpectFalse(f.DebugFuse)
	ExpectFalse(f.DebugGCS)
	ExpectFalse(f.DebugHTTP)
	ExpectFalse(f.DebugInvariants)

	// --foo=true form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=true", n))
	}

	f = parseArgs(args)
	ExpectTrue(f.ImplicitDirs)
	ExpectTrue(f.DebugFuse)
	ExpectTrue(f.DebugGCS)
	ExpectTrue(f.DebugHTTP)
	ExpectTrue(f.DebugInvariants)
}

func (t *FlagsTest) Numbers() {
	args := []string{
		"--dir-mode=0711",
		"--file-mode", "0611",
		"--uid=17",
		"--gid=19",
		"--limit-bytes-per-sec=123.4",
		"--limit-ops-per-sec=56.78",
		"--gcs-chunk-size=1000",
		"--temp-dir-bytes=2000",
	}

	f := parseArgs(args)
	ExpectEq(os.FileMode(0711), f.DirMode)
	ExpectEq(os.FileMode(0611), f.FileMode)
	ExpectEq(17, f.Uid)
	ExpectEq(19, f.Gid)
	ExpectEq(123.4, f.EgressBandwidthLimitBytesPerSecond)
	ExpectEq(56.78, f.OpRateLimitHz)
	ExpectEq(1000, f.GCSChunkSize)
	ExpectEq(2000, f.TempDirLimit)
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

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
	"sort"
	"testing"
	"time"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/urfave/cli"
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
	ExpectNe(nil, f.MountOptions)
	ExpectEq(0, len(f.MountOptions), "Options: %v", f.MountOptions)

	ExpectEq(os.FileMode(0755), f.DirMode)
	ExpectEq(os.FileMode(0644), f.FileMode)
	ExpectEq(-1, f.Uid)
	ExpectEq(-1, f.Gid)
	ExpectFalse(f.ImplicitDirs)

	// GCS
	ExpectEq("", f.KeyFile)
	ExpectEq(-1, f.EgressBandwidthLimitBytesPerSecond)
	ExpectEq(-1, f.OpRateLimitHz)

	// Tuning
	ExpectEq(4096, f.StatCacheCapacity)
	ExpectEq(time.Minute, f.StatCacheTTL)
	ExpectEq(time.Minute, f.TypeCacheTTL)
	ExpectEq("", f.TempDir)

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

func (t *FlagsTest) DecimalNumbers() {
	args := []string{
		"--uid=17",
		"--gid=19",
		"--limit-bytes-per-sec=123.4",
		"--limit-ops-per-sec=56.78",
		"--stat-cache-capacity=8192",
	}

	f := parseArgs(args)
	ExpectEq(17, f.Uid)
	ExpectEq(19, f.Gid)
	ExpectEq(123.4, f.EgressBandwidthLimitBytesPerSecond)
	ExpectEq(56.78, f.OpRateLimitHz)
	ExpectEq(8192, f.StatCacheCapacity)
}

func (t *FlagsTest) OctalNumbers() {
	args := []string{
		"--dir-mode=711",
		"--file-mode", "611",
	}

	f := parseArgs(args)
	ExpectEq(os.FileMode(0711), f.DirMode)
	ExpectEq(os.FileMode(0611), f.FileMode)
}

func (t *FlagsTest) Strings() {
	args := []string{
		"--key-file", "-asdf",
		"--temp-dir=foobar",
		"--only-dir=baz",
	}

	f := parseArgs(args)
	ExpectEq("-asdf", f.KeyFile)
	ExpectEq("foobar", f.TempDir)
	ExpectEq("baz", f.OnlyDir)
}

func (t *FlagsTest) Durations() {
	args := []string{
		"--stat-cache-ttl", "1m17s",
		"--type-cache-ttl", "19ns",
	}

	f := parseArgs(args)
	ExpectEq(77*time.Second, f.StatCacheTTL)
	ExpectEq(19*time.Nanosecond, f.TypeCacheTTL)
}

func (t *FlagsTest) Maps() {
	args := []string{
		"-o", "rw,nodev",
		"-o", "user=jacobsa,noauto",
	}

	f := parseArgs(args)

	var keys sort.StringSlice
	for k := range f.MountOptions {
		keys = append(keys, k)
	}

	sort.Sort(keys)
	AssertThat(keys, ElementsAre("noauto", "nodev", "rw", "user"))

	ExpectEq("", f.MountOptions["noauto"])
	ExpectEq("", f.MountOptions["nodev"])
	ExpectEq("", f.MountOptions["rw"])
	ExpectEq("jacobsa", f.MountOptions["user"])
}

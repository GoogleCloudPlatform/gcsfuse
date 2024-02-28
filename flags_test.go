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
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/mount"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
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
	var err error
	app.Action = func(appCtx *cli.Context) {
		flags, err = populateFlags(appCtx)
		AssertEq(nil, err)
	}

	// Simulate argv.
	fullArgs := append([]string{"some_app"}, args...)

	err = app.Run(fullArgs)
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
	ExpectTrue(f.ReuseTokenFromUrl)
	ExpectEq(nil, f.CustomEndpoint)

	// Tuning
	ExpectEq(mount.DefaultStatCacheCapacity, f.StatCacheCapacity)
	ExpectEq(mount.DefaultStatOrTypeCacheTTL, f.StatCacheTTL)
	ExpectEq(mount.DefaultStatOrTypeCacheTTL, f.TypeCacheTTL)
	ExpectEq(0, f.HttpClientTimeout)
	ExpectEq("", f.TempDir)
	ExpectEq(2, f.RetryMultiplier)
	ExpectFalse(f.EnableNonexistentTypeCache)

	// Logging
	ExpectTrue(f.DebugFuseErrors)

	// Debugging
	ExpectFalse(f.DebugFuse)
	ExpectFalse(f.DebugGCS)
	ExpectFalse(f.DebugHTTP)
	ExpectFalse(f.DebugInvariants)
}

func (t *FlagsTest) Bools() {
	names := []string{
		"implicit-dirs",
		"reuse-token-from-url",
		"debug_fuse_errors",
		"debug_fuse",
		"debug_http",
		"debug_gcs",
		"debug_invariants",
		"enable-nonexistent-type-cache",
		"experimental-enable-json-read",
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
	ExpectTrue(f.ReuseTokenFromUrl)
	ExpectTrue(f.DebugFuseErrors)
	ExpectTrue(f.DebugFuse)
	ExpectTrue(f.DebugGCS)
	ExpectTrue(f.DebugHTTP)
	ExpectTrue(f.DebugInvariants)
	ExpectTrue(f.EnableNonexistentTypeCache)
	ExpectTrue(f.ExperimentalEnableJsonRead)

	// --foo=false form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=false", n))
	}

	f = parseArgs(args)
	ExpectFalse(f.ImplicitDirs)
	ExpectFalse(f.ReuseTokenFromUrl)
	ExpectFalse(f.DebugFuseErrors)
	ExpectFalse(f.DebugFuse)
	ExpectFalse(f.DebugGCS)
	ExpectFalse(f.DebugHTTP)
	ExpectFalse(f.DebugInvariants)
	ExpectFalse(f.EnableNonexistentTypeCache)

	// --foo=true form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=true", n))
	}

	f = parseArgs(args)
	ExpectTrue(f.ImplicitDirs)
	ExpectTrue(f.ReuseTokenFromUrl)
	ExpectTrue(f.DebugFuseErrors)
	ExpectTrue(f.DebugFuse)
	ExpectTrue(f.DebugGCS)
	ExpectTrue(f.DebugHTTP)
	ExpectTrue(f.DebugInvariants)
	ExpectTrue(f.EnableNonexistentTypeCache)
}

func (t *FlagsTest) DecimalNumbers() {
	args := []string{
		"--uid=17",
		"--gid=19",
		"--limit-bytes-per-sec=123.4",
		"--limit-ops-per-sec=56.78",
		"--stat-cache-capacity=8192",
		"--max-idle-conns-per-host=100",
		"--max-conns-per-host=100",
	}

	f := parseArgs(args)
	ExpectEq(17, f.Uid)
	ExpectEq(19, f.Gid)
	ExpectEq(123.4, f.EgressBandwidthLimitBytesPerSecond)
	ExpectEq(56.78, f.OpRateLimitHz)
	ExpectEq(8192, f.StatCacheCapacity)
	ExpectEq(100, f.MaxIdleConnsPerHost)
	ExpectEq(100, f.MaxConnsPerHost)
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
		"--client-protocol=HTTP2",
	}

	f := parseArgs(args)
	ExpectEq("-asdf", f.KeyFile)
	ExpectEq("foobar", f.TempDir)
	ExpectEq("baz", f.OnlyDir)
	ExpectEq(mountpkg.HTTP2, f.ClientProtocol)
}

func (t *FlagsTest) Durations() {
	args := []string{
		"--stat-cache-ttl", "1m17s100ms",
		"--type-cache-ttl", "50s900ms",
		"--http-client-timeout", "800ms",
		"--max-retry-duration", "-1s",
		"--max-retry-sleep", "30s",
	}

	f := parseArgs(args)
	ExpectEq(time.Minute+17*time.Second+100*time.Millisecond, f.StatCacheTTL)
	ExpectEq(50*time.Second+900*time.Millisecond, f.TypeCacheTTL)
	ExpectEq(800*time.Millisecond, f.HttpClientTimeout)
	ExpectEq(-1*time.Second, f.MaxRetryDuration)
	ExpectEq(30*time.Second, f.MaxRetrySleep)
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

func (t *FlagsTest) TestResolvePathForTheFlagInContext() {
	app := newApp()
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	app.Action = func(appCtx *cli.Context) {
		err = resolvePathForTheFlagInContext("log-file", appCtx)
		AssertEq(nil, err)
		err = resolvePathForTheFlagInContext("key-file", appCtx)
		AssertEq(nil, err)
		err = resolvePathForTheFlagInContext("config-file", appCtx)
		AssertEq(nil, err)

		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("log-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("key-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--log-file=test.txt",
		"--key-file=test.txt", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	AssertEq(nil, err)
}

func (t *FlagsTest) TestResolvePathForTheFlagsInContext() {
	app := newApp()
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	app.Action = func(appCtx *cli.Context) {
		resolvePathForTheFlagsInContext(appCtx)

		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("log-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("key-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--log-file=test.txt",
		"--key-file=test.txt", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	AssertEq(nil, err)
}

func (t *FlagsTest) TestValidateFlagsForValidSequentialReadSizeAndHTTP1ClientProtocol() {
	flags := &flagStorage{
		SequentialReadSizeMb: 10,
		ClientProtocol:       mountpkg.ClientProtocol("http1"),
	}

	err := validateFlags(flags)

	AssertEq(nil, err)
}

func (t *FlagsTest) TestValidateFlagsForZeroSequentialReadSizeAndValidClientProtocol() {
	flags := &flagStorage{
		SequentialReadSizeMb: 0,
		ClientProtocol:       mountpkg.ClientProtocol("http2"),
	}

	err := validateFlags(flags)

	AssertNe(nil, err)
	AssertEq("SequentialReadSizeMb should be less than 1024", err.Error())
}

func (t *FlagsTest) TestValidateFlagsForSequentialReadSizeGreaterThan1024AndValidClientProtocol() {
	flags := &flagStorage{
		SequentialReadSizeMb: 2048,
		ClientProtocol:       mountpkg.ClientProtocol("http1"),
	}

	err := validateFlags(flags)

	AssertNe(nil, err)
	AssertEq("SequentialReadSizeMb should be less than 1024", err.Error())
}

func (t *FlagsTest) TestValidateFlagsForValidSequentialReadSizeAndInValidClientProtocol() {
	flags := &flagStorage{
		SequentialReadSizeMb: 10,
		ClientProtocol:       mountpkg.ClientProtocol("http4"),
	}

	err := validateFlags(flags)

	AssertEq("client protocol: http4 is not valid", err.Error())
}

func (t *FlagsTest) TestValidateFlagsForValidSequentialReadSizeAndHTTP2ClientProtocol() {
	flags := &flagStorage{
		SequentialReadSizeMb: 10,
		ClientProtocol:       mountpkg.ClientProtocol("http2"),
	}

	err := validateFlags(flags)

	AssertEq(nil, err)
}

func (t *FlagsTest) Test_resolveConfigFilePaths() {
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		FilePath: "~/test.txt",
	}
	mountConfig.CacheDir = "~/cache-dir"

	err := resolveConfigFilePaths(mountConfig)

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), mountConfig.LogConfig.FilePath)
	ExpectEq(filepath.Join(homeDir, "cache-dir"), mountConfig.CacheDir)
}

func (t *FlagsTest) Test_resolveConfigFilePaths_WithoutSettingPaths() {
	mountConfig := &config.MountConfig{}

	err := resolveConfigFilePaths(mountConfig)

	AssertEq(nil, err)
	ExpectEq("", mountConfig.LogConfig.FilePath)
	ExpectEq("", mountConfig.CacheDir)
}

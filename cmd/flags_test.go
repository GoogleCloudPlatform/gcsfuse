// Copyright 2024 Google LLC
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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	mountpkg "github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

func TestFlags(t *testing.T) { suite.Run(t, new(FlagsTest)) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FlagsTest struct {
	suite.Suite
}

func parseArgs(t *FlagsTest, args []string) (flags *flagStorage) {
	// Create a CLI app, and abuse it to snoop on the flags.
	app := newApp()
	var err error
	app.Action = func(appCtx *cli.Context) {
		flags, err = populateFlags(appCtx)
		assert.Equal(t.T(), nil, err)
	}

	// Simulate argv.
	fullArgs := append([]string{"some_app"}, args...)

	err = app.Run(fullArgs)
	assert.Equal(t.T(), nil, err)

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FlagsTest) TestDefaults() {
	f := parseArgs(t, []string{})

	// File system
	assert.NotEqual(t.T(), nil, f.MountOptions)
	assert.Equal(t.T(), 0, len(f.MountOptions), "Options: %v", f.MountOptions)

	assert.Equal(t.T(), os.FileMode(0755), f.DirMode)
	assert.Equal(t.T(), os.FileMode(0644), f.FileMode)
	assert.EqualValues(t.T(), -1, f.Uid)
	assert.EqualValues(t.T(), -1, f.Gid)
	assert.False(t.T(), f.ImplicitDirs)
	assert.True(t.T(), f.IgnoreInterrupts)
	assert.Equal(t.T(), config.DefaultKernelListCacheTtlSeconds, f.KernelListCacheTtlSeconds)

	// GCS
	assert.Equal(t.T(), "", f.KeyFile)
	assert.EqualValues(t.T(), -1, f.EgressBandwidthLimitBytesPerSecond)
	assert.EqualValues(t.T(), -1, f.OpRateLimitHz)
	assert.True(t.T(), f.ReuseTokenFromUrl)
	assert.Nil(t.T(), f.CustomEndpoint)
	assert.False(t.T(), f.AnonymousAccess)

	// Tuning
	assert.Equal(t.T(), mount.DefaultStatCacheCapacity, f.StatCacheCapacity)
	assert.Equal(t.T(), mount.DefaultStatOrTypeCacheTTL, f.StatCacheTTL)
	assert.Equal(t.T(), mount.DefaultStatOrTypeCacheTTL, f.TypeCacheTTL)
	assert.EqualValues(t.T(), 0, f.HttpClientTimeout)
	assert.Equal(t.T(), "", f.TempDir)
	assert.Equal(t.T(), config.DefaultMaxRetryAttempts, f.MaxRetryAttempts)
	assert.EqualValues(t.T(), 2, f.RetryMultiplier)
	assert.False(t.T(), f.EnableNonexistentTypeCache)
	assert.Equal(t.T(), 0, f.MaxConnsPerHost)

	// Debugging
	assert.False(t.T(), f.DebugFuse)
	assert.False(t.T(), f.DebugGCS)
	assert.False(t.T(), f.DebugInvariants)

	// Post-mount actions
	assert.Equal(t.T(), cfg.ExperimentalMetadataPrefetchOnMountDisabled, f.ExperimentalMetadataPrefetchOnMount)

	// Metrics
	assert.Equal(t.T(), 0, f.PrometheusPort)
}

func (t *FlagsTest) TestBools() {
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
		"ignore-interrupts",
		"anonymous-access",
	}

	var args []string
	var f *flagStorage

	// --foo form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s", n))
	}

	f = parseArgs(t, args)
	assert.True(t.T(), f.ImplicitDirs)
	assert.True(t.T(), f.ReuseTokenFromUrl)
	assert.True(t.T(), f.DebugFuse)
	assert.True(t.T(), f.DebugGCS)
	assert.True(t.T(), f.DebugInvariants)
	assert.True(t.T(), f.EnableNonexistentTypeCache)
	assert.True(t.T(), f.ExperimentalEnableJsonRead)
	assert.True(t.T(), f.IgnoreInterrupts)
	assert.True(t.T(), f.AnonymousAccess)

	// --foo=false form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=false", n))
	}

	f = parseArgs(t, args)
	assert.False(t.T(), f.ImplicitDirs)
	assert.False(t.T(), f.ReuseTokenFromUrl)
	assert.False(t.T(), f.DebugFuse)
	assert.False(t.T(), f.DebugGCS)
	assert.False(t.T(), f.DebugInvariants)
	assert.False(t.T(), f.EnableNonexistentTypeCache)

	// --foo=true form
	args = nil
	for _, n := range names {
		args = append(args, fmt.Sprintf("-%s=true", n))
	}

	f = parseArgs(t, args)
	assert.True(t.T(), f.ImplicitDirs)
	assert.True(t.T(), f.ReuseTokenFromUrl)
	assert.True(t.T(), f.DebugFuse)
	assert.True(t.T(), f.DebugGCS)
	assert.True(t.T(), f.DebugInvariants)
	assert.True(t.T(), f.EnableNonexistentTypeCache)
}

func (t *FlagsTest) TestDecimalNumbers() {
	args := []string{
		"--uid=17",
		"--gid=19",
		"--limit-bytes-per-sec=123.4",
		"--limit-ops-per-sec=56.78",
		"--stat-cache-capacity=8192",
		"--max-idle-conns-per-host=100",
		"--max-conns-per-host=100",
		"--kernel-list-cache-ttl-secs=234",
		"--max-retry-attempts=100",
	}

	f := parseArgs(t, args)
	assert.EqualValues(t.T(), 17, f.Uid)
	assert.EqualValues(t.T(), 19, f.Gid)
	assert.Equal(t.T(), 123.4, f.EgressBandwidthLimitBytesPerSecond)
	assert.Equal(t.T(), 56.78, f.OpRateLimitHz)
	assert.Equal(t.T(), 8192, f.StatCacheCapacity)
	assert.Equal(t.T(), 100, f.MaxIdleConnsPerHost)
	assert.Equal(t.T(), 100, f.MaxConnsPerHost)
	assert.EqualValues(t.T(), 234, f.KernelListCacheTtlSeconds)
	assert.EqualValues(t.T(), 100, f.MaxRetryAttempts)
}

func (t *FlagsTest) TestOctalNumbers() {
	args := []string{
		"--dir-mode=711",
		"--file-mode", "611",
	}

	f := parseArgs(t, args)
	assert.Equal(t.T(), os.FileMode(0711), f.DirMode)
	assert.Equal(t.T(), os.FileMode(0611), f.FileMode)
}

func (t *FlagsTest) TestStrings() {
	args := []string{
		"--key-file", "-asdf",
		"--temp-dir=foobar",
		"--only-dir=baz",
		"--client-protocol=HTTP2",
		"--experimental-metadata-prefetch-on-mount=async",
	}

	f := parseArgs(t, args)
	assert.Equal(t.T(), "-asdf", f.KeyFile)
	assert.Equal(t.T(), "foobar", f.TempDir)
	assert.Equal(t.T(), "baz", f.OnlyDir)
	assert.Equal(t.T(), mountpkg.HTTP2, f.ClientProtocol)
	assert.Equal(t.T(), cfg.ExperimentalMetadataPrefetchOnMountAsynchronous, f.ExperimentalMetadataPrefetchOnMount)
}

func (t *FlagsTest) TestDurations() {
	args := []string{
		"--stat-cache-ttl", "1m17s100ms",
		"--type-cache-ttl", "50s900ms",
		"--http-client-timeout", "800ms",
		"--max-retry-duration", "-1s",
		"--max-retry-sleep", "30s",
	}

	f := parseArgs(t, args)
	assert.Equal(t.T(), time.Minute+17*time.Second+100*time.Millisecond, f.StatCacheTTL)
	assert.Equal(t.T(), 50*time.Second+900*time.Millisecond, f.TypeCacheTTL)
	assert.Equal(t.T(), 800*time.Millisecond, f.HttpClientTimeout)
	assert.Equal(t.T(), 30*time.Second, f.MaxRetrySleep)
}

func (t *FlagsTest) TestSlice() {
	args := []string{
		"-o", "rw,nodev",
		"-o", "user=jacobsa,noauto",
	}

	f := parseArgs(t, args)

	sort.Strings(f.MountOptions)
	require.Equal(t.T(), 2, len(f.MountOptions))
	assert.Equal(t.T(), "rw,nodev", f.MountOptions[0])
	assert.Equal(t.T(), "user=jacobsa,noauto", f.MountOptions[1])
}

func (t *FlagsTest) TestResolvePathForTheFlagInContext() {
	app := newApp()
	currentWorkingDir, err := os.Getwd()
	assert.Equal(t.T(), nil, err)
	app.Action = func(appCtx *cli.Context) {
		err = resolvePathForTheFlagInContext("key-file", appCtx)
		assert.Equal(t.T(), nil, err)
		err = resolvePathForTheFlagInContext("config-file", appCtx)
		assert.Equal(t.T(), nil, err)

		assert.Equal(t.T(), filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("key-file"))
		assert.Equal(t.T(), filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--key-file=test.txt", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	assert.Equal(t.T(), nil, err)
}

func (t *FlagsTest) TestResolvePathForTheFlagsInContext() {
	app := newApp()
	currentWorkingDir, err := os.Getwd()
	assert.Equal(t.T(), nil, err)
	app.Action = func(appCtx *cli.Context) {
		resolvePathForTheFlagsInContext(appCtx)

		assert.Equal(t.T(), filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	assert.Equal(t.T(), nil, err)
}

func (t *FlagsTest) Test_resolveConfigFilePaths() {
	mountConfig := &config.MountConfig{}
	mountConfig.CacheDir = "~/cache-dir"

	err := resolveConfigFilePaths(mountConfig)

	assert.Equal(t.T(), nil, err)
	homeDir, err := os.UserHomeDir()
	assert.Equal(t.T(), nil, err)
	assert.EqualValues(t.T(), filepath.Join(homeDir, "cache-dir"), mountConfig.CacheDir)
}

func (t *FlagsTest) Test_resolveConfigFilePaths_WithoutSettingPaths() {
	mountConfig := &config.MountConfig{}

	err := resolveConfigFilePaths(mountConfig)

	assert.Equal(t.T(), nil, err)
	assert.EqualValues(t.T(), "", mountConfig.CacheDir)
}

func (t *FlagsTest) Test_KernelListCacheTtlSecs() {
	args := []string{
		"--kernel-list-cache-ttl-secs=-1",
	}

	f := parseArgs(t, args)

	assert.Equal(t.T(), int64(-1), f.KernelListCacheTtlSeconds)
}

func (t *FlagsTest) Test_KernelListCacheTtlSecs_MaxValid() {
	args := []string{
		"--kernel-list-cache-ttl-secs=9223372036",
	}

	f := parseArgs(t, args)

	assert.Equal(t.T(), int64(9223372036), f.KernelListCacheTtlSeconds)
}

func (t *FlagsTest) Test_PrometheusPort() {
	args := []string{
		"--prometheus-port=8080",
	}

	f := parseArgs(t, args)

	assert.Equal(t.T(), 8080, f.PrometheusPort)
}

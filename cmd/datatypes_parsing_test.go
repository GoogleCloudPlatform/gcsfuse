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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIFlagPassing(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		args   []string
		testFn func(t *testing.T, c *cfg.Config)
	}{
		{
			name:   "int1",
			args:   []string{"--log-rotate-backup-file-count=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(0), c.Logging.LogRotate.BackupFileCount) },
		},
		{
			name:   "int2",
			args:   []string{"--log-rotate-backup-file-count", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(0), c.Logging.LogRotate.BackupFileCount) },
		},
		{
			name:   "int3",
			args:   []string{"--max-conns-per-host=-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(-1), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int4",
			args:   []string{"--max-conns-per-host", "-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(-1), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int5",
			args:   []string{"--max-conns-per-host=12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(12), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int6",
			args:   []string{"--max-conns-per-host", "12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(12), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int7",
			args:   []string{"-log-rotate-backup-file-count=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(0), c.Logging.LogRotate.BackupFileCount) },
		},
		{
			name:   "int8",
			args:   []string{"-log-rotate-backup-file-count", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(0), c.Logging.LogRotate.BackupFileCount) },
		},
		{
			name:   "int9",
			args:   []string{"-max-conns-per-host=-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(-1), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int10",
			args:   []string{"-max-conns-per-host", "-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(-1), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int11",
			args:   []string{"-max-conns-per-host=12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(12), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "int12",
			args:   []string{"-max-conns-per-host", "12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, int64(12), c.GcsConnection.MaxConnsPerHost) },
		},
		{
			name:   "float1",
			args:   []string{"--limit-bytes-per-sec=-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float2",
			args:   []string{"--limit-bytes-per-sec", "-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float3",
			args:   []string{"--limit-bytes-per-sec=12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float4",
			args:   []string{"--limit-bytes-per-sec", "12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float5",
			args:   []string{"--limit-bytes-per-sec", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float6",
			args:   []string{"--limit-bytes-per-sec=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float7",
			args:   []string{"--limit-bytes-per-sec=12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float8",
			args:   []string{"--limit-bytes-per-sec", "12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float9",
			args:   []string{"--limit-bytes-per-sec=-12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float10",
			args:   []string{"--limit-bytes-per-sec", "-12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float11",
			args:   []string{"-limit-bytes-per-sec=-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float12",
			args:   []string{"-limit-bytes-per-sec", "-1"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float13",
			args:   []string{"-limit-bytes-per-sec=12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float14",
			args:   []string{"-limit-bytes-per-sec", "12"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float15",
			args:   []string{"-limit-bytes-per-sec", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float16",
			args:   []string{"-limit-bytes-per-sec=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0.0, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float17",
			args:   []string{"-limit-bytes-per-sec=12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float18",
			args:   []string{"-limit-bytes-per-sec", "12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float19",
			args:   []string{"-limit-bytes-per-sec=-12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "float20",
			args:   []string{"-limit-bytes-per-sec", "-12.5"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -12.5, c.GcsConnection.LimitBytesPerSec) },
		},
		{
			name:   "string1",
			args:   []string{"--app-name", ""},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "", c.AppName) },
		},
		{
			name:   "string2",
			args:   []string{"--app-name=", ""},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "", c.AppName) },
		},
		{
			name:   "string3",
			args:   []string{"--app-name", "Hello"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "Hello", c.AppName) },
		},
		{
			name:   "string4",
			args:   []string{"--app-name=Hello"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "Hello", c.AppName) },
		},
		{
			name:   "string5",
			args:   []string{"-app-name", ""},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "", c.AppName) },
		},
		{
			name:   "string6",
			args:   []string{"-app-name=", ""},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "", c.AppName) },
		},
		{
			name:   "string7",
			args:   []string{"-app-name", "Hello"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "Hello", c.AppName) },
		},
		{
			name:   "string8",
			args:   []string{"-app-name=Hello"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, "Hello", c.AppName) },
		},
		{
			name:   "bool1",
			args:   []string{"--foreground"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, true, c.Foreground) },
		},
		{
			name:   "bool2",
			args:   []string{"--foreground=true"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, true, c.Foreground) },
		},
		{
			name:   "bool3",
			args:   []string{},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, false, c.Foreground) },
		},
		{
			name:   "bool4",
			args:   []string{"--foreground=false"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, false, c.Foreground) },
		},
		{
			name:   "bool5",
			args:   []string{"-foreground"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, true, c.Foreground) },
		},
		{
			name:   "bool6",
			args:   []string{"-foreground=true"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, true, c.Foreground) },
		},
		{
			name:   "bool7",
			args:   []string{"-foreground=false"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, false, c.Foreground) },
		},
		{
			name:   "duration1",
			args:   []string{"--http-client-timeout=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration2",
			args:   []string{"--http-client-timeout", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration3",
			args:   []string{"--http-client-timeout", "0s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration4",
			args:   []string{"--http-client-timeout=0s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration5",
			args:   []string{"--http-client-timeout=1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration6",
			args:   []string{"--http-client-timeout", "1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration7",
			args:   []string{"--http-client-timeout=-1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration8",
			args:   []string{"--http-client-timeout", "-1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name: "duration9",
			args: []string{"--http-client-timeout=1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, time.Hour+2*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration10",
			args: []string{"--http-client-timeout", "1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, time.Hour+2*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration11",
			args: []string{"--http-client-timeout=-1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -(time.Hour + 2*time.Second), c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration12",
			args: []string{"--http-client-timeout", "-1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -(time.Hour + 2*time.Second), c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name:   "duration13",
			args:   []string{"-http-client-timeout=0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration14",
			args:   []string{"-http-client-timeout", "0"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration15",
			args:   []string{"-http-client-timeout", "0s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration16",
			args:   []string{"-http-client-timeout=0s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration17",
			args:   []string{"-http-client-timeout=1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration18",
			args:   []string{"-http-client-timeout", "1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration19",
			args:   []string{"-http-client-timeout=-1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name:   "duration20",
			args:   []string{"-http-client-timeout", "-1s"},
			testFn: func(t *testing.T, c *cfg.Config) { assert.Equal(t, -1*time.Second, c.GcsConnection.HttpClientTimeout) },
		},
		{
			name: "duration21",
			args: []string{"-http-client-timeout=1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, time.Hour+2*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration22",
			args: []string{"-http-client-timeout", "1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, time.Hour+2*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration23",
			args: []string{"-http-client-timeout=-1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -(time.Hour + 2*time.Second), c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration24",
			args: []string{"-http-client-timeout", "-1h2s"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -(time.Hour + 2*time.Second), c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "octal1",
			args: []string{"--file-mode", "677"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Octal(0677), c.FileSystem.FileMode)
			},
		},
		{
			name: "octal2",
			args: []string{"--file-mode=677"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Octal(0677), c.FileSystem.FileMode)
			},
		},
		{
			name: "octal3",
			args: []string{"-file-mode", "677"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Octal(0677), c.FileSystem.FileMode)
			},
		},
		{
			name: "octal4",
			args: []string{"-file-mode=677"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Octal(0677), c.FileSystem.FileMode)
			},
		},
		{
			name: "stringslice1",
			args: []string{"-o", "a"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice2",
			args: []string{"-o", "a", "-o", "b"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice3",
			args: []string{"-o", "a", "-o", "b,c"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b", "c"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice4",
			args: []string{"--o", "a"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice5",
			args: []string{"--o", "a", "--o", "b"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice6",
			args: []string{"--o", "a", "--o", "b,c"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b", "c"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "stringslice7",
			args: []string{"--o", "a", "-o", "b,c"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b", "c"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "logseverity1",
			args: []string{"--log-severity", "TRACE"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "logseverity2",
			args: []string{"--log-severity", "trace"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "logseverity3",
			args: []string{"-log-severity", "TRACE"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "logseverity4",
			args: []string{"-log-severity", "trace"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "logseverity5",
			args: []string{},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("INFO"), c.Logging.Severity)
			},
		},
		{
			name: "protocol1",
			args: []string{},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http1"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "protocol2",
			args: []string{"--client-protocol=HTTP2"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http2"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "protocol3",
			args: []string{"--client-protocol=http2"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http2"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "protocol4",
			args: []string{"--client-protocol", "HTTP2"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http2"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "protocol5",
			args: []string{"--client-protocol", "http2"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http2"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "resolvedpath1",
			args: []string{"--cache-dir", "/home"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.ResolvedPath("/home"), c.CacheDir)
			},
		},
		{
			name: "resolvedpath2",
			args: []string{"--cache-dir=/home"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.ResolvedPath("/home"), c.CacheDir)
			},
		},
		{
			name: "resolvedpath3",
			args: []string{},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.ResolvedPath(""), c.CacheDir)
			},
		},
		{
			name: "profile1",
			args: []string{"--profile", cfg.ProfileAIMLTraining},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.ProfileAIMLTraining, c.Profile)
			},
		},
		{
			name: "socketaddress1",
			args: []string{"--experimental-local-socket-address", "127.0.0.1"},
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, "127.0.0.1", c.GcsConnection.ExperimentalLocalSocketAddress)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var c *cfg.Config
			command, err := newRootCmd(func(mountInfo *mountInfo, _, _ string) error {
				c = mountInfo.config
				return nil
			})
			require.NoError(t, err)
			cmdArgs := append([]string{"gcsfuse"}, tc.args...)
			cmdArgs = append(cmdArgs, "a")
			command.SetArgs(convertToPosixArgs(cmdArgs, command))

			require.NoError(t, command.Execute())

			tc.testFn(t, c)
		})
	}
}

func TestConfigPassing(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		file   string
		testFn func(t *testing.T, c *cfg.Config)
	}{
		{
			name: "int1",
			file: "int1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, int64(0), c.Logging.LogRotate.BackupFileCount)
			},
		},
		{
			name: "int2",
			file: "int2.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, int64(-1), c.GcsConnection.MaxConnsPerHost)
			},
		},
		{
			name: "int3",
			file: "int3.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, int64(12), c.GcsConnection.MaxConnsPerHost)
			},
		},
		{
			name: "float1",
			file: "float1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -1.0, c.GcsConnection.LimitBytesPerSec)
			},
		},
		{
			name: "float2",
			file: "float2.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, 1.0, c.GcsConnection.LimitBytesPerSec)
			},
		},
		{
			name: "float3",
			file: "float3.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, 0.0, c.GcsConnection.LimitBytesPerSec)
			},
		},
		{
			name: "float4",
			file: "float4.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, 12.5, c.GcsConnection.LimitBytesPerSec)
			},
		},
		{
			name: "float5",
			file: "float5.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -12.5, c.GcsConnection.LimitBytesPerSec)
			},
		},
		{
			name: "bool1",
			file: "bool1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, true, c.Foreground)
			},
		},
		{
			name: "bool2",
			file: "bool2.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, false, c.Foreground)
			},
		},
		{
			name: "string1",
			file: "string1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, "abc", c.AppName)
			},
		},
		{
			name: "duration1",
			file: "duration1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, 0*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration2",
			file: "duration2.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, 15*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration3",
			file: "duration3.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, time.Hour+15*time.Second, c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "duration4",
			file: "duration4.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, -(time.Hour + 15*time.Second), c.GcsConnection.HttpClientTimeout)
			},
		},
		{
			name: "octal1",
			file: "octal1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Octal(0765), c.FileSystem.FileMode)
			},
		},
		{
			name: "stringslice1",
			file: "stringslice1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, []string{"a", "b", "c"}, c.FileSystem.FuseOptions)
			},
		},
		{
			name: "logseverity1",
			file: "logseverity1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "logseverity2",
			file: "logseverity2.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.LogSeverity("TRACE"), c.Logging.Severity)
			},
		},
		{
			name: "protocol1",
			file: "protocol1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.Protocol("http2"), c.GcsConnection.ClientProtocol)
			},
		},
		{
			name: "resolvedpath1",
			file: "resolvedpath1.yaml",
			testFn: func(t *testing.T, c *cfg.Config) {
				assert.Equal(t, cfg.ResolvedPath("/home"), c.CacheDir)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var c *cfg.Config
			command, err := newRootCmd(func(mountInfo *mountInfo, _, _ string) error {
				c = mountInfo.config
				return nil
			})
			require.NoError(t, err)
			cmdArgs := append([]string{"gcsfuse", fmt.Sprintf("--config-file=testdata/%s", tc.file)}, "a")
			command.SetArgs(convertToPosixArgs(cmdArgs, command))

			require.NoError(t, command.Execute())

			tc.testFn(t, c)
		})
	}
}

func TestPredefinedFlagThrowNoError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "help",
			args: []string{"--help"},
		},
		{
			name: "help_single_hyphen",
			args: []string{"-help"},
		},
		{
			name: "help_shorthand",
			args: []string{"-h"},
		},
		{
			name: "help_shorthand_two_hyphens",
			args: []string{"--h"},
		},
		{
			name: "version",
			args: []string{"--version"},
		},
		{
			name: "version_single_hyphen",
			args: []string{"-version"},
		},
		{
			name: "version_shorthand",
			args: []string{"-v"},
		},
		{
			name: "version_shorthand_two_hyphens",
			args: []string{"--v"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			command, err := newRootCmd(func(mountInfo *mountInfo, _, _ string) error {
				return nil
			})
			require.NoError(t, err)
			cmdArgs := append([]string{"gcsfuse"}, tc.args...)
			command.SetArgs(convertToPosixArgs(cmdArgs, command))

			assert.NoError(t, command.Execute())
		})
	}
}

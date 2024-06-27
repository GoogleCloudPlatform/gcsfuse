// Copyright 2024 Google Inc. All Rights Reserved.
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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgParsing(t *testing.T) {
	testcases := []struct {
		name     string
		args     []string
		actualFn func(config cfg.Config) any
		expected any
	}{

		{
			name:     "Test flag: foreground parsing #0",
			args:     []string{"gcsfuse", "abc", "--foreground"},
			actualFn: func(config cfg.Config) any { return config.Foreground },
			expected: true,
		},

		{
			name:     "Test flag: foreground parsing #1",
			args:     []string{"gcsfuse", "abc", "--foreground=true"},
			actualFn: func(config cfg.Config) any { return config.Foreground },
			expected: true,
		},

		{
			name:     "Test flag: foreground parsing #2",
			args:     []string{"gcsfuse", "abc", "--foreground=false"},
			actualFn: func(config cfg.Config) any { return config.Foreground },
			expected: false,
		},

		{
			name:     "Test flag: uid parsing #0",
			args:     []string{"gcsfuse", "abc", "--uid=11"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Uid },
			expected: 11,
		},

		{
			name:     "Test flag: uid parsing #1",
			args:     []string{"gcsfuse", "abc", "--uid", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Uid },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: uid parsing #2",
			args:     []string{"gcsfuse", "abc", "--uid", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Uid },
			expected: int64(-123),
		},

		{
			name:     "Test flag: gid parsing #0",
			args:     []string{"gcsfuse", "abc", "--gid=11"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Gid },
			expected: 11,
		},

		{
			name:     "Test flag: gid parsing #1",
			args:     []string{"gcsfuse", "abc", "--gid", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Gid },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: gid parsing #2",
			args:     []string{"gcsfuse", "abc", "--gid", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.Gid },
			expected: int64(-123),
		},

		{
			name:     "Test flag: implicit-dirs parsing #0",
			args:     []string{"gcsfuse", "abc", "--implicit-dirs"},
			actualFn: func(config cfg.Config) any { return config.ImplicitDirs },
			expected: true,
		},

		{
			name:     "Test flag: implicit-dirs parsing #1",
			args:     []string{"gcsfuse", "abc", "--implicit-dirs=true"},
			actualFn: func(config cfg.Config) any { return config.ImplicitDirs },
			expected: true,
		},

		{
			name:     "Test flag: implicit-dirs parsing #2",
			args:     []string{"gcsfuse", "abc", "--implicit-dirs=false"},
			actualFn: func(config cfg.Config) any { return config.ImplicitDirs },
			expected: false,
		},

		{
			name:     "Test flag: rename-dir-limit parsing #0",
			args:     []string{"gcsfuse", "abc", "--rename-dir-limit=11"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.RenameDirLimit },
			expected: 11,
		},

		{
			name:     "Test flag: rename-dir-limit parsing #1",
			args:     []string{"gcsfuse", "abc", "--rename-dir-limit", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.RenameDirLimit },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: rename-dir-limit parsing #2",
			args:     []string{"gcsfuse", "abc", "--rename-dir-limit", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.RenameDirLimit },
			expected: int64(-123),
		},

		{
			name:     "Test flag: ignore-interrupts parsing #0",
			args:     []string{"gcsfuse", "abc", "--ignore-interrupts"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.IgnoreInterrupts },
			expected: true,
		},

		{
			name:     "Test flag: ignore-interrupts parsing #1",
			args:     []string{"gcsfuse", "abc", "--ignore-interrupts=true"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.IgnoreInterrupts },
			expected: true,
		},

		{
			name:     "Test flag: ignore-interrupts parsing #2",
			args:     []string{"gcsfuse", "abc", "--ignore-interrupts=false"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.IgnoreInterrupts },
			expected: false,
		},

		{
			name:     "Test flag: disable-parallel-dirops parsing #0",
			args:     []string{"gcsfuse", "abc", "--disable-parallel-dirops"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.DisableParallelDirops },
			expected: true,
		},

		{
			name:     "Test flag: disable-parallel-dirops parsing #1",
			args:     []string{"gcsfuse", "abc", "--disable-parallel-dirops=true"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.DisableParallelDirops },
			expected: true,
		},

		{
			name:     "Test flag: disable-parallel-dirops parsing #2",
			args:     []string{"gcsfuse", "abc", "--disable-parallel-dirops=false"},
			actualFn: func(config cfg.Config) any { return config.FileSystem.DisableParallelDirops },
			expected: false,
		},

		{
			name:     "Test flag: anonymous-access parsing #0",
			args:     []string{"gcsfuse", "abc", "--anonymous-access"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.AnonymousAccess },
			expected: true,
		},

		{
			name:     "Test flag: anonymous-access parsing #1",
			args:     []string{"gcsfuse", "abc", "--anonymous-access=true"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.AnonymousAccess },
			expected: true,
		},

		{
			name:     "Test flag: anonymous-access parsing #2",
			args:     []string{"gcsfuse", "abc", "--anonymous-access=false"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.AnonymousAccess },
			expected: false,
		},

		{
			name:     "Test flag: reuse-token-from-url parsing #0",
			args:     []string{"gcsfuse", "abc", "--reuse-token-from-url"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.ReuseTokenFromUrl },
			expected: true,
		},

		{
			name:     "Test flag: reuse-token-from-url parsing #1",
			args:     []string{"gcsfuse", "abc", "--reuse-token-from-url=true"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.ReuseTokenFromUrl },
			expected: true,
		},

		{
			name:     "Test flag: reuse-token-from-url parsing #2",
			args:     []string{"gcsfuse", "abc", "--reuse-token-from-url=false"},
			actualFn: func(config cfg.Config) any { return config.GcsAuth.ReuseTokenFromUrl },
			expected: false,
		},

		{
			name:     "Test flag: limit-bytes-per-sec parsing #0",
			args:     []string{"gcsfuse", "abc", "--limit-bytes-per-sec=2.5"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.LimitBytesPerSec },
			expected: 2.5,
		},

		{
			name:     "Test flag: limit-bytes-per-sec parsing #1",
			args:     []string{"gcsfuse", "abc", "--limit-bytes-per-sec", "3.5"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.LimitBytesPerSec },
			expected: 3.5,
		},

		{
			name:     "Test flag: limit-ops-per-sec parsing #0",
			args:     []string{"gcsfuse", "abc", "--limit-ops-per-sec=2.5"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.LimitOpsPerSec },
			expected: 2.5,
		},

		{
			name:     "Test flag: limit-ops-per-sec parsing #1",
			args:     []string{"gcsfuse", "abc", "--limit-ops-per-sec", "3.5"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.LimitOpsPerSec },
			expected: 3.5,
		},

		{
			name:     "Test flag: sequential-read-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--sequential-read-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.SequentialReadSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: sequential-read-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--sequential-read-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.SequentialReadSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: sequential-read-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--sequential-read-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.SequentialReadSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: max-retry-sleep parsing #0",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep", "2h45m"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: max-retry-sleep parsing #1",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep", "300ms"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: max-retry-sleep parsing #2",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep", "1h49m12s11ms"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond,
		},

		{
			name:     "Test flag: max-retry-sleep parsing #3",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep=2h45m"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: max-retry-sleep parsing #4",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep=300ms"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: max-retry-sleep parsing #5",
			args:     []string{"gcsfuse", "abc", "--max-retry-sleep=25h49m12s"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.MaxRetrySleep },
			expected: 25*time.Hour + 49*time.Minute + 12*time.Second,
		},

		{
			name:     "Test flag: stat-cache-max-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--stat-cache-max-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.StatCacheMaxSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: stat-cache-max-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--stat-cache-max-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.StatCacheMaxSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: stat-cache-max-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--stat-cache-max-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.StatCacheMaxSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: type-cache-max-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--type-cache-max-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TypeCacheMaxSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: type-cache-max-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--type-cache-max-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TypeCacheMaxSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: type-cache-max-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--type-cache-max-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TypeCacheMaxSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: metadata-cache-ttl parsing #0",
			args:     []string{"gcsfuse", "abc", "--metadata-cache-ttl=11"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TtlSecs },
			expected: 11,
		},

		{
			name:     "Test flag: metadata-cache-ttl parsing #1",
			args:     []string{"gcsfuse", "abc", "--metadata-cache-ttl", "3478923"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TtlSecs },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: metadata-cache-ttl parsing #2",
			args:     []string{"gcsfuse", "abc", "--metadata-cache-ttl", "-123"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.TtlSecs },
			expected: int64(-123),
		},

		{
			name:     "Test flag: stat-cache-capacity parsing #0",
			args:     []string{"gcsfuse", "abc", "--stat-cache-capacity=11"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheCapacity },
			expected: 11,
		},

		{
			name:     "Test flag: stat-cache-capacity parsing #1",
			args:     []string{"gcsfuse", "abc", "--stat-cache-capacity", "3478923"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheCapacity },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: stat-cache-capacity parsing #2",
			args:     []string{"gcsfuse", "abc", "--stat-cache-capacity", "-123"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheCapacity },
			expected: int64(-123),
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #0",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl", "2h45m"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #1",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl", "300ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #2",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl", "1h49m12s11ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond,
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #3",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl=2h45m"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #4",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl=300ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: stat-cache-ttl parsing #5",
			args:     []string{"gcsfuse", "abc", "--stat-cache-ttl=25h49m12s"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedStatCacheTtl },
			expected: 25*time.Hour + 49*time.Minute + 12*time.Second,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #0",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl", "2h45m"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #1",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl", "300ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #2",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl", "1h49m12s11ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #3",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl=2h45m"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #4",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl=300ms"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: type-cache-ttl parsing #5",
			args:     []string{"gcsfuse", "abc", "--type-cache-ttl=25h49m12s"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.DeprecatedTypeCacheTtl },
			expected: 25*time.Hour + 49*time.Minute + 12*time.Second,
		},

		{
			name:     "Test flag: kernel-list-cache-ttl-secs parsing #0",
			args:     []string{"gcsfuse", "abc", "--kernel-list-cache-ttl-secs=11"},
			actualFn: func(config cfg.Config) any { return config.List.KernelListCacheTtlSecs },
			expected: 11,
		},

		{
			name:     "Test flag: kernel-list-cache-ttl-secs parsing #1",
			args:     []string{"gcsfuse", "abc", "--kernel-list-cache-ttl-secs", "3478923"},
			actualFn: func(config cfg.Config) any { return config.List.KernelListCacheTtlSecs },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: kernel-list-cache-ttl-secs parsing #2",
			args:     []string{"gcsfuse", "abc", "--kernel-list-cache-ttl-secs", "-123"},
			actualFn: func(config cfg.Config) any { return config.List.KernelListCacheTtlSecs },
			expected: int64(-123),
		},

		{
			name:     "Test flag: http-client-timeout parsing #0",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout", "2h45m"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: http-client-timeout parsing #1",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout", "300ms"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: http-client-timeout parsing #2",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout", "1h49m12s11ms"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond,
		},

		{
			name:     "Test flag: http-client-timeout parsing #3",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout=2h45m"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: http-client-timeout parsing #4",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout=300ms"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: http-client-timeout parsing #5",
			args:     []string{"gcsfuse", "abc", "--http-client-timeout=25h49m12s"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.HttpClientTimeout },
			expected: 25*time.Hour + 49*time.Minute + 12*time.Second,
		},

		{
			name:     "Test flag: retry-multiplier parsing #0",
			args:     []string{"gcsfuse", "abc", "--retry-multiplier=2.5"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.Multiplier },
			expected: 2.5,
		},

		{
			name:     "Test flag: retry-multiplier parsing #1",
			args:     []string{"gcsfuse", "abc", "--retry-multiplier", "3.5"},
			actualFn: func(config cfg.Config) any { return config.GcsRetries.Multiplier },
			expected: 3.5,
		},

		{
			name:     "Test flag: max-conns-per-host parsing #0",
			args:     []string{"gcsfuse", "abc", "--max-conns-per-host=11"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxConnsPerHost },
			expected: 11,
		},

		{
			name:     "Test flag: max-conns-per-host parsing #1",
			args:     []string{"gcsfuse", "abc", "--max-conns-per-host", "3478923"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxConnsPerHost },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: max-conns-per-host parsing #2",
			args:     []string{"gcsfuse", "abc", "--max-conns-per-host", "-123"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxConnsPerHost },
			expected: int64(-123),
		},

		{
			name:     "Test flag: max-idle-conns-per-host parsing #0",
			args:     []string{"gcsfuse", "abc", "--max-idle-conns-per-host=11"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxIdleConnsPerHost },
			expected: 11,
		},

		{
			name:     "Test flag: max-idle-conns-per-host parsing #1",
			args:     []string{"gcsfuse", "abc", "--max-idle-conns-per-host", "3478923"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxIdleConnsPerHost },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: max-idle-conns-per-host parsing #2",
			args:     []string{"gcsfuse", "abc", "--max-idle-conns-per-host", "-123"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.MaxIdleConnsPerHost },
			expected: int64(-123),
		},

		{
			name:     "Test flag: enable-nonexistent-type-cache parsing #0",
			args:     []string{"gcsfuse", "abc", "--enable-nonexistent-type-cache"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.EnableNonexistentTypeCache },
			expected: true,
		},

		{
			name:     "Test flag: enable-nonexistent-type-cache parsing #1",
			args:     []string{"gcsfuse", "abc", "--enable-nonexistent-type-cache=true"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.EnableNonexistentTypeCache },
			expected: true,
		},

		{
			name:     "Test flag: enable-nonexistent-type-cache parsing #2",
			args:     []string{"gcsfuse", "abc", "--enable-nonexistent-type-cache=false"},
			actualFn: func(config cfg.Config) any { return config.MetadataCache.EnableNonexistentTypeCache },
			expected: false,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #0",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval", "2h45m"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #1",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval", "300ms"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #2",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval", "1h49m12s11ms"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 1*time.Hour + 49*time.Minute + 12*time.Second + 11*time.Millisecond,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #3",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval=2h45m"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 2*time.Hour + 45*time.Minute,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #4",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval=300ms"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 300 * time.Millisecond,
		},

		{
			name:     "Test flag: stackdriver-export-interval parsing #5",
			args:     []string{"gcsfuse", "abc", "--stackdriver-export-interval=25h49m12s"},
			actualFn: func(config cfg.Config) any { return config.Metrics.StackdriverExportInterval },
			expected: 25*time.Hour + 49*time.Minute + 12*time.Second,
		},

		{
			name:     "Test flag: experimental-enable-json-read parsing #0",
			args:     []string{"gcsfuse", "abc", "--experimental-enable-json-read"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.ExperimentalEnableJsonRead },
			expected: true,
		},

		{
			name:     "Test flag: experimental-enable-json-read parsing #1",
			args:     []string{"gcsfuse", "abc", "--experimental-enable-json-read=true"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.ExperimentalEnableJsonRead },
			expected: true,
		},

		{
			name:     "Test flag: experimental-enable-json-read parsing #2",
			args:     []string{"gcsfuse", "abc", "--experimental-enable-json-read=false"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.ExperimentalEnableJsonRead },
			expected: false,
		},

		{
			name:     "Test flag: debug_gcs parsing #0",
			args:     []string{"gcsfuse", "abc", "--debug_gcs"},
			actualFn: func(config cfg.Config) any { return config.Debug.Gcs },
			expected: true,
		},

		{
			name:     "Test flag: debug_gcs parsing #1",
			args:     []string{"gcsfuse", "abc", "--debug_gcs=true"},
			actualFn: func(config cfg.Config) any { return config.Debug.Gcs },
			expected: true,
		},

		{
			name:     "Test flag: debug_gcs parsing #2",
			args:     []string{"gcsfuse", "abc", "--debug_gcs=false"},
			actualFn: func(config cfg.Config) any { return config.Debug.Gcs },
			expected: false,
		},

		{
			name:     "Test flag: debug_invariants parsing #0",
			args:     []string{"gcsfuse", "abc", "--debug_invariants"},
			actualFn: func(config cfg.Config) any { return config.Debug.ExitOnInvariantViolation },
			expected: true,
		},

		{
			name:     "Test flag: debug_invariants parsing #1",
			args:     []string{"gcsfuse", "abc", "--debug_invariants=true"},
			actualFn: func(config cfg.Config) any { return config.Debug.ExitOnInvariantViolation },
			expected: true,
		},

		{
			name:     "Test flag: debug_invariants parsing #2",
			args:     []string{"gcsfuse", "abc", "--debug_invariants=false"},
			actualFn: func(config cfg.Config) any { return config.Debug.ExitOnInvariantViolation },
			expected: false,
		},

		{
			name:     "Test flag: debug_mutex parsing #0",
			args:     []string{"gcsfuse", "abc", "--debug_mutex"},
			actualFn: func(config cfg.Config) any { return config.Debug.LogMutex },
			expected: true,
		},

		{
			name:     "Test flag: debug_mutex parsing #1",
			args:     []string{"gcsfuse", "abc", "--debug_mutex=true"},
			actualFn: func(config cfg.Config) any { return config.Debug.LogMutex },
			expected: true,
		},

		{
			name:     "Test flag: debug_mutex parsing #2",
			args:     []string{"gcsfuse", "abc", "--debug_mutex=false"},
			actualFn: func(config cfg.Config) any { return config.Debug.LogMutex },
			expected: false,
		},

		{
			name:     "Test flag: create-empty-file parsing #0",
			args:     []string{"gcsfuse", "abc", "--create-empty-file"},
			actualFn: func(config cfg.Config) any { return config.Write.CreateEmptyFile },
			expected: true,
		},

		{
			name:     "Test flag: create-empty-file parsing #1",
			args:     []string{"gcsfuse", "abc", "--create-empty-file=true"},
			actualFn: func(config cfg.Config) any { return config.Write.CreateEmptyFile },
			expected: true,
		},

		{
			name:     "Test flag: create-empty-file parsing #2",
			args:     []string{"gcsfuse", "abc", "--create-empty-file=false"},
			actualFn: func(config cfg.Config) any { return config.Write.CreateEmptyFile },
			expected: false,
		},

		{
			name:     "Test flag: file-cache-max-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--file-cache-max-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: file-cache-max-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--file-cache-max-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: file-cache-max-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--file-cache-max-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: cache-file-for-range-read parsing #0",
			args:     []string{"gcsfuse", "abc", "--cache-file-for-range-read"},
			actualFn: func(config cfg.Config) any { return config.FileCache.CacheFileForRangeRead },
			expected: true,
		},

		{
			name:     "Test flag: cache-file-for-range-read parsing #1",
			args:     []string{"gcsfuse", "abc", "--cache-file-for-range-read=true"},
			actualFn: func(config cfg.Config) any { return config.FileCache.CacheFileForRangeRead },
			expected: true,
		},

		{
			name:     "Test flag: cache-file-for-range-read parsing #2",
			args:     []string{"gcsfuse", "abc", "--cache-file-for-range-read=false"},
			actualFn: func(config cfg.Config) any { return config.FileCache.CacheFileForRangeRead },
			expected: false,
		},

		{
			name:     "Test flag: enable-crc parsing #0",
			args:     []string{"gcsfuse", "abc", "--enable-crc"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableCrc },
			expected: true,
		},

		{
			name:     "Test flag: enable-crc parsing #1",
			args:     []string{"gcsfuse", "abc", "--enable-crc=true"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableCrc },
			expected: true,
		},

		{
			name:     "Test flag: enable-crc parsing #2",
			args:     []string{"gcsfuse", "abc", "--enable-crc=false"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableCrc },
			expected: false,
		},

		{
			name:     "Test flag: enable-parallel-downloads parsing #0",
			args:     []string{"gcsfuse", "abc", "--enable-parallel-downloads"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableParallelDownloads },
			expected: true,
		},

		{
			name:     "Test flag: enable-parallel-downloads parsing #1",
			args:     []string{"gcsfuse", "abc", "--enable-parallel-downloads=true"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableParallelDownloads },
			expected: true,
		},

		{
			name:     "Test flag: enable-parallel-downloads parsing #2",
			args:     []string{"gcsfuse", "abc", "--enable-parallel-downloads=false"},
			actualFn: func(config cfg.Config) any { return config.FileCache.EnableParallelDownloads },
			expected: false,
		},

		{
			name:     "Test flag: parallel-downloads-per-file parsing #0",
			args:     []string{"gcsfuse", "abc", "--parallel-downloads-per-file=11"},
			actualFn: func(config cfg.Config) any { return config.FileCache.ParallelDownloadsPerFile },
			expected: 11,
		},

		{
			name:     "Test flag: parallel-downloads-per-file parsing #1",
			args:     []string{"gcsfuse", "abc", "--parallel-downloads-per-file", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileCache.ParallelDownloadsPerFile },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: parallel-downloads-per-file parsing #2",
			args:     []string{"gcsfuse", "abc", "--parallel-downloads-per-file", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileCache.ParallelDownloadsPerFile },
			expected: int64(-123),
		},

		{
			name:     "Test flag: max-parallel-downloads parsing #0",
			args:     []string{"gcsfuse", "abc", "--max-parallel-downloads=11"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxParallelDownloads },
			expected: 11,
		},

		{
			name:     "Test flag: max-parallel-downloads parsing #1",
			args:     []string{"gcsfuse", "abc", "--max-parallel-downloads", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxParallelDownloads },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: max-parallel-downloads parsing #2",
			args:     []string{"gcsfuse", "abc", "--max-parallel-downloads", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileCache.MaxParallelDownloads },
			expected: int64(-123),
		},

		{
			name:     "Test flag: download-chunk-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--download-chunk-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.FileCache.DownloadChunkSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: download-chunk-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--download-chunk-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.FileCache.DownloadChunkSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: download-chunk-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--download-chunk-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.FileCache.DownloadChunkSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: enable-empty-managed-folders parsing #0",
			args:     []string{"gcsfuse", "abc", "--enable-empty-managed-folders"},
			actualFn: func(config cfg.Config) any { return config.List.EnableEmptyManagedFolders },
			expected: true,
		},

		{
			name:     "Test flag: enable-empty-managed-folders parsing #1",
			args:     []string{"gcsfuse", "abc", "--enable-empty-managed-folders=true"},
			actualFn: func(config cfg.Config) any { return config.List.EnableEmptyManagedFolders },
			expected: true,
		},

		{
			name:     "Test flag: enable-empty-managed-folders parsing #2",
			args:     []string{"gcsfuse", "abc", "--enable-empty-managed-folders=false"},
			actualFn: func(config cfg.Config) any { return config.List.EnableEmptyManagedFolders },
			expected: false,
		},

		{
			name:     "Test flag: experimental-grpc-conn-pool-size parsing #0",
			args:     []string{"gcsfuse", "abc", "--experimental-grpc-conn-pool-size=11"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.GrpcConnPoolSize },
			expected: 11,
		},

		{
			name:     "Test flag: experimental-grpc-conn-pool-size parsing #1",
			args:     []string{"gcsfuse", "abc", "--experimental-grpc-conn-pool-size", "3478923"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.GrpcConnPoolSize },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: experimental-grpc-conn-pool-size parsing #2",
			args:     []string{"gcsfuse", "abc", "--experimental-grpc-conn-pool-size", "-123"},
			actualFn: func(config cfg.Config) any { return config.GcsConnection.GrpcConnPoolSize },
			expected: int64(-123),
		},

		{
			name:     "Test flag: enable-hns parsing #0",
			args:     []string{"gcsfuse", "abc", "--enable-hns"},
			actualFn: func(config cfg.Config) any { return config.EnableHns },
			expected: true,
		},

		{
			name:     "Test flag: enable-hns parsing #1",
			args:     []string{"gcsfuse", "abc", "--enable-hns=true"},
			actualFn: func(config cfg.Config) any { return config.EnableHns },
			expected: true,
		},

		{
			name:     "Test flag: enable-hns parsing #2",
			args:     []string{"gcsfuse", "abc", "--enable-hns=false"},
			actualFn: func(config cfg.Config) any { return config.EnableHns },
			expected: false,
		},

		{
			name:     "Test flag: log-rotate-max-log-file-size-mb parsing #0",
			args:     []string{"gcsfuse", "abc", "--log-rotate-max-log-file-size-mb=11"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.MaxFileSizeMb },
			expected: 11,
		},

		{
			name:     "Test flag: log-rotate-max-log-file-size-mb parsing #1",
			args:     []string{"gcsfuse", "abc", "--log-rotate-max-log-file-size-mb", "3478923"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.MaxFileSizeMb },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: log-rotate-max-log-file-size-mb parsing #2",
			args:     []string{"gcsfuse", "abc", "--log-rotate-max-log-file-size-mb", "-123"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.MaxFileSizeMb },
			expected: int64(-123),
		},

		{
			name:     "Test flag: log-rotate-backup-file-count parsing #0",
			args:     []string{"gcsfuse", "abc", "--log-rotate-backup-file-count=11"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.BackupFileCount },
			expected: 11,
		},

		{
			name:     "Test flag: log-rotate-backup-file-count parsing #1",
			args:     []string{"gcsfuse", "abc", "--log-rotate-backup-file-count", "3478923"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.BackupFileCount },
			expected: int64(3478923),
		},

		{
			name:     "Test flag: log-rotate-backup-file-count parsing #2",
			args:     []string{"gcsfuse", "abc", "--log-rotate-backup-file-count", "-123"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.BackupFileCount },
			expected: int64(-123),
		},

		{
			name:     "Test flag: log-rotate-compress parsing #0",
			args:     []string{"gcsfuse", "abc", "--log-rotate-compress"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.Compress },
			expected: true,
		},

		{
			name:     "Test flag: log-rotate-compress parsing #1",
			args:     []string{"gcsfuse", "abc", "--log-rotate-compress=true"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.Compress },
			expected: true,
		},

		{
			name:     "Test flag: log-rotate-compress parsing #2",
			args:     []string{"gcsfuse", "abc", "--log-rotate-compress=false"},
			actualFn: func(config cfg.Config) any { return config.Logging.LogRotate.Compress },
			expected: false,
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var actual cfg.Config
			cmd, err := NewRootCmd(func(c cfg.Config) error {
				actual = c
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			if assert.Nil(t, cmd.Execute()) {
				assert.EqualValues(t, tc.expected, tc.actualFn(actual))
			}
		})
	}
}

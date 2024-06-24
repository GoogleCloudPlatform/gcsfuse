// Copyright 2023 Google Inc. All Rights Reserved.
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

package config

import (
	"fmt"
	"math"
	"time"
)

const (
	IgnoreInterruptsFlagName   = "ignore-interrupts"
	AnonymousAccess            = "anonymous-access"
	KernelListCacheTtlFlagName = "kernel-list-cache-ttl-secs"
	TtlInSecsInvalidValueError = "the value of ttl-secs can't be less than -1"
	TtlInSecsTooHighError      = "the value of ttl-secs is too high to be supported. Max is 9223372036"

	// MaxSupportedTtlInSeconds represents maximum multiple of seconds representable by time.Duration.
	MaxSupportedTtlInSeconds = math.MaxInt64 / int64(time.Second)
	MaxSupportedTtl          = time.Duration(MaxSupportedTtlInSeconds * int64(time.Second))
	MaxRetryAttempts         = "max-retry-attempts"
)

// OverrideWithLoggingFlags overwrites the configs with the flag values if the
// config values are empty.
func OverrideWithLoggingFlags(mountConfig *MountConfig, logFile string, logFormat string,
	debugFuse bool, debugGCS bool, debugMutex bool) {
	// If log file is not set in config file, override it with flag value.
	if mountConfig.LogConfig.FilePath == "" {
		mountConfig.LogConfig.FilePath = logFile
	}
	// If log format is not set in config file, override it with flag value.
	if mountConfig.LogConfig.Format == "" {
		mountConfig.LogConfig.Format = logFormat
	}
	// If debug_fuse, debug_gcsfuse or debug_mutex flag is set, override log
	// severity to TRACE.
	if debugFuse || debugGCS || debugMutex {
		mountConfig.LogConfig.Severity = TRACE
	}
}

// cliContext is abstraction over the IsSet() method of cli.Context, Specially
// added to keep OverrideWithIgnoreInterruptsFlag method's unit test simple.
type cliContext interface {
	IsSet(string) bool
}

// OverrideWithIgnoreInterruptsFlag overwrites the ignore-interrupts config with
// the ignore-interrupts flag value if the flag is set.
func OverrideWithIgnoreInterruptsFlag(c cliContext, mountConfig *MountConfig, ignoreInterruptsFlag bool) {
	// If the ignore-interrupts flag is set, give it priority over the value in config file.
	if c.IsSet(IgnoreInterruptsFlagName) {
		mountConfig.FileSystemConfig.IgnoreInterrupts = ignoreInterruptsFlag
	}
}

// OverrideWithAnonymousAccessFlag overwrites the anonymous-access config with
// the anonymous-access flag value if the flag is set.
func OverrideWithAnonymousAccessFlag(c cliContext, mountConfig *MountConfig, anonymousAccess bool) {
	// If the  anonymous-access flag is set, give it priority over the value in config file.
	if c.IsSet(AnonymousAccess) {
		mountConfig.GCSAuth.AnonymousAccess = anonymousAccess
	}
}

// OverrideWithKernelListCacheTtlFlag overwrites the kernel-list-cache-ttl-secs config
// with the kernel-list-cache-ttl-secs cli-flag value if the cli-flag is set by user.
func OverrideWithKernelListCacheTtlFlag(c cliContext, mountConfig *MountConfig, ttl int64) {
	if c.IsSet(KernelListCacheTtlFlagName) {
		mountConfig.ListConfig.KernelListCacheTtlSeconds = ttl
	}
}

// OverrideWithMaxRetryAttemptFlag overwrites the max-retry-attempts config with
// the max-retry-attempts flag value if the flag is set.
func OverrideWithMaxRetryAttemptFlag(c cliContext, mountConfig *MountConfig, retries int64) {
	if c.IsSet(MaxRetryAttempts) {
		mountConfig.GCSRetries.MaxRetryAttempts = retries
	}
}

func IsFileCacheEnabled(mountConfig *MountConfig) bool {
	return mountConfig.FileCacheConfig.MaxSizeMB != 0 && string(mountConfig.CacheDir) != ""
}

// IsTtlInSecsValid return nil error if ttlInSecs is valid.
func IsTtlInSecsValid(ttlInSecs int64) error {
	if ttlInSecs < -1 {
		return fmt.Errorf(TtlInSecsInvalidValueError)
	}

	if ttlInSecs > MaxSupportedTtlInSeconds {
		return fmt.Errorf(TtlInSecsTooHighError)
	}

	return nil
}

func ListCacheTtlSecsToDuration(secs int64) time.Duration {
	err := IsTtlInSecsValid(secs)
	if err != nil {
		panic(fmt.Sprintf("invalid argument: %d, %v", secs, err))
	}

	if secs == -1 {
		return MaxSupportedTtl
	}

	return time.Duration(secs * int64(time.Second))
}

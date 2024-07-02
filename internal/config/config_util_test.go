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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type flags struct {
	LogFile          string
	LogFormat        string
	DebugFuse        bool
	DebugGCS         bool
	DebugMutex       bool
	IgnoreInterrupts bool
	AnonymousAccess  bool
}
type ConfigTest struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConfigTest) TestIsFileCacheEnabled() {
	mountConfig := &MountConfig{
		CacheDir: "/tmp/folder/",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	assert.True(t.T(), IsFileCacheEnabled(mountConfig))

	mountConfig1 := &MountConfig{}
	assert.False(t.T(), IsFileCacheEnabled(mountConfig1))

	mountConfig2 := &MountConfig{
		CacheDir: "",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	assert.False(t.T(), IsFileCacheEnabled(mountConfig2))

	mountConfig3 := &MountConfig{
		CacheDir: "//tmp//folder//",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: 0,
		},
	}
	assert.False(t.T(), IsFileCacheEnabled(mountConfig3))
}

type TestCliContext struct {
	isSet bool
}

func (s *TestCliContext) IsSet(flag string) bool {
	return s.isSet
}

func TestOverrideWithIgnoreInterruptsFlag(t *testing.T) {
	var overrideWithIgnoreInterruptsFlagTests = []struct {
		testName                   string
		ignoreInterruptConfigValue bool
		isFlagSet                  bool
		ignoreInterruptFlagValue   bool
		expectedIgnoreInterrupt    bool
	}{
		{"ignore-interrupts config true and flag not set", true, false, false, true},
		{"ignore-interrupts config false and flag not set", false, false, false, false},
		{"ignore-interrupts config false and ignore-interrupts flag false", false, true, false, false},
		{"ignore-interrupts config false and ignore-interrupts flag true", false, true, true, true},
		{"ignore-interrupts config true and ignore-interrupts flag false", true, true, false, false},
		{"ignore-interrupts config true and ignore-interrupts flag true", true, true, true, true},
	}

	for _, tt := range overrideWithIgnoreInterruptsFlagTests {
		t.Run(tt.testName, func(t *testing.T) {
			testContext := &TestCliContext{isSet: tt.isFlagSet}
			mountConfig := &MountConfig{FileSystemConfig: FileSystemConfig{IgnoreInterrupts: tt.ignoreInterruptConfigValue}}

			OverrideWithIgnoreInterruptsFlag(testContext, mountConfig, tt.ignoreInterruptFlagValue)

			assert.Equal(t, tt.expectedIgnoreInterrupt, mountConfig.FileSystemConfig.IgnoreInterrupts)
		})
	}
}

func TestOverrideWithAnonymousAccessFlag(t *testing.T) {
	var overrideWithAnonymousAccessFlagTests = []struct {
		testName                   string
		anonymousAccessConfigValue bool
		isFlagSet                  bool
		anonymousAccessFlagValue   bool
		expectedAnonymousAccess    bool
	}{
		{"anonymous-access config true and flag not set", true, false, false, true},
		{"anonymous-access config false and flag not set", false, false, false, false},
		{"anonymous-access config false and anonymous-access flag false", false, true, false, false},
		{"anonymous-access config false and anonymous-access flag true", false, true, true, true},
		{"anonymous-access config true and anonymous-access flag false", true, true, false, false},
		{"anonymous-access config true and anonymous-access flag true", true, true, true, true},
	}

	for _, tt := range overrideWithAnonymousAccessFlagTests {
		t.Run(tt.testName, func(t *testing.T) {
			testContext := &TestCliContext{isSet: tt.isFlagSet}
			mountConfig := &MountConfig{GCSAuth: GCSAuth{AnonymousAccess: tt.anonymousAccessConfigValue}}

			OverrideWithAnonymousAccessFlag(testContext, mountConfig, tt.anonymousAccessFlagValue)

			assert.Equal(t, tt.expectedAnonymousAccess, mountConfig.GCSAuth.AnonymousAccess)
		})
	}
}

func Test_OverrideWithKernelListCacheTtlFlag(t *testing.T) {
	var testCases = []struct {
		configValue   int64
		flagValue     int64
		isFlagSet     bool
		expectedValue int64
	}{
		{34, -1, true, -1},
		{34, -1, false, 34},
		{0, 435, true, 435},
		{0, 0, false, 0},
		{0, 1, true, 1},
		{9223372036, -1, false, 9223372036}, // MaxSupportedTtlInSeconds
		{5, -6, true, -6},
		{9223372037, 5, false, 9223372037}, // MaxSupportedTtlInSeconds + 1
	}

	for index, tt := range testCases {
		t.Run(fmt.Sprintf("Test case: %d", index), func(t *testing.T) {
			testContext := &TestCliContext{isSet: tt.isFlagSet}
			mountConfig := &MountConfig{FileSystemConfig: FileSystemConfig{KernelListCacheTtlSeconds: tt.configValue}}

			OverrideWithKernelListCacheTtlFlag(testContext, mountConfig, tt.flagValue)

			assert.Equal(t, tt.expectedValue, mountConfig.FileSystemConfig.KernelListCacheTtlSeconds)
		})
	}
}

func Test_IsTtlInSecsValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		ttlInSecs   int64
		expectedErr error
	}{
		{"Negative", -5, fmt.Errorf(TtlInSecsInvalidValueError)},
		{"Valid negative", -1, nil},
		{"Positive", 8, nil},
		{"Unsupported Large positive", 9223372037, fmt.Errorf(TtlInSecsTooHighError)},
		{"Zero", 0, nil},
		{"Valid upper limit", 9223372036, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedErr, IsTtlInSecsValid(tt.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration(t *testing.T) {
	var testCases = []struct {
		testName         string
		ttlInSecs        int64
		expectedDuration time.Duration
	}{
		{"-1", -1, MaxSupportedTtl},
		{"0", 0, time.Duration(0)},
		{"Max supported positive", 9223372036, MaxSupportedTtl},
		{"Positive", 1, time.Second},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedDuration, ListCacheTtlSecsToDuration(tt.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration_InvalidCall(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// Calling with invalid argument to trigger panic.
	ListCacheTtlSecsToDuration(-3)
}

func Test_DefaultMaxParallelDownloads(t *testing.T) {
	assert.GreaterOrEqual(t, DefaultMaxParallelDownloads(), 16)
}

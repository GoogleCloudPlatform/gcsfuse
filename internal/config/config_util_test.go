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
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestConfig(t *testing.T) { RunTests(t) }

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
}
type ConfigTest struct {
}

func init() { RegisterTestSuite(&ConfigTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConfigTest) TestOverrideLoggingFlags_WithNonEmptyLogConfigs() {
	f := &flags{
		LogFile:    "a.txt",
		LogFormat:  "json",
		DebugFuse:  true,
		DebugGCS:   false,
		DebugMutex: false,
	}
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		Severity: ERROR,
		FilePath: "/tmp/hello.txt",
		Format:   "text",
	}
	mountConfig.WriteConfig = WriteConfig{
		CreateEmptyFile: true,
	}

	OverrideWithLoggingFlags(mountConfig, f.LogFile, f.LogFormat, f.DebugFuse, f.DebugGCS, f.DebugMutex)

	AssertEq(mountConfig.LogConfig.Format, "text")
	AssertEq(mountConfig.LogConfig.FilePath, "/tmp/hello.txt")
	AssertEq(mountConfig.LogConfig.Severity, TRACE)
}

func (t *ConfigTest) TestOverrideLoggingFlags_WithEmptyLogConfigs() {
	f := &flags{
		LogFile:   "a.txt",
		LogFormat: "json",
	}
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		Severity: INFO,
		FilePath: "",
		Format:   "",
	}
	mountConfig.WriteConfig = WriteConfig{
		CreateEmptyFile: true,
	}

	OverrideWithLoggingFlags(mountConfig, f.LogFile, f.LogFormat, f.DebugFuse, f.DebugGCS, f.DebugMutex)

	AssertEq(mountConfig.LogConfig.Format, "json")
	AssertEq(mountConfig.LogConfig.FilePath, "a.txt")
	AssertEq(mountConfig.LogConfig.Severity, INFO)
}

func (t *ConfigTest) TestIsFileCacheEnabled() {
	mountConfig := &MountConfig{
		CacheDir: "/tmp/folder/",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	AssertEq(IsFileCacheEnabled(mountConfig), true)

	mountConfig1 := &MountConfig{}
	AssertEq(IsFileCacheEnabled(mountConfig1), false)

	mountConfig2 := &MountConfig{
		CacheDir: "",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: -1,
		},
	}
	AssertEq(IsFileCacheEnabled(mountConfig2), false)

	mountConfig3 := &MountConfig{
		CacheDir: "//tmp//folder//",
		FileCacheConfig: FileCacheConfig{
			MaxSizeMB: 0,
		},
	}
	AssertEq(IsFileCacheEnabled(mountConfig3), false)
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

			AssertEq(tt.expectedIgnoreInterrupt, mountConfig.FileSystemConfig.IgnoreInterrupts)
		})
	}
}

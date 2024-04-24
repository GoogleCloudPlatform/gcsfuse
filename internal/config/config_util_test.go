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
	"log"
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

func (t *ConfigTest) TestOverrideWithIgnoreInterruptsFlag() {
	var overrideWithIgnoreInterruptsFlagTests = []struct {
		testName                       string
		fileSystemConfig               FileSystemConfig
		testFlags                      flags
		expectedIgnoreInterruptsConfig bool
	}{
		{"file system config empty and flag empty", FileSystemConfig{}, flags{}, false},
		{"file system config empty and ignore-interrupts flag false", FileSystemConfig{}, flags{IgnoreInterrupts: false}, false},
		{"file system config empty and ignore-interrupts flag false", FileSystemConfig{}, flags{IgnoreInterrupts: true}, true},
		{"ignore-interrupts config true and flag empty", FileSystemConfig{IgnoreInterrupts: true}, flags{}, true},
		{"ignore-interrupts config false and flag empty", FileSystemConfig{IgnoreInterrupts: false}, flags{}, false},
		{"ignore-interrupts config false and ignore-interrupts flag false", FileSystemConfig{IgnoreInterrupts: false}, flags{IgnoreInterrupts: false}, false},
		{"ignore-interrupts config false and ignore-interrupts flag true", FileSystemConfig{IgnoreInterrupts: false}, flags{IgnoreInterrupts: true}, true},
		{"ignore-interrupts config true and ignore-interrupts flag false", FileSystemConfig{IgnoreInterrupts: true}, flags{IgnoreInterrupts: false}, true},
		{"ignore-interrupts config true and ignore-interrupts flag true", FileSystemConfig{IgnoreInterrupts: true}, flags{IgnoreInterrupts: true}, true},
	}

	for _, tt := range overrideWithIgnoreInterruptsFlagTests {
		log.Print("Running:" + tt.testName)
		mountConfig := &MountConfig{}
		mountConfig.FileSystemConfig = tt.fileSystemConfig
		OverrideWithIgnoreInterruptsFlag(mountConfig, tt.testFlags.IgnoreInterrupts)
		AssertEq(mountConfig.FileSystemConfig.IgnoreInterrupts, tt.expectedIgnoreInterruptsConfig)
	}
}

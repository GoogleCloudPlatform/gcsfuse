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
	LogFile    string
	LogFormat  string
	DebugFuse  bool
	DebugGCS   bool
	DebugMutex bool
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
		File:     "/tmp/hello.txt",
		Format:   "text",
	}
	mountConfig.WriteConfig = WriteConfig{
		CreateEmptyFile: true,
	}

	OverrideWithLoggingFlags(mountConfig, f.LogFile, f.LogFormat, f.DebugFuse, f.DebugGCS, f.DebugMutex)

	AssertEq(mountConfig.LogConfig.Format, "text")
	AssertEq(mountConfig.LogConfig.File, "/tmp/hello.txt")
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
		File:     "",
		Format:   "",
	}
	mountConfig.WriteConfig = WriteConfig{
		CreateEmptyFile: true,
	}

	OverrideWithLoggingFlags(mountConfig, f.LogFile, f.LogFormat, f.DebugFuse, f.DebugGCS, f.DebugMutex)

	AssertEq(mountConfig.LogConfig.Format, "json")
	AssertEq(mountConfig.LogConfig.File, "a.txt")
	AssertEq(mountConfig.LogConfig.Severity, INFO)
}

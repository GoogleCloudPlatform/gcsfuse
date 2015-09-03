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

package integration_test

import (
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/wiring"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestGcsfuse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type GcsfuseTest struct {
	gcsfusePath string
}

var _ SetUpInterface = &GcsfuseTest{}

func init() { RegisterTestSuite(&GcsfuseTest{}) }

func (t *GcsfuseTest) SetUp(_ *TestInfo) {
	t.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseTest) BadUsage() {
	testCases := []struct {
		args           []string
		expectedOutput string
	}{
		// Too few args
		0: {
			[]string{wiring.FakeBucket},
			"exactly two arguments",
		},

		// Too many args
		1: {
			[]string{wiring.FakeBucket, "a", "b"},
			"exactly two arguments",
		},

		// Unknown flag
		2: {
			[]string{"--tweak_frobnicator", wiring.FakeBucket, "a"},
			"not defined.*tweak_frobnicator",
		},
	}

	// Run each test case.
	for i, tc := range testCases {
		cmd := exec.Command(t.gcsfusePath)
		cmd.Args = append(cmd.Args, tc.args...)

		output, err := cmd.CombinedOutput()
		ExpectThat(err, Error(HasSubstr("exit status")), "case %d", i)
		ExpectThat(string(output), MatchesRegexp(tc.expectedOutput), "case %d", i)
	}
}

func (t *GcsfuseTest) ErrorOpeningBucket() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) ReadOnlyMode() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) ReadWriteMode() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) FileAndDirModeFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) UidAndGidFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) ImplicitDirs() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) VersionFlags() {
	AssertTrue(false, "TODO")
}

func (t *GcsfuseTest) HelpFlags() {
	AssertTrue(false, "TODO")
}

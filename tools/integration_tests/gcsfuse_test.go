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
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestGcsfuse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type GcsfuseTest struct {
}

func init() { RegisterTestSuite(&GcsfuseTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *GcsfuseTest) BadUsage() {
	AssertTrue(false, "TODO")
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

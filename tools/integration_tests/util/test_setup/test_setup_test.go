// Copyright 2024 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test_setup_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	. "github.com/jacobsa/ogletest"
)

type testStructure struct {
	setupCtr, teardownCtr, test1, test2 int
}

func (t *testStructure) Setup(*testing.T) {
	t.setupCtr++
}
func (t *testStructure) TestExample1(*testing.T) {
	t.test1++
}

func (t *testStructure) TestExample2(*testing.T) {
	t.test2++
}

func (t *testStructure) Teardown(*testing.T) {
	t.teardownCtr++
}

func TestRunTests(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	testStruct := &testStructure{}

	test_setup.RunTests(t, testStruct)

	AssertEq(testStruct.setupCtr, 2)
	AssertEq(testStruct.test1, 1)
	AssertEq(testStruct.test2, 1)
	AssertEq(testStruct.teardownCtr, 2)
}

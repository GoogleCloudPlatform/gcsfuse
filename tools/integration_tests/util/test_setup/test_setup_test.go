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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
	. "github.com/jacobsa/ogletest"
)

type testStructure struct {
	setup, test1, test2, teardown bool
}

func (t *testStructure) Setup(*testing.T) {
	t.setup = true
}
func (t *testStructure) TestExample1(*testing.T) {
	t.test1 = true
}

func (t *testStructure) TestExample2(*testing.T) {
	t.test2 = true
}

func (t *testStructure) Teardown(*testing.T) {
	t.teardown = true
}

func TestRunSubTests(t *testing.T) {
	testStruct := &testStructure{}

	test_setup.RunSubTests(t, testStruct)

	AssertTrue(testStruct.setup)
	AssertTrue(testStruct.test1)
	AssertTrue(testStruct.test2)
	AssertTrue(testStruct.teardown)
}

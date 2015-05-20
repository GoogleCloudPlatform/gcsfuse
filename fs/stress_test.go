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

package fs_test

import (
	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Run the supplied function for each name, with parallelism. Fail the test if
// an error is returned for any name.
func forEachName(names []string, f func(context.Context, string) error) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Stress testing
////////////////////////////////////////////////////////////////////////

type StressTest struct {
	fsTest
}

func init() { RegisterTestSuite(&StressTest{}) }

func (t *StressTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

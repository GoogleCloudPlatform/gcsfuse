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
	"fmt"
	"io/ioutil"
	"path"
	"runtime"

	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Run the supplied function for each name, with parallelism.
func forEachName(names []string, f func(context.Context, string)) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Stress testing
////////////////////////////////////////////////////////////////////////

type StressTest struct {
	fsTest
}

func init() { RegisterTestSuite(&StressTest{}) }

func (t *StressTest) CreateAndReadManyFilesInParallel() {
	const numFiles = 1024

	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Choose a bunch of file names.
	var names []string
	for i := 0; i < numFiles; i++ {
		names = append(names, fmt.Sprintf("%d", i))
	}

	// Create a file for each name with concurrent workers.
	forEachName(
		names,
		func(ctx context.Context, n string) {
			err := ioutil.WriteFile(path.Join(t.Dir, n), []byte(n), 0400)
			AssertEq(nil, err)
		})

	// Read each back.
	forEachName(
		names,
		func(ctx context.Context, n string) {
			contents, err := ioutil.ReadFile(path.Join(t.Dir, n))
			AssertEq(nil, err)
			AssertEq(n, string(contents))
		})
}

func (t *StressTest) LinkAndUnlinkFileManyTimesInParallel() {
	AssertFalse(true, "TODO")
}

func (t *StressTest) TruncateFileManyTimesInParallel() {
	AssertFalse(true, "TODO")
}

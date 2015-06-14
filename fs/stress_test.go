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
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Run the supplied function for each name, with parallelism.
func forEachName(
	ctx context.Context,
	names []string,
	f func(string) error) (err error) {
	const parallelism = 8

	// Fill a channel.
	c := make(chan string, len(names))
	for _, n := range names {
		c <- n
	}
	close(c)

	// Run workers.
	b := syncutil.NewBundle(ctx)
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			for n := range c {
				err = f(n)
				if err != nil {
					return
				}
			}

			return
		})
	}

	err = b.Join()
	return
}

////////////////////////////////////////////////////////////////////////
// Stress testing
////////////////////////////////////////////////////////////////////////

type StressTest struct {
	fsTest
}

func init() { ogletest.RegisterTestSuite(&StressTest{}) }

func (s *StressTest) CreateAndReadManyFilesInParallel(t *ogletest.T) {
	var err error

	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Exercise lease revocation logic.
	numFiles := 2 * s.serverCfg.TempDirLimitNumFiles

	// Choose a bunch of file names.
	var names []string
	for i := 0; i < numFiles; i++ {
		names = append(names, fmt.Sprintf("%d", i))
	}

	// Create a file for each name with concurrent workers.
	err = forEachName(
		t.Ctx,
		names,
		func(n string) (err error) {
			err = ioutil.WriteFile(path.Join(s.Dir, n), []byte(n), 0400)
			if err != nil {
				err = fmt.Errorf("WriteFile(%q): %v", path.Join(s.Dir, n), err)
				return
			}

			return
		})

	t.AssertEq(nil, err)

	// Read each back.
	err = forEachName(
		t.Ctx,
		names,
		func(n string) (err error) {
			contents, err := ioutil.ReadFile(path.Join(s.Dir, n))
			if err != nil {
				err = fmt.Errorf("ReadFile(%q): %v", path.Join(s.Dir, n), err)
				return
			}

			if string(contents) != n {
				err = fmt.Errorf("Unexpected contents for name %q: %q", n, contents)
				return
			}

			return
		})

	t.AssertEq(nil, err)
}

func (s *StressTest) TruncateFileManyTimesInParallel(t *ogletest.T) {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Create a file.
	f, err := os.Create(path.Join(s.Dir, "foo"))
	t.AssertEq(nil, err)
	defer f.Close()

	// Set up a function that repeatedly truncates the file to random lengths,
	// writing the final size to a channel.
	worker := func(finalSize chan<- int64) (err error) {
		const desiredDuration = 500 * time.Millisecond

		var size int64
		startTime := time.Now()
		for time.Since(startTime) < desiredDuration {
			for i := 0; i < 10; i++ {
				size = rand.Int63n(1 << 14)
				err = f.Truncate(size)
				if err != nil {
					err = fmt.Errorf("Truncate(%d): %v", size, err)
					return
				}
			}
		}

		finalSize <- size
		return
	}

	// Run several workers.
	const numWorkers = 16
	finalSizes := make(chan int64, numWorkers)

	b := syncutil.NewBundle(t.Ctx)
	for i := 0; i < numWorkers; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = worker(finalSizes)
			return
		})
	}

	err = b.Join()
	t.AssertEq(nil, err)

	close(finalSizes)

	// The final size should be consistent.
	fi, err := f.Stat()
	t.AssertEq(nil, err)

	var found = false
	for s := range finalSizes {
		if s == fi.Size() {
			found = true
			break
		}
	}

	t.ExpectTrue(found, "Unexpected size: %d", fi.Size())
}

func (s *StressTest) CreateInParallel_NoTruncate(t *ogletest.T) {
	fusetesting.RunCreateInParallelTest_NoTruncate(t, s.Dir)
}

func (s *StressTest) CreateInParallel_Truncate(t *ogletest.T) {
	fusetesting.RunCreateInParallelTest_Truncate(t, s.Dir)
}

func (s *StressTest) CreateInParallel_Exclusive(t *ogletest.T) {
	fusetesting.RunCreateInParallelTest_Exclusive(t, s.Dir)
}

func (s *StressTest) MkdirInParallel(t *ogletest.T) {
	fusetesting.RunMkdirInParallelTest(t, s.Dir)
}

func (s *StressTest) SymlinkInParallel(t *ogletest.T) {
	fusetesting.RunSymlinkInParallelTest(t, s.Dir)
}

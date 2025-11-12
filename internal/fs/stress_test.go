// Copyright 2015 Google LLC
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
	"math/rand"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Run the supplied function for each name, with parallelism. Return an error
// if any invocation does.
func forEachName(names []string, f func(string) error) (err error) {
	const parallelism = 8

	// Fill a channel.
	c := make(chan string, len(names))
	for _, n := range names {
		c <- n
	}
	close(c)

	// Run workers.
	firstErr := make(chan error, 1)

	var wg sync.WaitGroup
	for range parallelism {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range c {
				err := f(n)
				if err != nil {
					select {
					case firstErr <- err:
					default:
					}
				}
			}
		}()
	}

	wg.Wait()

	// Read the first error, if any.
	close(firstErr)
	err, _ = <-firstErr

	return
}

////////////////////////////////////////////////////////////////////////
// Stress testing
////////////////////////////////////////////////////////////////////////

type StressTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&StressTest{})
}

func (t *StressTest) CreateAndReadManyFilesInParallel() {
	var err error

	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Choose a bunch of file names.
	const numFiles = 32

	var names []string
	for i := range numFiles {
		names = append(names, fmt.Sprintf("%d", i))
	}

	// Create a file for each name with concurrent workers.
	err = forEachName(
		names,
		func(n string) (err error) {
			err = os.WriteFile(path.Join(mntDir, n), []byte(n), 0400)
			return
		})

	AssertEq(nil, err)

	// Read each back.
	err = forEachName(
		names,
		func(n string) (err error) {
			contents, err := os.ReadFile(path.Join(mntDir, n))
			if err != nil {
				err = fmt.Errorf("ReadFile: %w", err)
				return
			}

			if string(contents) != n {
				err = fmt.Errorf("Contents mismatch: %q vs. %q", contents, n)
				return
			}

			return
		})

	AssertEq(nil, err)
}

func (t *StressTest) TruncateFileManyTimesInParallel() {
	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Create a file.
	f, err := os.Create(path.Join(mntDir, "foo"))
	AssertEq(nil, err)
	defer f.Close()

	// Set up a function that repeatedly truncates the file to random lengths,
	// writing the final size to a channel.
	worker := func(finalSize chan<- int64) (err error) {
		const desiredDuration = 500 * time.Millisecond

		var size int64
		startTime := time.Now()
		for time.Since(startTime) < desiredDuration {
			for range 10 {
				size = rand.Int63n(1 << 14)
				err = f.Truncate(size)
				if err != nil {
					return
				}
			}
		}

		finalSize <- size
		return
	}

	// Run several workers.
	group := new(errgroup.Group)

	const numWorkers = 16
	finalSizes := make(chan int64, numWorkers)

	for range numWorkers {
		group.Go(func() (err error) {
			err = worker(finalSizes)
			return
		})
	}

	err = group.Wait()
	AssertEq(nil, err)

	close(finalSizes)

	// The final size should be consistent.
	fi, err := f.Stat()
	AssertEq(nil, err)

	var found = false
	for s := range finalSizes {
		if s == fi.Size() {
			found = true
			break
		}
	}

	ExpectTrue(found, "Unexpected size: %d", fi.Size())
}

func (t *StressTest) CreateInParallel_NoTruncate() {
	fusetesting.RunCreateInParallelTest_NoTruncate(ctx, mntDir)
}

func (t *StressTest) CreateInParallel_Truncate() {
	fusetesting.RunCreateInParallelTest_Truncate(ctx, mntDir)
}

func (t *StressTest) CreateInParallel_Exclusive() {
	fusetesting.RunCreateInParallelTest_Exclusive(ctx, mntDir)
}

func (t *StressTest) MkdirInParallel() {
	fusetesting.RunMkdirInParallelTest(ctx, mntDir)
}

func (t *StressTest) SymlinkInParallel() {
	fusetesting.RunSymlinkInParallelTest(ctx, mntDir)
}

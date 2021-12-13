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
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/fuse/fusetesting"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
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
	for i := 0; i < parallelism; i++ {
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

func init() { RegisterTestSuite(&StressTest{}) }

func (t *StressTest) CreateAndReadManyFilesInParallel() {
	var err error

	// Ensure that we get parallelism for this test.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Choose a bunch of file names.
	const numFiles = 32

	var names []string
	for i := 0; i < numFiles; i++ {
		names = append(names, fmt.Sprintf("%d", i))
	}

	// Create a file for each name with concurrent workers.
	err = forEachName(
		names,
		func(n string) (err error) {
			err = ioutil.WriteFile(path.Join(t.Dir, n), []byte(n), 0400)
			return
		})

	AssertEq(nil, err)

	// Read each back.
	err = forEachName(
		names,
		func(n string) (err error) {
			contents, err := ioutil.ReadFile(path.Join(t.Dir, n))
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
	f, err := os.Create(path.Join(t.Dir, "foo"))
	AssertEq(nil, err)
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
					return
				}
			}
		}

		finalSize <- size
		return
	}

	// Run several workers.
	b := syncutil.NewBundle(t.ctx)

	const numWorkers = 16
	finalSizes := make(chan int64, numWorkers)

	for i := 0; i < numWorkers; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = worker(finalSizes)
			return
		})
	}

	err = b.Join()
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
	fusetesting.RunCreateInParallelTest_NoTruncate(t.ctx, t.Dir)
}

func (t *StressTest) CreateInParallel_Truncate() {
	fusetesting.RunCreateInParallelTest_Truncate(t.ctx, t.Dir)
}

func (t *StressTest) CreateInParallel_Exclusive() {
	fusetesting.RunCreateInParallelTest_Exclusive(t.ctx, t.Dir)
}

func (t *StressTest) MkdirInParallel() {
	fusetesting.RunMkdirInParallelTest(t.ctx, t.Dir)
}

func (t *StressTest) SymlinkInParallel() {
	fusetesting.RunSymlinkInParallelTest(t.ctx, t.Dir)
}

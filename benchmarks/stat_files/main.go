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

// Create and open a bunch of files, then measure the performance of repeatedly
// statting them.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/benchmarks/internal/format"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/syncutil"
)

var fDir = flag.String("dir", "", "Directory within which to create the files.")
var fNumFiles = flag.Int("num_files", 4, "Number of files to create.")
var fDuration = flag.Duration("duration", 10*time.Second, "How long to run.")

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func closeAll(files []*os.File) {
	for _, f := range files {
		f.Close()
	}
}

func createFiles(
	dir string,
	numFiles int) (files []*os.File, err error) {
	b := syncutil.NewBundle(context.Background())

	// Create files in parallel, and write them to a channel.
	const parallelism = 128

	var counter uint64
	fileChan := make(chan *os.File)
	var wg sync.WaitGroup

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		b.Add(func(ctx context.Context) (err error) {
			defer wg.Done()
			for {
				// Should we create another?
				count := atomic.AddUint64(&counter, 1)
				if count > uint64(numFiles) {
					return
				}

				// Create it.
				var f *os.File
				f, err = fsutil.AnonymousFile(dir)
				if err != nil {
					err = fmt.Errorf("AnonymousFile: %w", err)
					return
				}

				// Write it to the channel.
				select {
				case fileChan <- f:
				case <-ctx.Done():
					err = ctx.Err()
					return
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(fileChan)
	}()

	// Accumulate into the slice.
	b.Add(func(ctx context.Context) (err error) {
		for f := range fileChan {
			files = append(files, f)
		}

		return
	})

	err = b.Join()
	if err != nil {
		closeAll(files)
		files = nil
	}

	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

func run() (err error) {
	if *fDir == "" {
		err = errors.New("You must set --dir.")
		return
	}

	if *fNumFiles <= 0 {
		err = fmt.Errorf("Invalid setting for --num_files: %d", *fNumFiles)
		return
	}

	// Create the temporary files.
	log.Printf("Creating %d temporary files...", *fNumFiles)

	files, err := createFiles(*fDir, *fNumFiles)
	if err != nil {
		err = fmt.Errorf("ListBackups: %w", err)
		return
	}

	defer closeAll(files)

	// Repeatedly stat the files.
	log.Println("Measuring...")

	var statCount int64
	start := time.Now()

	for ; time.Since(start) < *fDuration; statCount++ {
		_, err = files[statCount%int64(len(files))].Stat()
		if err != nil {
			err = fmt.Errorf("Stat: %w", err)
			return
		}
	}

	d := time.Since(start)

	// Report.
	seconds := float64(d) / float64(time.Second)
	statsPerSec := float64(statCount) / seconds

	fmt.Printf(
		"Statted %d times in %v (%s)\n",
		statCount,
		d,
		format.Hertz(statsPerSec))

	fmt.Println()
	return
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

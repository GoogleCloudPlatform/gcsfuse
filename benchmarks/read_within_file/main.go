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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/benchmarks/internal/format"
)

var fFile = flag.String("file", "", "Path to file to read.")
var fRandom = flag.Bool("random", false, "Read randomly? Otherwise sequentially.")
var fDuration = flag.Duration("duration", 10*time.Second, "How long to run.")
var fReadSize = flag.Int("read_size", 1<<20, "Size of each call to read(2).")

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readRandom(
	r io.ReaderAt,
	fileSize int64,
	readSize int,
	desiredDuration time.Duration) (err error) {
	// Make sure the logic below for choosing offsets works.
	if fileSize < int64(readSize) {
		err = fmt.Errorf(
			"File size of %d bytes not large enough for reads of %d bytes",
			fileSize,
			readSize)
		return
	}

	buf := make([]byte, readSize)

	start := time.Now()
	var readCount int64
	var bytesRead int64
	for time.Since(start) < desiredDuration {
		// Choose a random offset at which to read.
		off := rand.Int63n(fileSize - int64(readSize))

		// Read, ignoring io.EOF which io.ReaderAt is allowed to return for reads
		// that abut the end of the file.
		var n int
		n, err = r.ReadAt(buf, off)

		switch {
		case err == io.EOF && n == readSize:
			err = nil

		case err != nil:
			err = fmt.Errorf("ReadAt: %w", err)
			return
		}

		readCount++
		bytesRead += int64(n)
	}

	d := time.Since(start)

	// Report.
	seconds := float64(d) / float64(time.Second)
	readsPerSec := float64(readCount) / seconds

	fmt.Printf(
		"Read %d times (%s) in %v (%.1f Hz)\n",
		readCount,
		format.Bytes(float64(bytesRead)),
		d,
		readsPerSec)

	fmt.Println()

	return
}

func readSequential(
	r io.ReadSeeker,
	readSize int,
	desiredDuration time.Duration) (err error) {
	buf := make([]byte, readSize)
	start := time.Now()

	var readCount int64
	var bytesRead int64
	for time.Since(start) < desiredDuration {
		var n int
		n, err = r.Read(buf)

		switch {
		case err == io.EOF:
			_, err = r.Seek(0, 0)
			if err != nil {
				err = fmt.Errorf("Seek: %w", err)
				return
			}

		case err != nil:
			err = fmt.Errorf("Read: %w", err)
		}

		bytesRead += int64(n)
		readCount++
	}

	d := time.Since(start)

	// Report.
	seconds := float64(d) / float64(time.Second)
	readsPerSec := float64(readCount) / seconds
	bytesPerSec := float64(bytesRead) / seconds

	fmt.Printf(
		"Read %d times (%s) in %v (%.1f Hz, %s/s)\n",
		readCount,
		format.Bytes(float64(bytesRead)),
		d,
		readsPerSec,
		format.Bytes(bytesPerSec))

	fmt.Println()
	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

func run() (err error) {
	if *fFile == "" {
		err = errors.New("You must set --file.")
		return
	}

	// Open the file for reading.
	f, err := os.Open(*fFile)
	if err != nil {
		return
	}

	// Find its size.
	size, err := f.Seek(0, 2)
	if err != nil {
		err = fmt.Errorf("Seek: %w", err)
		return
	}

	log.Printf("%s has size %s.", f.Name(), format.Bytes(float64(size)))

	// Perform reads.
	if *fRandom {
		err = readRandom(f, size, *fReadSize, *fDuration)
		if err != nil {
			err = fmt.Errorf("readRandom: %w", err)
			return
		}
	} else {
		err = readSequential(f, *fReadSize, *fDuration)
		if err != nil {
			err = fmt.Errorf("readSequential: %w", err)
			return
		}
	}

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

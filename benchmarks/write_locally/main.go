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

// Create a file, truncate it up to a particular size, then measure the
// throughput of repeatedly overwriting its contents without closing it each
// time. This is intended to measure the CPU efficiency of the file system
// rather than GCS throughput.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/benchmarks/internal/format"
)

var fDir = flag.String("dir", "", "Directory within which to write the file.")
var fDuration = flag.Duration("duration", 10*time.Second, "How long to run.")
var fFileSize = flag.Int64("file_size", 1<<26, "Size of file to use.")
var fWriteSize = flag.Int64("write_size", 1<<20, "Size of each call to write(2).")

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

func run() (err error) {
	if *fDir == "" {
		err = errors.New("You must set --dir.")
		return
	}

	// Create a temporary file.
	log.Printf("Creating a temporary file in %s.", *fDir)

	f, err := os.CreateTemp(*fDir, "write_locally")
	if err != nil {
		err = fmt.Errorf("TempFile: %w", err)
		return
	}

	path := f.Name()

	// Make sure we clean it up later.
	defer func() {
		log.Printf("Truncating and closing %s.", path)
		_ = f.Truncate(0)
		_ = f.Close()

		log.Printf("Deleting %s.", path)
		_ = os.Remove(path)
	}()

	// Extend to the initial size.
	log.Printf("Truncating to %d bytes.", *fFileSize)

	err = f.Truncate(*fFileSize)
	if err != nil {
		err = fmt.Errorf("Truncate: %w", err)
		return
	}

	// Repeatedly overwrite the file with zeroes.
	log.Println("Measuring...")

	var bytesWritten int64
	var writeCount int64

	buf := make([]byte, *fWriteSize)
	start := time.Now()

	for time.Since(start) < *fDuration {
		// Seek to the beginning.
		_, err = f.Seek(0, 0)
		if err != nil {
			err = fmt.Errorf("Seek: %w", err)
			return
		}

		// Overwrite.
		var n int64
		for n < *fFileSize && time.Since(start) < *fDuration {
			var tmp int
			tmp, err = f.Write(buf)
			if err != nil {
				err = fmt.Errorf("Write: %w", err)
				return
			}

			n += int64(tmp)
			bytesWritten += int64(tmp)
			writeCount++
		}
	}

	d := time.Since(start)

	// Report.
	seconds := float64(d) / float64(time.Second)
	writesPerSec := float64(writeCount) / seconds
	bytesPerSec := float64(bytesWritten) / seconds

	fmt.Printf(
		"Wrote %d times (%s) in %v (%.1f Hz, %s/s)\n",
		writeCount,
		format.Bytes(float64(bytesWritten)),
		d,
		writesPerSec,
		format.Bytes(bytesPerSec))

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

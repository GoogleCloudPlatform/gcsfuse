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

// Create a file, write a bunch of data into it, then close it. Measure the
// time taken to write the data and to close. The former says something about
// the CPU-efficiency of gcsfuse, and the latter says something about GCS
// throughput (assuming gcsfuse CPU is not the bottleneck).
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
var fFileSize = flag.Int64("file_size", 1<<30, "How many bytes to write.")
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

	f, err := os.CreateTemp(*fDir, "write_to_gcs")
	if err != nil {
		err = fmt.Errorf("TempFile: %w", err)
		return
	}

	path := f.Name()

	// Make sure we clean it up later.
	defer func() {
		log.Printf("Deleting %s.", path)
		os.Remove(path)
	}()

	// Write the configured number of zeroes to the file, measuing the time
	// taken.
	log.Println("Writing...")

	buf := make([]byte, *fWriteSize)

	var bytesWritten int64
	start := time.Now()

	for bytesWritten < *fFileSize {
		// Decide how many bytes to write.
		toWrite := *fFileSize - bytesWritten
		if toWrite > *fWriteSize {
			toWrite = *fWriteSize
		}

		// Write them.
		_, err = f.Write(buf)
		if err != nil {
			err = fmt.Errorf("Write: %w", err)
			return
		}

		bytesWritten += toWrite
	}

	writeDuration := time.Since(start)

	// Close the file, measuring the time taken.
	log.Println("Flushing...")

	start = time.Now()
	err = f.Close()
	closeDuration := time.Since(start)

	if err != nil {
		err = fmt.Errorf("Close: %w", err)
		return
	}

	// Report.
	{
		seconds := float64(writeDuration) / float64(time.Second)
		bytesPerSec := float64(bytesWritten) / seconds

		fmt.Printf(
			"Wrote %s in %v (%s/s)\n",
			format.Bytes(float64(bytesWritten)),
			writeDuration,
			format.Bytes(bytesPerSec))
	}

	{
		seconds := float64(closeDuration) / float64(time.Second)
		bytesPerSec := float64(bytesWritten) / seconds

		fmt.Printf(
			"Flushed %s in %v (%s/s)\n",
			format.Bytes(float64(bytesWritten)),
			closeDuration,
			format.Bytes(bytesPerSec))
	}

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

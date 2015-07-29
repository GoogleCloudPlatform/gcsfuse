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
	"os"
	"time"
)

var fFile = flag.String("file", "", "Path to file to read.")
var fRandom = flag.Bool("random", false, "Read randomly? Otherwise sequentially.")
var fDuration = flag.Duration("duration", 10*time.Second, "How long to run.")
var fReadSize = flag.Int("read_size", 1<<20, "Size of each call to read(2).")

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

func readRandom(
	r io.ReaderAt,
	fileSize int64,
	readSize int,
	d time.Duration) (bytesRead int64, err error)

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
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	log.Printf("%s has size %d.", f.Name(), size)

	// Perform reads.
	start := time.Now()

	var bytesRead int64
	if *fRandom {
		bytesRead, err = readRandom(f, size, *fReadSize, *fDuration)
		if err != nil {
			err = fmt.Errorf("readRandom: %v", err)
			return
		}
	} else {
		panic("TODO")
	}

	d := time.Since(start)
	bandwidthBytesPerSec := float64(bytesRead) / (float64(d) / float64(time.Second))

	fmt.Printf("Read %d bytes in %v (%s/s)\n", bytesRead, d, bandwidthBytesPerSec)
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

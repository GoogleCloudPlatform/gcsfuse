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
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jacobsa/fuse"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func handleSIGINT(mountPoint string) {
	log.Println("Received SIGINT, attempting to unmount...")

	err := fuse.Unmount(mountPoint)
	if err != nil {
		log.Printf("Failed to unmount in response to SIGINT: %v", err)
	} else {
		log.Printf("Successfully unmounted in response to SIGINT.")
		return
	}
}

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up a custom usage function, then parse flags.
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [flags] bucket_name mount_point\n",
			os.Args[0])

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Help mode?
	if *fHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Extract positional arguments.
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	bucketName := args[0]
	mountPoint := args[1]

	// Run.
	err := run(bucketName, mountPoint)
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	log.Println("Successfully exiting.")
}

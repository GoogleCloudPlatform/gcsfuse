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
	"log"
	"os"
	"os/signal"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/gcloud/gcs"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func registerSIGINTHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			log.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				log.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				log.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromPath(path string) (ts oauth2.TokenSource, err error) {
	err = errors.New("TODO")
	return
}

func getConn(flags *flagStorage) (c gcs.Conn, err error) {
	// Create the oauth2 token source.
	var tokenSrc oauth2.TokenSource
	if flags.KeyFile != "" {
		tokenSrc, err = newTokenSourceFromPath(flags.KeyFile)
		if err != nil {
			err = fmt.Errorf("newTokenSourceFromPath: %v", err)
			return
		}
	} else {
		const scope = gcs.Scope_FullControl
		tokenSrc, err = google.DefaultTokenSource(context.Background(), scope)
		if err != nil {
			err = fmt.Errorf("DefaultTokenSource: %v", err)
			return
		}
	}

	// Create the connection.
	const userAgent = "gcsfuse/0.0"
	cfg := &gcs.ConnConfig{
		TokenSource: tokenSrc,
		UserAgent:   userAgent,
	}

	return gcs.NewConn(cfg)
}

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func main() {
	var err error
	flagSet := flag.CommandLine

	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up a custom usage function.
	flagSet.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [flags] bucket_name mount_point\n",
			os.Args[0])

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	// Populate and parse flags, exiting cleanly on a request for help.
	flagSet.Init("", flag.ContinueOnError)

	flags := populateFlagSet(flagSet)

	err = flagSet.Parse(os.Args[1:])
	switch {
	case err == flag.ErrHelp:
		return

	case err != nil:
		log.Fatalf("Parsing flags: %v", err)
	}

	// Extract positional arguments.
	if flagSet.NArg() != 2 {
		flagSet.Usage()
		os.Exit(1)
	}

	bucketName := flagSet.Arg(0)
	mountPoint := flagSet.Arg(1)

	// Grab the connection.
	conn, err := getConn(flags)
	if err != nil {
		log.Fatalf("getConn: %v", err)
	}

	// Mount the file system.
	mfs, err := mount(
		context.Background(),
		bucketName,
		mountPoint,
		flags,
		conn)

	if err != nil {
		log.Fatalf("Mounting file system: %v", err)
	}

	log.Println("File system has been successfully mounted.")

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mfs.Dir())

	// Wait for the file system to be unmounted.
	err = mfs.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	log.Println("Successfully exiting.")
}

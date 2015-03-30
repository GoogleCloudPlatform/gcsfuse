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
	"os/signal"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"golang.org/x/net/context"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [flags] <mount-point>\n", os.Args[0])
	flag.PrintDefaults()
}

var fBucketName = flag.String("bucket", "", "Name of GCS bucket to mount.")

var fImplicitDirs = flag.Bool(
	"implicit_dirs",
	false,
	"Implicitly define directories based on their content. See docs/semantics.md.")

func getBucketName() string {
	s := *fBucketName
	if s == "" {
		fmt.Println("You must set --bucket.")
		os.Exit(1)
	}

	return s
}

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

func main() {
	// Set up flags.
	flag.Usage = usage
	flag.Parse()

	// Grab the mount point.
	if flag.NArg() != 1 {
		usage()
		os.Exit(1)
	}

	mountPoint := flag.Arg(0)

	// Set up a GCS connection.
	log.Println("Initializing GCS connection.")
	conn, err := getConn()
	if err != nil {
		log.Fatal("Couldn't get GCS connection: ", err)
	}

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		Clock:               timeutil.RealClock(),
		Bucket:              conn.GetBucket(getBucketName()),
		ImplicitDirectories: *fImplicitDirs,
	}

	server, err := fs.NewServer(serverCfg)
	if err != nil {
		log.Fatal("fs.NewServer:", err)
	}

	// Mount the file system.
	mountedFS, err := fuse.Mount(mountPoint, server, &fuse.MountConfig{})
	if err != nil {
		log.Fatal("Mount:", err)
	}

	log.Println("File system has been successfully mounted.")

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mountedFS.Dir())

	// Wait for it to be unmounted.
	if err := mountedFS.Join(context.Background()); err != nil {
		log.Fatal("MountedFileSystem.Join:", err)
	}

	log.Println("Successfully exiting.")
}

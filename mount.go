// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s [flags] <mount-point>\n", os.Args[0])
	flag.PrintDefaults()
}

var fBucketName = flag.String("bucket", "", "Name of GCS bucket to mount.")

func getBucketName() string {
	s := *fBucketName
	if s == "" {
		fmt.Println("You must set -bucket.")
		os.Exit(1)
	}

	return s
}

func main() {
	// Set up flags.
	flag.Usage = usage
	flag.Parse()

	// Enable debugging, if requested.
	initDebugging()

	// Grab the mount point.
	if flag.NArg() != 1 {
		usage()
		os.Exit(1)
	}

	mountPoint := flag.Arg(0)

	// Set up a GCS authentication context.
	log.Println("Initializing GCS auth context.")
	authContext, err := getAuthContext()
	if err != nil {
		log.Fatal("Couldn't get GCS auth context: ", err)
	}

	// Open a FUSE connection.
	log.Println("Opening a FUSE connection.")
	c, err := fuse.Mount(mountPoint)
	if err != nil {
		log.Fatal("fuse.Mount: ", err)
	}

	defer c.Close()

	// Serve a file system on the connection.
	fileSystem := &fileSystem{
		authContext: authContext,
		bucketName:  getBucketName(),
	}

	log.Println("Beginning to serve FUSE connection.")
	if err := fs.Serve(c, fileSystem); err != nil {
		log.Fatal("fuse.Conn.Serve: ", err)
	}

	// Report any errors that occurred while mounting.
	log.Println("Waiting for FUSE shutdown.")
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal("Error mounting: ", err)
	}
}

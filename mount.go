// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jacobsa/gcsfuse/fs"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
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

	// Set up a GCS connection.
	log.Println("Initializing GCS connection.")
	conn, err := getConn()
	if err != nil {
		log.Fatal("Couldn't get GCS connection: ", err)
	}

	// Open a FUSE connection.
	log.Println("Opening a FUSE connection.")
	c, err := fuse.Mount(mountPoint)
	if err != nil {
		log.Fatal("fuse.Mount: ", err)
	}

	defer c.Close()

	// Create a file system.
	fileSystem, err := fs.NewFuseFS(conn.GetBucket(getBucketName()))
	if err != nil {
		log.Fatal("fs.NewFuseFS:", err)
	}

	// Serve the file system on the connection.
	log.Println("Beginning to serve FUSE connection.")
	if err := fusefs.Serve(c, fileSystem); err != nil {
		log.Fatal("fuse.Conn.Serve: ", err)
	}

	// Report any errors that occurred while mounting.
	log.Println("Waiting for FUSE shutdown.")
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal("Error mounting: ", err)
	}
}

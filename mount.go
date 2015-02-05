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
	authContext, err := getAuthContext()
	if err != nil {
		log.Fatal("Couldn't get GCS auth context: ", err)
	}

	// Open a FUSE connection.
	c, err := fuse.Mount(mountPoint)
	if err != nil {
		log.Fatal("fuse.Mount: ", err)
	}

	defer c.Close()

	// Serve a file system on the connection.
	fileSystem := &fileSystem{
		authContext: authContext,
	}

	if err := fs.Serve(c, fileSystem); err != nil {
		log.Fatal("fuse.Conn.Serve: ", err)
	}

	// Report any errors that occurred while mounting.
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal("Error mounting: ", err)
	}
}

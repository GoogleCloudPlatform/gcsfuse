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
	fmt.Fprintf(os.Stderr, "  %s <mount-point>\n", os.Args[0])
	flag.PrintDefaults()
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

	// Open a FUSE connection.
	c, err := fuse.Mount(mountPoint)
	if err != nil {
		log.Fatal("fuse.Mount: ", err)
	}

	defer c.Close()

	// Serve a file system on the connection.
	if err := fs.Serve(c, &fileSystem{}); err != nil {
		log.Fatal("fuse.Conn.Serve: ", err)
	}

	// Report any errors that occurred while mounting.
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal("Error mounting: ", err)
	}
}

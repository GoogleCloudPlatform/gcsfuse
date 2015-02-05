// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package main

import (
	"flag"
	"log"

	"bazil.org/fuse"
)

var enableDebugMessages = flag.Bool(
	"fuse_debug",
	false,
	"Write FUSE debugging messages to stderr.")

func logDebugMessage(msg interface{}) {
	log.Println("FUSE:", msg)
}

func initDebugging() {
	// If the debug flag has been set, log messages to stderr.
	if *enableDebugMessages {
		fuse.Debug = logDebugMessage
	}
}

// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var fEnableDebug = flag.Bool(
	"fuseutil.debug",
	false,
	"Write FUSE debugging messages to stderr.")

// Create a logger based on command-line flag settings.
func getLogger() *log.Logger {
	var writer io.Writer = ioutil.Discard
	if *fEnableDebug {
		writer = os.Stderr
	}

	return log.New(writer, "fuseutil: ", log.LstdFlags)
}

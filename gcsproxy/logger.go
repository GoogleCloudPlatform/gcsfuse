// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var fEnableDebug = flag.Bool(
	"gcsproxy.debug",
	false,
	"Write GCS proxy debugging messages to stderr.")

// Return a logger configured based on command-line flag settings.
func getLogger() *log.Logger {
	var writer io.Writer = ioutil.Discard
	if *fEnableDebug {
		writer = os.Stderr
	}

	return log.New(writer, "gcsproxy: ", log.LstdFlags)
}

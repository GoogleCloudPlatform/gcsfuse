// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fuseutil

import (
	"flag"
	"log"
)

var fEnableDebug = flag.Bool(
	"fuseutil.debug",
	false,
	"Write FUSE debugging messages to stderr.")

func logDebugMessage(msg interface{}) {
	log.Println("fuseutil:", msg)
}

func getDebugFunc() func(interface{}) {
	if *fEnableDebug {
		return logDebugMessage
	}

	return nil
}

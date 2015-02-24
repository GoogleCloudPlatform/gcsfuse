// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package fs

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	fusefs "bazil.org/fuse/fs"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcsfuse/timeutil"
)

var fEnableDebug = flag.Bool(
	"fs.debug",
	false,
	"Write gcsfuse/fs debugging messages to stderr.")

type fileSystem struct {
	logger *log.Logger
	clock  timeutil.Clock
	bucket gcs.Bucket
}

func (fs *fileSystem) Root() (fusefs.Node, error) {
	d := newDir(fs.logger, fs.clock, fs.bucket, "")
	return d, nil
}

func getLogger() *log.Logger {
	var writer io.Writer = ioutil.Discard
	if *fEnableDebug {
		writer = os.Stderr
	}

	return log.New(writer, "gcsfuse/fs: ", log.LstdFlags)
}

// Create a fuse file system whose root directory is the root of the supplied
// bucket. The supplied clock will be used for cache invalidation; it is *not*
// used for file modification times.
func NewFuseFS(clock timeutil.Clock, bucket gcs.Bucket) (fusefs.FS, error) {
	fs := &fileSystem{
		logger: getLogger(),
		clock:  clock,
		bucket: bucket,
	}

	return fs, nil
}

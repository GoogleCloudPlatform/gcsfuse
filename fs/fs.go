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
)

var fEnableDebug = flag.Bool(
	"fs.debug",
	false,
	"Write gcsfuse/fs debugging messages to stderr.")

type fileSystem struct {
	logger *log.Logger
	bucket gcs.Bucket
}

func (fs *fileSystem) Root() (fusefs.Node, error) {
	d := &dir{
		logger: fs.logger,
		bucket: fs.bucket,
	}

	return d, nil
}

func makeLogger() *log.Logger {
	var writer io.Writer = ioutil.Discard
	if *fEnableDebug {
		writer = os.Stderr
	}

	return log.New(writer, "gcsfuse/fs: ", log.LstdFlags)
}

// Create a fuse file system whose root directory is the root of the supplied
// bucket.
func NewFuseFS(bucket gcs.Bucket) (fusefs.FS, error) {
	fs := &fileSystem{
		logger: makeLogger(),
		bucket: bucket,
	}

	return fs, nil
}

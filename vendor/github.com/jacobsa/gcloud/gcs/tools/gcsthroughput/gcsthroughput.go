// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A tool to measure the upload throughput of GCS.
package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

var fBucket = flag.String("bucket", "", "Name of bucket.")
var fSize = flag.Int64("size", 1<<26, "Size of content to write.")
var fFile = flag.String("file", "", "If set, use pre-existing contents.")
var fRepeat = flag.Int("repeat", 1, "Repeat the content this many times.")

func createBucket() (bucket gcs.Bucket, err error) {
	// Create an authenticated HTTP client.
	tokenSrc, err := google.DefaultTokenSource(
		context.Background(),
		gcs.Scope_FullControl)

	if err != nil {
		err = fmt.Errorf("DefaultTokenSource: %v", err)
		return
	}

	// Use that to create a connection.
	connCfg := &gcs.ConnConfig{
		TokenSource: tokenSrc,
	}

	conn, err := gcs.NewConn(connCfg)
	if err != nil {
		err = fmt.Errorf("NewConn: %v", err)
		return
	}

	// Extract the bucket.
	if *fBucket == "" {
		err = errors.New("You must set --bucket.")
		return
	}

	bucket, err = conn.OpenBucket(context.Background(), *fBucket)
	if err != nil {
		err = fmt.Errorf("OpenBucket: %v", err)
		return
	}

	return
}

func getFile() (f *os.File, err error) {
	// Is there a pre-set file?
	if *fFile != "" {
		f, err = os.OpenFile(*fFile, os.O_RDONLY, 0)
		if err != nil {
			err = fmt.Errorf("OpenFile: %v", err)
			return
		}

		return
	}

	// Create a temporary file to hold random contents.
	f, err = fsutil.AnonymousFile("")
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %v", err)
		return
	}

	// Copy a bunch of random data into the file.
	log.Println("Reading random data.")
	_, err = io.Copy(f, io.LimitReader(rand.Reader, *fSize))
	if err != nil {
		err = fmt.Errorf("Copy: %v", err)
		return
	}

	// Seek back to the start for consumption.
	_, err = f.Seek(0, 0)
	if err != nil {
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	return
}

type repeatReader struct {
	f             *os.File
	N             int
	firstSeekDone bool
}

func (r *repeatReader) Read(p []byte) (n int, err error) {
	// On the very first read, seek the file to the start and cause N to be the
	// number of iterations left..
	if !r.firstSeekDone {
		if _, err = r.f.Seek(0, 0); err != nil {
			err = fmt.Errorf("Seek: %v", err)
			return
		}

		r.firstSeekDone = true

		if r.N <= 0 {
			err = io.EOF
			return
		}

		r.N--
	}

	// Read some content from the file.
	n, err = r.f.Read(p)

	// Ignore EOF if n != 0.
	if n != 0 && err == io.EOF {
		err = nil
	}

	// Otherwise, EOF errors cause a loop, if we're not yet done.
	if err == io.EOF {
		if r.N <= 0 {
			return
		}

		if _, err = r.f.Seek(0, 0); err != nil {
			err = fmt.Errorf("Seek: %v", err)
			return
		}

		r.N--
		return r.Read(p)
	}

	// Propagate other errors.
	if err != nil {
		return
	}

	// All is good.
	return
}

func run() (err error) {
	bucket, err := createBucket()
	if err != nil {
		err = fmt.Errorf("createBucket: %v", err)
		return
	}

	// Get an appropriate file.
	f, err := getFile()
	if err != nil {
		err = fmt.Errorf("getFile: %v", err)
		return
	}

	// Repeat the specified number of times.
	reader := &repeatReader{
		f: f,
		N: *fRepeat,
	}

	// Create an object using the repeated contents.
	log.Println("Creating object.")
	req := &gcs.CreateObjectRequest{
		Name:     "foo",
		Contents: reader,
	}

	before := time.Now()
	_, err = bucket.CreateObject(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	log.Printf("Wrote object in %v.", time.Since(before))

	return
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

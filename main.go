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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
	"golang.org/x/net/context"
)

var fBucketName = flag.String("bucket", "", "Name of GCS bucket to mount.")
var fMountPoint = flag.String("mount_point", "", "File system location.")
var fReadOnly = flag.Bool("read_only", false, "Mount in read-only mode.")

var fTempDir = flag.String(
	"temp_dir", "",
	"The temporary directory in which to store local copies of GCS objects. "+
		"If empty, the system default (probably /tmp) will be used.")

var fTempDirLimit = flag.Int64(
	"temp_dir_bytes", 1<<31,
	"A desired limit on the number of bytes used in --temp_dir. May be exceeded "+
		"for dirty files that have not been flushed or closed.")

var fGCSChunkSize = flag.Uint64(
	"gcs_chunk_size", 1<<24,
	"If set to a non-zero value N, split up GCS objects into multiple chunks of "+
		"size at most N when reading, and do not read or cache unnecessary chunks.")

var fImplicitDirs = flag.Bool(
	"implicit_dirs",
	false,
	"Implicitly define directories based on their content. See "+
		"docs/semantics.md.")

var fSupportNlink = flag.Bool(
	"support_nlink",
	false,
	"Return meaningful values for nlink from fstat(2). See docs/semantics.md.")

var fStatCacheTTL = flag.String(
	"stat_cache_ttl",
	"1m",
	"If non-empty, a duration specifying how long to cache StatObject results "+
		"from GCS, e.g. \"2s\" or \"15ms\". See docs/semantics.md for more.")

var fTypeCacheTTL = flag.String(
	"type_cache_ttl",
	"1m",
	"If non-empty, a duration specifying how long to cache name -> file/dir "+
		"type mappings in directory inodes, e.g. \"2s\" or \"15ms\". "+
		"See docs/semantics.md.")

func getBucketName() string {
	s := *fBucketName
	if s == "" {
		fmt.Println("You must set --bucket.")
		os.Exit(1)
	}

	return s
}

func registerSIGINTHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			log.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				log.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				log.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

func getBucket() (b gcs.Bucket) {
	// Set up a GCS connection.
	log.Println("Initializing GCS connection.")
	conn, err := getConn()
	if err != nil {
		log.Fatal("Couldn't get GCS connection: ", err)
	}

	// Extract the appropriate bucket.
	b = conn.GetBucket(getBucketName())

	// Enable cached StatObject results, if appropriate.
	if *fStatCacheTTL != "" {
		ttl, err := time.ParseDuration(*fStatCacheTTL)
		if err != nil {
			log.Fatalf("Invalid --stat_cache_ttl: %v", err)
			return
		}

		const cacheCapacity = 4096
		b = gcscaching.NewFastStatBucket(
			ttl,
			gcscaching.NewStatCache(cacheCapacity),
			timeutil.RealClock(),
			b)
	}

	return
}

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up flags.
	flag.Parse()

	// Check --mount_point.
	if *fMountPoint == "" {
		log.Fatalf("You must set --mount_point.")
	}

	mountPoint := *fMountPoint

	// Parse --type_cache_ttl
	var typeCacheTTL time.Duration
	if *fTypeCacheTTL != "" {
		var err error
		typeCacheTTL, err = time.ParseDuration(*fTypeCacheTTL)
		if err != nil {
			log.Fatalf("Invalid --type_cache_ttl: %v", err)
			return
		}
	}

	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if *fTempDir != "" {
		f, err := fsutil.AnonymousFile(*fTempDir)
		f.Close()

		if err != nil {
			log.Fatalf(
				"Error writing to temporary directory (%q); are you sure it exists "+
					"with the correct permissions?",
				err.Error())
		}
	}

	// Create a file system server.
	bucket := getBucket()
	serverCfg := &fs.ServerConfig{
		Clock:               timeutil.RealClock(),
		Bucket:              bucket,
		TempDir:             *fTempDir,
		TempDirLimit:        *fTempDirLimit,
		GCSChunkSize:        *fGCSChunkSize,
		ImplicitDirectories: *fImplicitDirs,
		SupportNlink:        *fSupportNlink,
		DirTypeCacheTTL:     typeCacheTTL,
	}

	server, err := fs.NewServer(serverCfg)
	if err != nil {
		log.Fatal("fs.NewServer:", err)
	}

	// Mount the file system.
	mountCfg := &fuse.MountConfig{
		FSName:   bucket.Name(),
		ReadOnly: *fReadOnly,
	}

	mountedFS, err := fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		log.Fatal("Mount:", err)
	}

	log.Println("File system has been successfully mounted.")

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mountedFS.Dir())

	// Wait for it to be unmounted.
	if err := mountedFS.Join(context.Background()); err != nil {
		log.Fatal("MountedFileSystem.Join:", err)
	}

	log.Println("Successfully exiting.")
}

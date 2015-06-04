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

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/perms"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////
// main function
////////////////////////////////////////////////////////////////////////

func run(bucketName string, mountPoint string) (err error) {
	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if *fTempDir != "" {
		var f *os.File
		f, err = fsutil.AnonymousFile(*fTempDir)
		f.Close()

		if err != nil {
			err = fmt.Errorf(
				"Error writing to temporary directory (%q); are you sure it exists "+
					"with the correct permissions?",
				err.Error())
			return
		}
	}

	// The file leaser used by the file system sizes its limit on number of
	// temporary files based on the process's rlimit. If this is too low, we'll
	// throw away cached content unnecessarily often. This is particularly a
	// problem on OS X, which has a crazy low default limit (256 as of OS X
	// 10.10.3). So print a warning if the limit is low.
	var rlimit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlimit); err == nil {
		const reasonableLimit = 4096

		if rlimit.Cur < reasonableLimit {
			log.Printf(
				"Warning: low file rlimit of %d will cause cached content to be "+
					"frequently evicted. Consider raising with `ulimit -n`.",
				rlimit.Cur)
		}
	}

	// Choose UID and GID.
	uid, gid, err := perms.MyUserAndGroup()
	if err != nil {
		err = fmt.Errorf("MyUserAndGroup: %v", err)
		return
	}

	if *fUid >= 0 {
		uid = uint32(*fUid)
	}

	if *fGid >= 0 {
		gid = uint32(*fGid)
	}

	// Create a file system server.
	bucket := getBucket(bucketName)
	serverCfg := &fs.ServerConfig{
		Clock:                timeutil.RealClock(),
		Bucket:               bucket,
		TempDir:              *fTempDir,
		TempDirLimitNumFiles: fs.ChooseTempDirLimitNumFiles(),
		TempDirLimitBytes:    *fTempDirLimit,
		GCSChunkSize:         *fGCSChunkSize,
		ImplicitDirectories:  *fImplicitDirs,
		DirTypeCacheTTL:      *fTypeCacheTTL,
		Uid:                  uid,
		Gid:                  gid,
		FilePerms:            os.FileMode(*fFileMode),
		DirPerms:             os.FileMode(*fDirMode),
	}

	server, err := fs.NewServer(serverCfg)
	if err != nil {
		err = fmt.Errorf("fs.NewServer: %v", err)
		return
	}

	// Mount the file system.
	mountCfg := &fuse.MountConfig{
		FSName:      bucket.Name(),
		Options:     fMountOptions,
		ErrorLogger: log.New(os.Stderr, "fuse: ", log.Flags()),
	}

	mountedFS, err := fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("Mount: %v", err)
		return
	}

	log.Println("File system has been successfully mounted.")

	// Let the user unmount with Ctrl-C (SIGINT).
	registerSIGINTHandler(mountedFS.Dir())

	// Wait for it to be unmounted.
	err = mountedFS.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	return
}

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up a custom usage function, then parse flags.
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [flags] bucket_name mount_point\n",
			os.Args[0])

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Help mode?
	if *fHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Extract positional arguments.
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	bucketName := args[0]
	mountPoint := args[1]

	// Run.
	err := run(bucketName, mountPoint)
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	log.Println("Successfully exiting.")
}

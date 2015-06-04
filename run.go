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

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/perms"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
)

// Set up a SIGINT handler that will invoke the supplied function for each
// SIGINT signal received. (Signals may be dropped while the handler is
// running.)
//
// Stop future calls to the handler by writing an element to the channel. Once
// the write proceeds, it is guaranteed that the handler will not be called
// again.
func registerSIGINTHandler(f func()) (stop chan<- struct{}) {
	panic("TODO")
}

// In main, set flagSet to flag.CommandLine and pass in os.Args[1:]. In a test,
// pass in a virgin flag set and test arguments.
func run(
	args []string,
	flagSet *flag.FlagSet,
	conn gcs.Conn,
	handleSIGINT func(mountPoint string)) (err error) {
	// Set up a custom usage function.
	flagSet.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [flags] bucket_name mount_point\n",
			os.Args[0])

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flagSet.PrintDefaults()
	}

	// Populate and parse flags.
	flags := populateFlagSet(flagSet)

	err = flagSet.Parse(args)
	if err != nil {
		err = fmt.Errorf("Parsing flags: %v", err)
		return
	}

	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if flags.TempDir != "" {
		var f *os.File
		f, err = fsutil.AnonymousFile(flags.TempDir)
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

	if flags.Uid >= 0 {
		uid = uint32(flags.Uid)
	}

	if flags.Gid >= 0 {
		gid = uint32(flags.Gid)
	}

	// Set up the bucket.
	bucket, err := setUpBucket(
		flags,
		conn,
		bucketName)

	if err != nil {
		err = fmt.Errorf("setUpBucket: %v", err)
		return
	}

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		Clock:                timeutil.RealClock(),
		Bucket:               bucket,
		TempDir:              flags.TempDir,
		TempDirLimitNumFiles: fs.ChooseTempDirLimitNumFiles(),
		TempDirLimitBytes:    flags.TempDirLimit,
		GCSChunkSize:         flags.GCSChunkSize,
		ImplicitDirectories:  flags.ImplicitDirs,
		DirTypeCacheTTL:      flags.TypeCacheTTL,
		Uid:                  uid,
		Gid:                  gid,
		FilePerms:            os.FileMode(flags.FileMode),
		DirPerms:             os.FileMode(flags.DirMode),
	}

	server, err := fs.NewServer(serverCfg)
	if err != nil {
		err = fmt.Errorf("fs.NewServer: %v", err)
		return
	}

	// Mount the file system.
	mountCfg := &fuse.MountConfig{
		FSName:      bucket.Name(),
		Options:     flags.MountOptions,
		ErrorLogger: log.New(os.Stderr, "fuse: ", log.Flags()),
	}

	mountedFS, err := fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("Mount: %v", err)
		return
	}

	log.Println("File system has been successfully mounted.")

	// Call the SIGINT handler as appropriate until this function returns.
	stopSIGINTHandler := registerSIGINTHandler(func() {
		handleSIGINT(mountPoint)
	})

	defer func() { stopSIGINTHandler <- struct{}{} }()

	// Wait for it to be unmounted.
	err = mountedFS.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	return
}

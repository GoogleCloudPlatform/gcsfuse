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
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/perms"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
)

// Mount the file system based on the supplied arguments, returning a
// fuse.MountedFileSystem that can be joined to wait for unmounting.
//
// In main, set flagSet to flag.CommandLine and pass in os.Args[1:]. In a test,
// pass in a virgin flag set and test arguments.
//
// Promises to pass on flag.ErrHelp from FlagSet.Parse.
func mount(
	args []string,
	flagSet *flag.FlagSet,
	conn gcs.Conn) (mfs *fuse.MountedFileSystem, err error) {
	// Populate and parse flags.
	flags := populateFlagSet(flagSet)

	err = flagSet.Parse(args)
	if err != nil {
		err = fmt.Errorf("Parsing flags: %v", err)
		return
	}

	// Help mode?
	if flags.Help {
		flagSet.Usage()
		return
	}

	// Extract positional arguments.
	if flagSet.NArg() != 2 {
		flagSet.Usage()
		err = errors.New("")
		return
	}

	bucketName := flagSet.Arg(0)
	mountPoint := flagSet.Arg(1)

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

		AppendThreshold: 1 << 21, // 2 MiB, a total guess.
		TmpObjectPrefix: ".gcsfuse_tmp/",
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

	mfs, err = fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("Mount: %v", err)
		return
	}

	return
}

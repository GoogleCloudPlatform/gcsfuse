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
	"time"

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"

	"github.com/googlecloudplatform/gcsfuse/fs"
	"github.com/googlecloudplatform/gcsfuse/perms"
	"github.com/googlecloudplatform/gcsfuse/ratelimit"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcscaching"
)

////////////////////////////////////////////////////////////////////////
// Wiring
////////////////////////////////////////////////////////////////////////

func setUpRateLimiting(
	in gcs.Bucket,
	opRateLimitHz float64,
	egressBandwidthLimit float64) (out gcs.Bucket, err error) {
	// If no rate limiting has been requested, just return the bucket.
	if !(opRateLimitHz > 0 || egressBandwidthLimit > 0) {
		out = in
		return
	}

	// Treat a disabled limit as a very large one.
	if !(opRateLimitHz > 0) {
		opRateLimitHz = 1e15
	}

	if !(egressBandwidthLimit > 0) {
		egressBandwidthLimit = 1e15
	}

	// Choose token bucket capacities.
	const window = 30 * time.Second

	opCapacity, err := ratelimit.ChooseTokenBucketCapacity(
		opRateLimitHz,
		window)

	if err != nil {
		err = fmt.Errorf("Choosing operation token bucket capacity: %v", err)
		return
	}

	egressCapacity, err := ratelimit.ChooseTokenBucketCapacity(
		egressBandwidthLimit,
		window)

	if err != nil {
		err = fmt.Errorf("Choosing egress bandwidth token bucket capacity: %v", err)
		return
	}

	// Create the throttles.
	opThrottle := ratelimit.NewThrottle(opRateLimitHz, opCapacity)
	egressThrottle := ratelimit.NewThrottle(egressBandwidthLimit, egressCapacity)

	// And the bucket.
	out = ratelimit.NewThrottledBucket(
		opThrottle,
		egressThrottle,
		in)

	return
}

func setUpBucket(
	flags *flagStorage,
	conn gcs.Conn,
	name string) (b gcs.Bucket, err error) {
	// Extract the appropriate bucket.
	b = conn.GetBucket(name)

	// Enable rate limiting, if requested.
	b, err = setUpRateLimiting(
		b,
		flags.OpRateLimitHz,
		flags.EgressBandwidthLimitBytesPerSecond)

	if err != nil {
		err = fmt.Errorf("setUpRateLimiting: %v", err)
		return
	}

	// Enable cached StatObject results, if appropriate.
	if flags.StatCacheTTL != 0 {
		const cacheCapacity = 4096
		b = gcscaching.NewFastStatBucket(
			flags.StatCacheTTL,
			gcscaching.NewStatCache(cacheCapacity),
			timeutil.RealClock(),
			b)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// run
////////////////////////////////////////////////////////////////////////

// In main, set flagSet to flag.CommandLine and pass in os.Args[1:]. In a test,
// pass in a virgin flag set and test arguments.
func run(
	args []string,
	flagSet *flag.FlagSet,
	handleSIGINT func(mountPoint string)) (err error) {
	// Populate the flag set and then parse flags.
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
	stopSIGINTHandler := registerSIGINTHandler(handleSIGINT)
	defer func() { stopSIGINTHandler <- struct{}{} }()

	// Wait for it to be unmounted.
	err = mountedFS.Join(context.Background())
	if err != nil {
		err = fmt.Errorf("MountedFileSystem.Join: %v", err)
		return
	}

	return
}

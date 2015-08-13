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
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/internal/perms"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/timeutil"
)

// Mount the file system based on the supplied arguments, returning a
// fuse.MountedFileSystem that can be joined to wait for unmounting.
func mount(
	ctx context.Context,
	bucketName string,
	mountPoint string,
	flags *flagStorage,
	conn gcs.Conn) (mfs *fuse.MountedFileSystem, err error) {
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
		ctx,
		flags,
		conn,
		bucketName)

	if err != nil {
		err = fmt.Errorf("setUpBucket: %v", err)
		return
	}

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		CacheClock:             timeutil.RealClock(),
		Bucket:                 bucket,
		TempDir:                flags.TempDir,
		ImplicitDirectories:    flags.ImplicitDirs,
		InodeAttributeCacheTTL: flags.StatCacheTTL,
		DirTypeCacheTTL:        flags.TypeCacheTTL,
		Uid:                    uid,
		Gid:                    gid,
		FilePerms:              os.FileMode(flags.FileMode),
		DirPerms:               os.FileMode(flags.DirMode),

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

	if flags.DebugFuse {
		mountCfg.DebugLogger = log.New(os.Stderr, "fuse_debug: ", 0)
	}

	mfs, err = fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("Mount: %v", err)
		return
	}

	return
}

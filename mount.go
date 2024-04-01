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
	"os"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/perms"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/timeutil"
)

// Mount the file system based on the supplied arguments, returning a
// fuse.MountedFileSystem that can be joined to wait for unmounting.
func mountWithStorageHandle(
	ctx context.Context,
	bucketName string,
	mountPoint string,
	flags *flagStorage,
	mountConfig *config.MountConfig,
	storageHandle storage.StorageHandle) (mfs *fuse.MountedFileSystem, err error) {
	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if flags.TempDir != "" {
		logger.Infof("Creating a temporary directory at %q\n", flags.TempDir)
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

	// Find the current process's UID and GID. If it was invoked as root and the
	// user hasn't explicitly overridden --uid, everything is going to be owned
	// by root. This is probably not what the user wants, so print a warning.
	uid, gid, err := perms.MyUserAndGroup()
	if err != nil {
		err = fmt.Errorf("MyUserAndGroup: %w", err)
		return
	}

	if uid == 0 && flags.Uid < 0 {
		fmt.Fprintln(os.Stdout, `
WARNING: gcsfuse invoked as root. This will cause all files to be owned by
root. If this is not what you intended, invoke gcsfuse as the user that will
be interacting with the file system.`)
	}

	// Choose UID and GID.
	if flags.Uid >= 0 {
		uid = uint32(flags.Uid)
	}

	if flags.Gid >= 0 {
		gid = uint32(flags.Gid)
	}

	metadataCacheTTL := mount.ResolveMetadataCacheTTL(flags.StatCacheTTL, flags.TypeCacheTTL, mountConfig.MetadataCacheConfig.TtlInSeconds)
	statCacheMaxSizeMB, err := mount.ResolveStatCacheMaxSizeMB(mountConfig.StatCacheMaxSizeMB, flags.StatCacheCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate StatCacheMaxSizeMB from stat-cache-ttl=%v, metadata-cache:stat-cache-max-size-mb=%v: %w", flags.StatCacheCapacity, mountConfig.StatCacheMaxSizeMB, err)
	}

	bucketCfg := gcsx.BucketConfig{
		BillingProject:                     flags.BillingProject,
		OnlyDir:                            flags.OnlyDir,
		EgressBandwidthLimitBytesPerSecond: flags.EgressBandwidthLimitBytesPerSecond,
		OpRateLimitHz:                      flags.OpRateLimitHz,
		StatCacheMaxSizeMB:                 statCacheMaxSizeMB,
		StatCacheTTL:                       metadataCacheTTL,
		EnableMonitoring:                   flags.StackdriverExportInterval > 0,
		AppendThreshold:                    1 << 21, // 2 MiB, a total guess.
		TmpObjectPrefix:                    ".gcsfuse_tmp/",
		DebugGCS:                           flags.DebugGCS,
	}
	bm := gcsx.NewBucketManager(bucketCfg, storageHandle)

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		CacheClock:                 timeutil.RealClock(),
		BucketManager:              bm,
		BucketName:                 bucketName,
		LocalFileCache:             flags.LocalFileCache,
		DebugFS:                    flags.DebugFS,
		TempDir:                    flags.TempDir,
		ImplicitDirectories:        flags.ImplicitDirs,
		InodeAttributeCacheTTL:     metadataCacheTTL,
		DirTypeCacheTTL:            metadataCacheTTL,
		Uid:                        uid,
		Gid:                        gid,
		FilePerms:                  os.FileMode(flags.FileMode),
		DirPerms:                   os.FileMode(flags.DirMode),
		RenameDirLimit:             flags.RenameDirLimit,
		SequentialReadSizeMb:       flags.SequentialReadSizeMb,
		EnableNonexistentTypeCache: flags.EnableNonexistentTypeCache,
		MountConfig:                mountConfig,
	}

	logger.Infof("Creating a new server...\n")
	server, err := fs.NewServer(ctx, serverCfg)
	if err != nil {
		err = fmt.Errorf("fs.NewServer: %w", err)
		return
	}

	fsName := bucketName
	if bucketName == "" || bucketName == "_" {
		// mounting all the buckets at once
		fsName = "gcsfuse"
	}

	// Mount the file system.
	logger.Infof("Mounting file system %q...", fsName)
	mountCfg := &fuse.MountConfig{
		FSName:     fsName,
		Subtype:    "gcsfuse",
		VolumeName: "gcsfuse",
		Options:    flags.MountOptions,
		// Allows parallel LookUpInode & ReadDir calls from Kernel's FUSE driver.
		// GCSFuse takes exclusive lock on directory inodes during ReadDir call,
		// hence there is no effect of parallelization of incoming ReadDir calls
		// from FUSE driver for user of GCSFuse. However, in case of LookUpInode
		// calls, GCSFuse takes read only lock during LookUpInode call which helps
		// users experience the performance gains. E.g. if a user workload tries to
		// access two files under same directory parallely, then the lookups also
		// happen parallely.
		EnableParallelDirOps: !(mountConfig.FileSystemConfig.DisableParallelDirops),
	}

	mountCfg.ErrorLogger = logger.NewLegacyLogger(logger.LevelError, "fuse: ")
	mountCfg.DebugLogger = logger.NewLegacyLogger(logger.LevelTrace, "fuse_debug: ")

	mfs, err = fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("Mount: %w", err)
		return
	}

	return
}

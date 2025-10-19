// Copyright 2024 Google LLC
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

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fuse"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/perms"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// Mount the file system based on the supplied arguments, returning a
// fuse.MountedFileSystem that can be joined to wait for unmounting.
func mountWithStorageHandle(
	ctx context.Context,
	bucketName string,
	mountPoint string,
	newConfig *cfg.Config,
	storageHandle storage.StorageHandle,
	metricHandle metrics.MetricHandle) (mfs *fuse.MountedFileSystem, err error) {
	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if newConfig.FileSystem.TempDir != "" {
		logger.Infof("Creating a temporary directory at %q\n", newConfig.FileSystem.TempDir)
		var f *os.File
		f, err = fsutil.AnonymousFile(string(newConfig.FileSystem.TempDir))
		f.Close()

		if err != nil {
			err = fmt.Errorf(
				"error writing to temporary directory (%q); are you sure it exists "+
					"with the correct permissions",
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

	if uid == 0 && newConfig.FileSystem.Uid < 0 {
		fmt.Fprintln(os.Stdout, `
WARNING: gcsfuse invoked as root. This will cause all files to be owned by
root. If this is not what you intended, invoke gcsfuse as the user that will
be interacting with the file system.`)
	}

	// Choose UID and GID.
	if newConfig.FileSystem.Uid >= 0 {
		uid = uint32(newConfig.FileSystem.Uid)
	}

	if newConfig.FileSystem.Gid >= 0 {
		gid = uint32(newConfig.FileSystem.Gid)
	}

	bucketCfg := gcsx.BucketConfig{
		BillingProject:                     newConfig.GcsConnection.BillingProject,
		OnlyDir:                            newConfig.OnlyDir,
		EgressBandwidthLimitBytesPerSecond: newConfig.GcsConnection.LimitBytesPerSec,
		OpRateLimitHz:                      newConfig.GcsConnection.LimitOpsPerSec,
		StatCacheMaxSizeMB:                 uint64(newConfig.MetadataCache.StatCacheMaxSizeMb),
		StatCacheTTL:                       time.Duration(newConfig.MetadataCache.TtlSecs) * time.Second,
		NegativeStatCacheTTL:               time.Duration(newConfig.MetadataCache.NegativeTtlSecs) * time.Second,
		EnableMonitoring:                   cfg.IsMetricsEnabled(&newConfig.Metrics),
		LogSeverity:                        newConfig.Logging.Severity,
		AppendThreshold:                    1 << 21, // 2 MiB, a total guess.
		ChunkTransferTimeoutSecs:           newConfig.GcsRetries.ChunkTransferTimeoutSecs,
		TmpObjectPrefix:                    ".gcsfuse_tmp/",
		EnableRapidAppends:                 newConfig.Write.EnableRapidAppends,
	}
	bm := gcsx.NewBucketManager(bucketCfg, storageHandle)

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		CacheClock:                 timeutil.RealClock(),
		BucketManager:              bm,
		BucketName:                 bucketName,
		LocalFileCache:             false,
		TempDir:                    string(newConfig.FileSystem.TempDir),
		ImplicitDirectories:        newConfig.ImplicitDirs,
		InodeAttributeCacheTTL:     time.Duration(newConfig.MetadataCache.TtlSecs) * time.Second,
		DirTypeCacheTTL:            time.Duration(newConfig.MetadataCache.TtlSecs) * time.Second,
		Uid:                        uid,
		Gid:                        gid,
		FilePerms:                  os.FileMode(newConfig.FileSystem.FileMode),
		DirPerms:                   os.FileMode(newConfig.FileSystem.DirMode),
		RenameDirLimit:             newConfig.FileSystem.RenameDirLimit,
		SequentialReadSizeMb:       int32(newConfig.GcsConnection.SequentialReadSizeMb),
		EnableNonexistentTypeCache: newConfig.MetadataCache.EnableNonexistentTypeCache,
		NewConfig:                  newConfig,
		MetricHandle:               metricHandle,
	}

	server, err := fuse.NewServer(newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create fuse server: %w", err)
	}
	mfs, err = server.Mount(ctx, mountPoint, fs.NewServer, serverCfg)
	if err != nil {
		err = fmt.Errorf("mount: %w", err)
		return
	}

	return
}

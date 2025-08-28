// Copyright 2023 Google LLC
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

//go:build !libfuse

package fuse

import (
	"context"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/mount"
	"github.com/jacobsa/fuse"
)

func NewServer(newConfig *cfg.Config) (Server, error) {
	return &jacobsaServer{
		newConfig: newConfig,
	}, nil
}

type jacobsaServer struct {
	newConfig *cfg.Config
}

func (s *jacobsaServer) Mount(
	ctx context.Context,
	mountPoint string,
	fsCreator func(context.Context, *fs.ServerConfig) (fuse.Server, error),
	serverCfg *fs.ServerConfig) (mfs *fuse.MountedFileSystem, err error) {
	if serverCfg.NewConfig.FileSystem.ExperimentalEnableDentryCache {
		serverCfg.Notifier = fuse.NewNotifier()
	}

	// Create a file system server.
	logger.Infof("Creating a new server...\n")
	server, err := fsCreator(ctx, serverCfg)
	if err != nil {
		err = fmt.Errorf("fs.NewServer: %w", err)
		return
	}

	// Mount the file system.
	fsName := serverCfg.BucketName
	if fsName == "" || fsName == "_" {
		// mounting all the buckets at once
		fsName = "gcsfuse"
	}
	logger.Infof("Mounting file system %q...", fsName)
	mountCfg := getFuseMountConfig(fsName, s.newConfig)
	mfs, err = fuse.Mount(mountPoint, server, mountCfg)
	if err != nil {
		err = fmt.Errorf("mount: %w", err)
		return
	}

	return
}

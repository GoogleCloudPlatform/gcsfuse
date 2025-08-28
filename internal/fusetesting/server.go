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

package fusetesting

import (
	"context"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fuse"
)

type Server struct {
	mfs *fuse.MountedFileSystem
}

func NewServer(
	ctx context.Context,
	fsCreator func(context.Context, *fs.ServerConfig) (fuse.Server, error),
	serverCfg *fs.ServerConfig) (*Server, error) {
	// Mount the file system.
	mfs, err := fuse.Mount(ctx, serverCfg.MountPoint, fsCreator, serverCfg)
	if err != nil {
		return nil, fmt.Errorf("fuse.Mount: %w", err)
	}
	return &Server{
		mfs: mfs,
	}, nil
}

func (s *Server) Unmount() error {
	return s.mfs.Unmount()
}

func (s *Server) Wait() error {
	return s.mfs.Join(context.Background())
}

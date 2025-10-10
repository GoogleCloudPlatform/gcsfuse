// Copyright 2015 Google LLC
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

package fs

import (
	"fmt"

	newcfg "github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// Server is a wrapper around a `fuse.Server` that also exposes the underlying
// file system.
type Server interface {
	// Serve serves the FUSE file system, until the connection is closed or an
	// error occurs.
	Serve(mfs *fuse.MountedFileSystem) error

	// PrepareToUnmount is a method that signals the file system to prepare for
	// unmounting. It cancels the file system's context, which should cause
	// new operations to be rejected.
	PrepareToUnmount()

	// MountedFS returns the associated `fuse.MountedFileSystem` instance.
	MountedFS() *fuse.MountedFileSystem

	// SetMountedFS sets the `fuse.MountedFileSystem` instance.
	SetMountedFS(mfs *fuse.MountedFileSystem)

	// FUSE-related methods to satisfy the fuse.Server interface.
	fuse.Server
}

type server struct {
	fs  *fileSystem
	mfs *fuse.MountedFileSystem
	// This context is for the FUSE server itself, used to signal shutdown.
	cancel context.CancelFunc
	fuse.Server
}

func (s *server) Serve(mfs *fuse.MountedFileSystem) error {
	// The actual FUSE serving is done by fuse.Mount. This method just waits for it to unmount.
	return mfs.Join(context.Background())
}

func (s *server) PrepareToUnmount() {
	s.cancel()
}

func (s *server) MountedFS() *fuse.MountedFileSystem {
	return s.mfs
}

func (s *server) SetMountedFS(mfs *fuse.MountedFileSystem) {
	s.mfs = mfs
}

// NewServer creates a file system server according to the supplied configuration.
func NewServer(ctx context.Context, cfg *ServerConfig) (Server, error) {
	// This context will be cancelled by PrepareToUnmount.
	// The context passed to NewFileSystem is used for internal operations and to reject new requests.
	// The context for the FUSE server's Join method is passed to the Serve method.
	fsCtx, cancel := context.WithCancel(ctx)

	fs, err := NewFileSystem(fsCtx, cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create file system: %w", err)
	}

	wrappedFS := wrappers.WithErrorMapping(fs, cfg.NewConfig.FileSystem.PreconditionErrors) // fs is *fileSystem
	if newcfg.IsTracingEnabled(cfg.NewConfig) {
		wrappedFS = wrappers.WithTracing(wrappedFS)
	}
	wrappedFS = wrappers.WithMonitoring(wrappedFS, cfg.MetricHandle)

	var fuseServer fuse.Server
	if cfg.Notifier != nil {
		fuseServer = fuse.NewServerWithNotifier(cfg.Notifier, fuseutil.NewFileSystemServer(wrappedFS))
	} else {
		fuseServer = fuseutil.NewFileSystemServer(wrappedFS)
	}
	return &server{fs: fs, cancel: cancel, Server: fuseServer}, nil
}
